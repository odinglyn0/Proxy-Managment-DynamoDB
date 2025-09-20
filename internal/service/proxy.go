package service

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "strings"
    "time"

    "golang.org/x/net/proxy"
    "proxy-system/internal/client"
    "proxy-system/internal/config"
    "proxy-system/internal/models"
    "proxy-system/internal/storage"
)

type ProxyService struct {
    client  *client.GeoNodeClient
    storage *storage.DynamoDBStorage
    config  *config.Config
}

func NewProxyService(cfg *config.Config) (*ProxyService, error) {
    geoNodeClient := client.NewGeoNodeClient()
    
    dynamoStorage, err := storage.NewDynamoDBStorage(cfg)
    if err != nil {
        return nil, err
    }

    return &ProxyService{
        client:  geoNodeClient,
        storage: dynamoStorage,
        config:  cfg,
    }, nil
}

func (s *ProxyService) Start(ctx context.Context) error {
    log.Println("Proxy service started")

    // Initial fetch
    successful, err := s.updateProxies()
    if err != nil {
        log.Printf("Initial proxy update failed: %v", err)
    } else if successful {
        log.Println("Initial update successful, scheduling next in 1 minute")
    } else {
        log.Println("Initial update had no changes, not scheduling next")
    }

    var timer *time.Timer
    if err == nil && successful {
        timer = time.NewTimer(s.config.UpdateInterval)
    }

    for {
        if timer == nil {
            select {
            case <-ctx.Done():
                log.Println("Proxy service stopping...")
                return ctx.Err()
            }
        } else {
            select {
            case <-ctx.Done():
                log.Println("Proxy service stopping...")
                timer.Stop()
                return ctx.Err()
            case <-timer.C:
                successful, err := s.updateProxies()
                if err != nil {
                    log.Printf("Failed to update proxies: %v", err)
                } else if successful {
                    log.Printf("Update successful, scheduling next in 1 minute")
                    timer.Reset(s.config.UpdateInterval)
                } else {
                    log.Println("Update had no changes, not scheduling next")
                    timer.Stop()
                    timer = nil
                }
            }
        }
    }
}

func (s *ProxyService) updateProxies() (bool, error) {
    log.Printf("Fetching proxies from API (limit: %d)...", s.config.ProxyLimit)

    var proxies []models.ProxyData
    var err error
    maxRetries := 3
    retryDelay := 3 * time.Second

    for attempt := 1; attempt <= maxRetries; attempt++ {
        proxies, err = s.client.FetchProxies(s.config.ProxyLimit)
        if err == nil {
            break
        }

        if attempt < maxRetries {
            log.Printf("Failed to fetch proxies (attempt %d/%d): %v. Retrying in %v...",
                attempt, maxRetries, err, retryDelay)
            time.Sleep(retryDelay)
        } else {
            log.Printf("Failed to fetch proxies after %d attempts: %v", maxRetries, err)
            return false, err
        }
    }

    log.Printf("Fetched %d proxies from API", len(proxies))

    // Check which proxies need updates
    var toUpdate []models.ProxyData
    var newProxies, updatedProxies int

    // Collect all proxy keys
    proxyKeys := make([]string, len(proxies))
    for i, proxy := range proxies {
        proxyKeys[i] = proxy.GetKey()
    }

    // Batch get existing proxies
    existingProxies, err := s.storage.BatchGetProxies(proxyKeys)
    if err != nil {
        return false, fmt.Errorf("failed to batch get proxies: %v", err)
    }

    // Validate proxies concurrently
    type validationResult struct {
        proxy   models.ProxyData
        valid   bool
        index   int
    }

    validationChan := make(chan validationResult, len(proxies))
    semaphore := make(chan struct{}, 500) // Limit to 10 concurrent validations

    for i, proxy := range proxies {
        go func(p models.ProxyData, idx int) {
            semaphore <- struct{}{} // Acquire
            defer func() { <-semaphore }() // Release

            valid := s.validateProxy(&p)
            validationChan <- validationResult{proxy: p, valid: valid, index: idx}
        }(proxy, i)
    }

    // Collect validation results
    validatedProxies := make([]models.ProxyData, 0, len(proxies))
    for i := 0; i < len(proxies); i++ {
        result := <-validationChan
        if result.valid {
            validatedProxies = append(validatedProxies, result.proxy)
        } else {
            log.Printf("Skipping invalid proxy: %s", result.proxy.GetKey())
        }
    }

    // Process validated proxies
    for _, proxy := range validatedProxies {
        proxyKey := proxy.GetKey()

        existingProxy := existingProxies[proxyKey]
        if existingProxy == nil {
            // New proxy
            toUpdate = append(toUpdate, proxy)
            newProxies++
        } else if s.hasProxyChanged(existingProxy, &proxy) {
            // Proxy has changed
            toUpdate = append(toUpdate, proxy)
            updatedProxies++
        }
    }

    if len(toUpdate) > 0 {
        log.Printf("Updating %d proxies (%d new, %d updated)", len(toUpdate), newProxies, updatedProxies)

        if err := s.storage.BatchUpsertProxies(toUpdate); err != nil {
            return false, err
        }

        log.Printf("Successfully updated %d proxies", len(toUpdate))
        return true, nil
    } else {
        log.Println("No proxy updates needed")
        return false, nil
    }
}

func (s *ProxyService) validateProxy(p *models.ProxyData) bool {
    // Only validate SOCKS4 and SOCKS5 proxies
    hasSocks := false
    for _, protocol := range p.Protocols {
        if protocol == "socks4" || protocol == "socks5" {
            hasSocks = true
            break
        }
    }

    if !hasSocks {
        return true // Skip validation for non-SOCKS proxies
    }

    proxyAddr := fmt.Sprintf("%s:%s", p.IP, p.Port)

    for _, protocol := range p.Protocols {
        if protocol != "socks4" && protocol != "socks5" {
            continue
        }

        var dialer proxy.Dialer
        var err error

        if protocol == "socks5" {
            dialer, err = proxy.SOCKS5("tcp", proxyAddr, nil, nil)
        } else if protocol == "socks4" {
            dialer, err = proxy.SOCKS5("tcp", proxyAddr, nil, nil) // SOCKS4 is handled by SOCKS5 dialer in most cases
        }

        if err != nil {
            log.Printf("Failed to create %s dialer for %s: %v", strings.ToUpper(protocol), proxyAddr, err)
            continue
        }

        // Create HTTP client with proxy
        httpClient := &http.Client{
            Transport: &http.Transport{
                Dial: dialer.Dial,
            },
            Timeout: 10 * time.Second,
        }

        // Test with a simple HTTP request
        testURL := "http://httpbin.org/ip"
        resp, err := httpClient.Get(testURL)
        if err != nil {
            log.Printf("Failed to validate %s proxy %s: %v", strings.ToUpper(protocol), proxyAddr, err)
            continue
        }
        resp.Body.Close()

        if resp.StatusCode == 200 {
            log.Printf("Successfully validated %s proxy %s", strings.ToUpper(protocol), proxyAddr)
            return true
        } else {
            log.Printf("Proxy %s returned status %d", proxyAddr, resp.StatusCode)
        }
    }

    return false
}

func (s *ProxyService) hasProxyChanged(existing, new *models.ProxyData) bool {
    return existing.LastChecked != new.LastChecked ||
            existing.ResponseTime != new.ResponseTime ||
            existing.UpTime != new.UpTime ||
            existing.UpTimeSuccessCount != new.UpTimeSuccessCount ||
            existing.Speed != new.Speed ||
            existing.Anonymity != new.Anonymity ||
            len(existing.Protocols) != len(new.Protocols)
}