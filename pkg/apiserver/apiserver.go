/*
Copyright 2024 PixelsDB.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package apiserver

import (
	"context"
	"fmt"
	"github.com/AgentGuo/spike/api"
	"github.com/AgentGuo/spike/cmd/server/config"
	"github.com/AgentGuo/spike/pkg/funcmanager"
	"github.com/AgentGuo/spike/pkg/logger"
	"github.com/AgentGuo/spike/pkg/reqscheduler"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/sirupsen/logrus"
	"log"
	"net"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type server struct {
	logger *logrus.Logger
	api.UnimplementedSpikeServiceServer
	funcManager   *funcmanager.FuncManager
	funcScheduler *reqscheduler.ReqScheduler
}

func (s *server) CallFunction(ctx context.Context, req *api.CallFunctionRequest) (*api.CallFunctionResponse, error) {
	resp, err := s.funcScheduler.CallFunction(req)
	if err != nil {
		s.logger.Errorf("call function %s failed, err: %v", req.FunctionName, err)
	}
	return resp, err
}

func (s *server) CreateFunction(ctx context.Context, req *api.CreateFunctionRequest) (*api.CreateFunctionResponse, error) {
	s.logger.Infof("create funciton %s", req.FunctionName)
	err := s.funcManager.CreateFunction(req)
	if err != nil {
		s.logger.Errorf("create function %s failed: %v", req.FunctionName, err)
		return nil, err
	}
	s.logger.Infof("create function %s success", req.FunctionName)
	return &api.CreateFunctionResponse{Code: 0, Message: "Function added"}, nil
}

func (s *server) DeleteFunction(ctx context.Context, req *api.DeleteFunctionRequest) (*api.DeleteFunctionResponse, error) {
	s.logger.Infof("delete function %s", req.FunctionName)
	err := s.funcManager.DeleteFunction(req)
	if err != nil {
		s.logger.Errorf("delete function %s failed: %v", req.FunctionName, err)
		return nil, err
	}
	s.logger.Infof("delete function %s success", req.FunctionName)
	return &api.DeleteFunctionResponse{Code: 0, Message: "Function deleted"}, nil
}

func (s *server) GetAllFunctions(context.Context, *api.Empty) (*api.GetAllFunctionsResponse, error) {
	resp, err := s.funcManager.GetAllFunction()
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *server) GetFunctionResources(ctx context.Context, req *api.GetFunctionResourcesRequest) (*api.GetFunctionResourcesResponse, error) {
	return s.funcManager.GetFunctionResources(req)
}

func (s *server) ScaleFunction(ctx context.Context, req *api.ScaleFunctionRequest) (*api.Empty, error) {
	s.logger.Infof("scale function %s, cpu: %d, memory: %d, scale_cnt: %d", req.FunctionName, req.Cpu, req.Memory, req.ScaleCnt)
	err := s.funcManager.ScaleFunction(req)
	if err != nil {
		s.logger.Errorf("scale function %s failed: %v", req.FunctionName, err)
		return nil, err
	}
	s.logger.Infof("scale function %s success", req.FunctionName)
	return &api.Empty{}, nil
}

func (s *server) GetReqScheduleInfo(ctx context.Context, req *api.GetReqScheduleInfoRequest) (*api.GetReqScheduleInfoResponse, error) {
	s.logger.Infof("get req schedule info, function name: %s", req.FunctionName)
	reqScheduleInfo, err := s.funcScheduler.GetReqScheduleInfo(req.FunctionName)
	if err != nil {
		s.logger.Errorf("get req schedule info failed, %v", err)
		return nil, err
	}
	s.logger.Infof("get req schedule info success, function name: %s", req.FunctionName)
	resp := &api.GetReqScheduleInfoResponse{}
	for _, info := range reqScheduleInfo {
		resp.ReqScheduleInfo = append(resp.ReqScheduleInfo, &api.GetReqScheduleInfoResponse_ReqScheduleInfo{
			ReqId:                info.ReqId,
			ReqPayload:           info.ReqPayload,
			FunctionName:         info.FunctionName,
			PlacedAwsServiceName: info.PlacedAwsServiceName,
			PlacedInsIpv4:        info.PlacedInsIpv4,
			RequiredCpu:          info.RequiredCpu,
			RequiredMemory:       info.RequiredMemory,
		})
	}
	return resp, nil
}

func StartApiServer() {
	address := fmt.Sprintf("0.0.0.0:%d", config.GetConfig().ServerConfig.GrpcPort)
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	api.RegisterSpikeServiceServer(grpcServer, &server{
		logger:        logger.GetLogger(),
		funcManager:   funcmanager.NewFuncManager(),
		funcScheduler: reqscheduler.NewReqScheduler(),
	})

	// Register reflection service on gRPC server.
	reflection.Register(grpcServer)

	go func() {
		log.Printf("gRPC server is running on port %d\n", config.GetConfig().ServerConfig.GrpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Start HTTP server (gRPC Gateway)
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err = api.RegisterSpikeServiceHandlerFromEndpoint(context.Background(), mux,
		fmt.Sprintf("localhost:%d", config.GetConfig().ServerConfig.GrpcPort), opts)
	if err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}

	httpAddress := fmt.Sprintf("0.0.0.0:%d", config.GetConfig().ServerConfig.HttpPort)
	log.Printf("HTTP server is running on port %d\n", config.GetConfig().ServerConfig.HttpPort)
	if err := http.ListenAndServe(httpAddress, mux); err != nil {
		log.Fatalf("Failed to serve HTTP: %v", err)
	}
}
