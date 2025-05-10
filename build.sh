#!/bin/bash

# Create swagger directory if it doesn't exist
# This directory is used to store the generated Swagger/OpenAPI documentation
if [ ! -d "api/swagger" ]; then
    mkdir -p api/swagger
fi

# Generate Go code from protobuf definitions and gRPC-Gateway code for apiserver and spikeworker
protoc -I api -I google --go_out=api --go_opt=paths=source_relative --go-grpc_out=api --go-grpc_opt=paths=source_relative api/apiserver.proto
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative pkg/worker/spikeworker.proto
protoc -I api -I google --grpc-gateway_out=api --grpc-gateway_opt=paths=source_relative api/apiserver.proto
protoc -I api -I google --swagger_out ./api/swagger api/apiserver.proto

# Build the spike-server binary
# -o: specify output file path
go build -o build/spike-server github.com/AgentGuo/spike/cmd/server

# dlv exec ./spike-server --headless --listen=:5006 --api-version=2 --accept-multiclient -- -f spike.yaml