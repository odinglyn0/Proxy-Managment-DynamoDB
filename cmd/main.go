package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    "proxy-system/internal/config"
    "proxy-system/internal/service"
)

func main() {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("Failed to load configuration: %v", err)
    }

    log.Printf("Starting PMS")

    // Initialize service
    proxyService, err := service.NewProxyService(cfg)
    if err != nil {
        log.Fatalf("Failed to initialize proxy service: %v", err)
    }

    // Create context for graceful shutdown
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Start the service
    go func() {
        if err := proxyService.Start(ctx); err != nil {
            log.Fatalf("Proxy service error: %v", err)
        }
    }()

    // Wait for interrupt signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan

    log.Println("Shutting down proxy management system...")
    cancel()
    
    // Give some time for cleanup
    time.Sleep(2 * time.Second)
    log.Println("Proxy management system stopped")
}