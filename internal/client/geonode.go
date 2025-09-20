package client

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"

    "proxy-system/internal/models"
)

type GeoNodeClient struct {
    httpClient *http.Client
    baseURL    string
}

func NewGeoNodeClient() *GeoNodeClient {
    return &GeoNodeClient{
        httpClient: &http.Client{
            Timeout: 15 * time.Second,
        },
        baseURL: "https://proxylist.geonode.com/api/proxy-list",
    }
}

func (c *GeoNodeClient) FetchProxies(limit int) ([]models.ProxyData, error) {
    url := fmt.Sprintf("%s?protocols=socks5%%2Csocks4&filterLastChecked=10&speed=fast&limit=%d&page=1&sort_by=lastChecked&sort_type=desc", 
        c.baseURL, limit)

    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %v", err)
    }

    req.Header.Set("User-Agent", "Proxy-System/1.0")
    req.Header.Set("Accept", "application/json")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch proxies: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("API returned status code: %d", resp.StatusCode)
    }

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read response body: %v", err)
    }

    // Find the start of the JSON object
    jsonStart := -1
    for i, b := range body {
        if b == '{' {
            jsonStart = i
            break
        }
    }
    if jsonStart == -1 {
        return nil, fmt.Errorf("no JSON object found in response")
    }

    var proxyResponse models.ProxyResponse
    if err := json.Unmarshal(body[jsonStart:], &proxyResponse); err != nil {
        return nil, fmt.Errorf("failed to unmarshal response: %v", err)
    }

    return proxyResponse.Data, nil
}