#!/bin/bash

# Create api directory if it doesn't exist
# This directory is used to store the generated source files for the function related rpc protocol used by invokers
if [ ! -d "api" ]; then
    mkdir -p api
fi

# Create swagger directory if it doesn't exist
# This directory is used to store the generated Swagger/OpenAPI documentation
if [ ! -d "api/swagger" ]; then
    mkdir -p api/swagger
fi

# Create worker directory if it doesn't exist
# This directory is used to store the generated source files for the worker related rpc protocol used by spike server
if [ ! -d "pkg/worker" ]; then
    mkdir -p pkg/worker
fi

# Add go path into the path environment variable
export PATH=$PATH:$(go env GOPATH)/bin

# Generate Go code from protobuf definitions and gRPC-Gateway code for apiserver and spikeworker
protoc -I proto -I google --go_out=api --go_opt=paths=source_relative --go-grpc_out=api --go-grpc_opt=paths=source_relative proto/apiserver.proto
protoc -I proto --go_out=pkg/worker --go_opt=paths=source_relative --go-grpc_out=pkg/worker --go-grpc_opt=paths=source_relative proto/spikeworker.proto
protoc -I proto -I google --grpc-gateway_out=api --grpc-gateway_opt=paths=source_relative proto/apiserver.proto
protoc -I proto -I google --swagger_out ./api/swagger proto/apiserver.proto


# Build the spike-server binary
# -o: specify output file path
go build -o build/spike-server pixelsdb.io/spike/cmd/server

# dlv exec ./spike-server --headless --listen=:5006 --api-version=2 --accept-multiclient -- -f spike.yaml
