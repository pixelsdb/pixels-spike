protoc -I api --go_out=api --go_opt=paths=source_relative --go-grpc_out=api --go-grpc_opt=paths=source_relative api/apiserver.proto
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative pkg/worker/spikeworker.proto
protoc -I api --grpc-gateway_out=api --grpc-gateway_opt=paths=source_relative api/apiserver.proto
protoc -I api --swagger_out ./api/swagger api/apiserver.proto