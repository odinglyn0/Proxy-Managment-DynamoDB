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
    url := fmt.Sprintf("%s?protocols=http%%2Chttps%%2Csocks4%%2Csocks5&limit=%d&page=1&sort_by=lastChecked&sort_type=desc",
        c.baseURL, limit)

    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %v", err)
    }

    req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36")
    req.Header.Set("Accept-Language", "en-GB,en-US;q=0.9,en;q=0.8")
    req.Header.Set("Cache-Control", "no-cache")
    req.Header.Set("Pragma", "no-cache")
    req.Header.Set("Priority", "u=0, i")
    req.Header.Set("Sec-CH-UA", `"Chromium";v="140", "Not=A?Brand";v="24", "Google Chrome";v="140"`)
    req.Header.Set("Sec-CH-UA-Mobile", "?0")
    req.Header.Set("Sec-CH-UA-Platform", `"Windows"`)
    req.Header.Set("Sec-Fetch-Dest", "document")
    req.Header.Set("Sec-Fetch-Mode", "navigate")
    req.Header.Set("Sec-Fetch-Site", "none")
    req.Header.Set("Sec-Fetch-User", "?1")
    req.Header.Set("Upgrade-Insecure-Requests", "1")


    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch proxies: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("API returned status code: %d, body: %s", resp.StatusCode, string(body))
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
        return nil, fmt.Errorf("failed to unmarshal response: %v, body: %s", err, string(body[jsonStart:]))
    }

    return proxyResponse.Data, nil
}