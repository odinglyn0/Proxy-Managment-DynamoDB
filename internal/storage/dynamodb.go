package storage

import (
    "fmt"
    "log"
    "time"

    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/credentials"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/dynamodb"
    "github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"

    "proxy-system/internal/config"
    "proxy-system/internal/models"
)

type DynamoDBStorage struct {
    client    *dynamodb.DynamoDB
    tableName string
}

func NewDynamoDBStorage(cfg *config.Config) (*DynamoDBStorage, error) {
    sess, err := session.NewSession(&aws.Config{
        Region: aws.String(cfg.AWSRegion),
        Credentials: credentials.NewStaticCredentials(
            cfg.AWSAccessKeyID,
            cfg.AWSSecretAccessKey,
            "",
        ),
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create AWS session: %v", err)
    }

    client := dynamodb.New(sess)
    storage := &DynamoDBStorage{
        client:    client,
        tableName: cfg.DynamoDBTableName,
    }

    // Ensure table exists
    if err := storage.ensureTableExists(); err != nil {
        return nil, fmt.Errorf("failed to ensure table exists: %v", err)
    }

    return storage, nil
}

func (s *DynamoDBStorage) ensureTableExists() error {
    // Check if table exists
    _, err := s.client.DescribeTable(&dynamodb.DescribeTableInput{
        TableName: aws.String(s.tableName),
    })
    
    if err == nil {
        log.Printf("Table %s already exists", s.tableName)
        return nil
    }

    // Create table if it doesn't exist
    log.Printf("Creating table: %s", s.tableName)
    
    input := &dynamodb.CreateTableInput{
        TableName: aws.String(s.tableName),
        KeySchema: []*dynamodb.KeySchemaElement{
            {
                AttributeName: aws.String("proxy_key"),
                KeyType:       aws.String("HASH"),
            },
        },
        AttributeDefinitions: []*dynamodb.AttributeDefinition{
            {
                AttributeName: aws.String("proxy_key"),
                AttributeType: aws.String("S"),
            },
        },
        BillingMode: aws.String("PAY_PER_REQUEST"),
    }

    _, err = s.client.CreateTable(input)
    if err != nil {
        return fmt.Errorf("failed to create table: %v", err)
    }

    // Wait for table to be active
    log.Printf("Waiting for table %s to be active...", s.tableName)
    err = s.client.WaitUntilTableExists(&dynamodb.DescribeTableInput{
        TableName: aws.String(s.tableName),
    })
    if err != nil {
        return fmt.Errorf("failed to wait for table creation: %v", err)
    }

    log.Printf("Table %s created successfully", s.tableName)
    return nil
}

func (s *DynamoDBStorage) BatchGetProxies(proxyKeys []string) (map[string]*models.ProxyData, error) {
    const batchSize = 100
    result := make(map[string]*models.ProxyData)
    
    for i := 0; i < len(proxyKeys); i += batchSize {
        end := i + batchSize
        if end > len(proxyKeys) {
            end = len(proxyKeys)
        }
        
        keys := make([]map[string]*dynamodb.AttributeValue, end-i)
        for j, key := range proxyKeys[i:end] {
            keys[j] = map[string]*dynamodb.AttributeValue{
                "proxy_key": {S: aws.String(key)},
            }
        }
        
        output, err := s.client.BatchGetItem(&dynamodb.BatchGetItemInput{
            RequestItems: map[string]*dynamodb.KeysAndAttributes{
                s.tableName: {Keys: keys},
            },
        })
        if err != nil {
            return nil, err
        }
        
        if items, ok := output.Responses[s.tableName]; ok {
            for _, item := range items {
                var proxy models.ProxyData
                if err := dynamodbattribute.UnmarshalMap(item, &proxy); err == nil {
                    result[proxy.GetKey()] = &proxy
                }
            }
        }
    }
    
    return result, nil
}

func (s *DynamoDBStorage) UpsertProxy(proxy *models.ProxyData) error {
    // Set proxy key and updated timestamp
    proxyKey := proxy.GetKey()
    now := time.Now()
    proxy.UpdatedAt = now

    // Marshal the proxy data
    item := map[string]*dynamodb.AttributeValue{
        "proxy_key": {S: aws.String(proxyKey)},
    }

    // Add all proxy fields
    proxyMap := map[string]interface{}{
        "id":                     proxy.ID,
        "ip":                     proxy.IP,
        "port":                   proxy.Port,
        "anonymity":              proxy.Anonymity,
        "asn":                    proxy.ASN,
        "city":                   proxy.City,
        "country":                proxy.Country,
        "created_at":             proxy.CreatedAt.Unix(),
        "google":                 proxy.Google,
        "isp":                    proxy.ISP,
        "last_checked":           proxy.LastChecked.Unix(),
        "latency":                proxy.Latency,
        "org":                    proxy.Org,
        "protocols":              proxy.Protocols,
        "region":                 proxy.Region,
        "response_time":          proxy.ResponseTime,
        "speed":                  proxy.Speed,
        "up_time":                proxy.UpTime,
        "up_time_success_count":  proxy.UpTimeSuccessCount,
        "up_time_try_count":      proxy.UpTimeTryCount,
        "updated_at":             now.Unix(),
    }

    av, err := dynamodbattribute.MarshalMap(proxyMap)
    if err != nil {
        return fmt.Errorf("failed to marshal proxy data: %v", err)
    }

    // Merge with item
    for k, v := range av {
        item[k] = v
    }

    _, err = s.client.PutItem(&dynamodb.PutItemInput{
        TableName: aws.String(s.tableName),
        Item:      item,
    })
    if err != nil {
        return fmt.Errorf("failed to upsert proxy: %v", err)
    }

    return nil
}

func (s *DynamoDBStorage) BatchUpsertProxies(proxies []models.ProxyData) error {
    const batchSize = 25 // DynamoDB batch write limit

    for i := 0; i < len(proxies); i += batchSize {
        end := i + batchSize
        if end > len(proxies) {
            end = len(proxies)
        }

        batch := proxies[i:end]
        if err := s.writeBatch(batch); err != nil {
            return fmt.Errorf("failed to write batch: %v", err)
        }
    }

    return nil
}

func (s *DynamoDBStorage) writeBatch(proxies []models.ProxyData) error {
    writeRequests := make([]*dynamodb.WriteRequest, 0, len(proxies))
    now := time.Now()

    for _, proxy := range proxies {
        proxy.UpdatedAt = now
        proxyKey := proxy.GetKey()

        proxyMap := map[string]interface{}{
            "proxy_key":              proxyKey,
            "id":                     proxy.ID,
            "ip":                     proxy.IP,
            "port":                   proxy.Port,
            "anonymity":              proxy.Anonymity,
            "asn":                    proxy.ASN,
            "city":                   proxy.City,
            "country":                proxy.Country,
            "created_at":             proxy.CreatedAt.Unix(),
            "google":                 proxy.Google,
            "isp":                    proxy.ISP,
            "last_checked":           proxy.LastChecked.Unix(),
            "latency":                proxy.Latency,
            "org":                    proxy.Org,
            "protocols":              proxy.Protocols,
            "region":                 proxy.Region,
            "response_time":          proxy.ResponseTime,
            "speed":                  proxy.Speed,
            "up_time":                proxy.UpTime,
            "up_time_success_count":  proxy.UpTimeSuccessCount,
            "up_time_try_count":      proxy.UpTimeTryCount,
            "updated_at":             now.Unix(),
        }

        item, err := dynamodbattribute.MarshalMap(proxyMap)
        if err != nil {
            return fmt.Errorf("failed to marshal proxy: %v", err)
        }

        writeRequests = append(writeRequests, &dynamodb.WriteRequest{
            PutRequest: &dynamodb.PutRequest{
                Item: item,
            },
        })
    }

    _, err := s.client.BatchWriteItem(&dynamodb.BatchWriteItemInput{
        RequestItems: map[string][]*dynamodb.WriteRequest{
            s.tableName: writeRequests,
        },
    })

    return err
}
