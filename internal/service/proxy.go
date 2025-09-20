package service

import (
    "context"
    "fmt"
    "log"
    "time"

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

    proxies, err := s.client.FetchProxies(s.config.ProxyLimit)
    if err != nil {
        return false, err
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

    for _, proxy := range proxies {
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

func (s *ProxyService) hasProxyChanged(existing, new *models.ProxyData) bool {
    return existing.LastChecked != new.LastChecked ||
           existing.ResponseTime != new.ResponseTime ||
           existing.UpTime != new.UpTime ||
           existing.UpTimeSuccessCount != new.UpTimeSuccessCount ||
           existing.Speed != new.Speed ||
           existing.Anonymity != new.Anonymity ||
           len(existing.Protocols) != len(new.Protocols)
}