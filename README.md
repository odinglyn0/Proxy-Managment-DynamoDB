# Proxy Management System

A Go application that fetches proxy data from GeoNode API and stores it in DynamoDB for easy management.

## Requirements

- Go 1.21 or later
- AWS credentials configured for DynamoDB access

## Installation

1. Clone the repository
2. Run `go mod tidy` to install dependencies

## Build

### Native Build

```bash
go build -o proxies cmd/main.go
```

### Docker Build

```bash
docker build -t proxies .
```

## Usage

### Run Natively

Run the application:

```bash
go run cmd/main.go
```

Or run the built binary:

```bash
./proxies
```

### Run with Docker

```bash
docker run --env-file .env proxies
```

### Deploy to AWS Fargate

The Docker image is optimized for deployment to AWS Fargate on EKS/ECS. Use the built image with appropriate environment variables and task definition.

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