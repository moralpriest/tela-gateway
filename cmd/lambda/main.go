package main

import (
	"net/http"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"

	"github.com/moralpriest/tela-gateway"
)

// Pure Go Lambda entrypoint (no Docker).
// Runtime: provided.al2023 — binary must be named "bootstrap".
// Function URL uses API Gateway HTTP API v2 payloads.
func main() {
	adapter := httpadapter.NewV2(http.HandlerFunc(gateway.ServeTELA))
	lambda.Start(adapter.ProxyWithContext)
}
