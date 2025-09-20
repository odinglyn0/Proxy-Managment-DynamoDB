# Proxy Management System

A Go application that fetches proxy data from GeoNode API and stores it in DynamoDB for easy management.

## Requirements

- Go 1.21 or later
- AWS credentials configured for DynamoDB access

## Installation

1. Clone the repository
2. Run `go mod tidy` to install dependencies

## Build

```bash
go build -o proxies.exe cmd/main.go
```

## Usage

Run the application:

```bash
go run cmd/main.go
```

Or run the built binary:

```bash
./proxies.exe
```

The service will start fetching and updating proxies periodically based on configuration.

## Configuration

Set the following environment variables:

- `AWS_ACCESS_KEY_ID` (required): AWS access key ID
- `AWS_SECRET_ACCESS_KEY` (required): AWS secret access key
- `DYNAMODB_TABLE_NAME` (required): DynamoDB table name for storing proxies
- `AWS_REGION` (optional): AWS region (defaults to eu-west-1)
- `PROXY_LIMIT` (optional): Maximum number of proxies to fetch (defaults to 500)

## License

Apache 2.0