package models

import (
    "encoding/json"
    "time"
)

type ProxyResponse struct {
    Data []ProxyData `json:"data"`
    Total int         `json:"total"`
    Page  int         `json:"page"`
    Limit int         `json:"limit"`
}

type ProxyData struct {
    ID                string    `json:"_id" dynamodb:"id"`
    IP                string    `json:"ip" dynamodb:"ip"`
    Port              string    `json:"port" dynamodb:"port"`
    Anonymity         string    `json:"anonymityLevel" dynamodb:"anonymity"`
    ASN               string    `json:"asn" dynamodb:"asn"`
    City              string    `json:"city" dynamodb:"city"`
    Country           string    `json:"country" dynamodb:"country"`
    CreatedAt         time.Time `json:"created_at" dynamodb:"created_at" dynamodbav:",unixtime"`
    Google            bool      `json:"google" dynamodb:"google"`
    ISP               string    `json:"isp" dynamodb:"isp"`
    LastChecked       time.Time `json:"lastChecked" dynamodb:"last_checked" dynamodbav:",unixtime"`
    Latency           float64   `json:"latency" dynamodb:"latency"`
    Org               string    `json:"org" dynamodb:"org"`
    Protocols         []string  `json:"protocols" dynamodb:"protocols"`
    Region            *string   `json:"region" dynamodb:"region"`
    ResponseTime      int       `json:"responseTime" dynamodb:"response_time"`
    Speed             int       `json:"speed" dynamodb:"speed"`
    UpdatedAt         time.Time `json:"updated_at" dynamodb:"updated_at" dynamodbav:",unixtime"`
    WorkingPercent    *float64  `json:"workingPercent" dynamodb:"working_percent"`
    UpTime            float64   `json:"upTime" dynamodb:"up_time"`
    UpTimeSuccessCount int      `json:"upTimeSuccessCount" dynamodb:"up_time_success_count"`
    UpTimeTryCount    int       `json:"upTimeTryCount" dynamodb:"up_time_try_count"`
}

// UnmarshalJSON custom unmarshaler to handle LastChecked as Unix timestamp
func (p *ProxyData) UnmarshalJSON(data []byte) error {
    type Alias ProxyData
    aux := &struct {
        LastChecked int64 `json:"lastChecked"`
        *Alias
    }{
        Alias: (*Alias)(p),
    }
    if err := json.Unmarshal(data, aux); err != nil {
        return err
    }
    p.LastChecked = time.Unix(aux.LastChecked, 0)
    return nil
}

// GetKey returns the primary key for DynamoDB
func (p *ProxyData) GetKey() string {
    return p.IP + ":" + p.Port
}