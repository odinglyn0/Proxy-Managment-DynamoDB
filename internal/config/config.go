package config

import (
    "fmt"
    "os"
    "strconv"
    "time"
)

type Config struct {
    AWSAccessKeyID     string
    AWSSecretAccessKey string
    AWSRegion          string
    DynamoDBTableName  string
    ProxyLimit         int
    UpdateInterval     time.Duration
}

func Load() (*Config, error) {
    cfg := &Config{
        UpdateInterval: time.Minute, // Default 1 minute
    }

    // AWS Configuration
    cfg.AWSAccessKeyID = os.Getenv("AWS_ACCESS_KEY_ID")
    if cfg.AWSAccessKeyID == "" {
        return nil, fmt.Errorf("AWS_ACCESS_KEY_ID is required")
    }

    cfg.AWSSecretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
    if cfg.AWSSecretAccessKey == "" {
        return nil, fmt.Errorf("AWS_SECRET_ACCESS_KEY is required")
    }

    cfg.AWSRegion = os.Getenv("AWS_REGION")
    if cfg.AWSRegion == "" {
        cfg.AWSRegion = "eu-west-1" // Europe (Ireland) default region
    }

    cfg.DynamoDBTableName = os.Getenv("DYNAMODB_TABLE_NAME")
    if cfg.DynamoDBTableName == "" {
        return nil, fmt.Errorf("DYNAMODB_TABLE_NAME is required")
    }

    // Proxy Limit
    limitStr := os.Getenv("PROXY_LIMIT")
    if limitStr == "" {
        cfg.ProxyLimit = 500 // Default limit
    } else {
        limit, err := strconv.Atoi(limitStr)
        if err != nil {
            return nil, fmt.Errorf("invalid PROXY_LIMIT: %v", err)
        }
        cfg.ProxyLimit = limit
    }

    return cfg, nil
}