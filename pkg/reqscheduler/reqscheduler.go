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

package reqscheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/AgentGuo/spike/api"
	"github.com/AgentGuo/spike/cmd/server/config"
	"github.com/AgentGuo/spike/pkg/logger"
	"github.com/AgentGuo/spike/pkg/storage"
	"github.com/AgentGuo/spike/pkg/storage/model"
	"github.com/AgentGuo/spike/pkg/worker"
	"github.com/sirupsen/logrus"
	"github.com/sony/sonyflake"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// Response 表示请求的响应
type Response struct {
	ResponsePayload string
	Err             error
}

type ReqScheduler struct {
	reqQueue       *ReqQueue
	mysql          *storage.Mysql
	logger         *logrus.Logger
	scheduler      Scheduler
	flake          *sonyflake.Sonyflake
	triggerCh      chan struct{}
	requestTimeout int
}

func NewReqScheduler() *ReqScheduler {
	mysqlClient := storage.NewMysql()
	r := &ReqScheduler{
		scheduler:      &BaseScheduler{},
		mysql:          mysqlClient,
		logger:         logger.GetLogger(),
		reqQueue:       NewReqQueue(),
		flake:          sonyflake.NewSonyflake(sonyflake.Settings{}),
		triggerCh:      make(chan struct{}),
		requestTimeout: config.GetConfig().ServerConfig.RequestTimeout,
	}
	err := r.CleanReqScheduleInfo()
	if err != nil {
		r.logger.Errorf("clean req schedule info failed, %v", err)
	}
	go r.ScheduleRoutine()
	return r
}

func (r *ReqScheduler) ScheduleRoutine() {
	// 创建一个定时器，每秒触发一次
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
		case <-r.triggerCh:
		}
		r.Schedule()
	}
}

func (r *ReqScheduler) Schedule() {
	// step1: get request from queue
	req := r.reqQueue.Peek()
	if req == nil {
		return
	}

	// step2: get function instance
	funcInstances, err := r.mysql.GetFuncInstanceByCondition(map[string]interface{}{
		"function_name": req.FunctionName,
		"last_status":   "RUNNING",
	})
	if err != nil {
		r.logger.Errorf("get function instance failed, %v", err)
		return
	}

	// step3: get processing request
	storageReqScheduleInfo, err := r.mysql.GetReqScheduleInfoByFunctionName(req.FunctionName)
	if err != nil {
		r.logger.Errorf("get request schedule info failed, %v", err)
		return
	}
	reqScheduleInfo := convertToReqScheduleInfo(storageReqScheduleInfo)

	// step4: prepare instance statistics
	insStatMap := make(map[string]*InstanceStat)
	for _, instance := range funcInstances {
		insStatMap[instance.AwsServiceName] = &InstanceStat{
			AwsServiceName: instance.AwsServiceName,
			Ipv4:           instance.Ipv4,
			Cpu:            instance.Cpu,
			Memory:         instance.Memory,
			CpuUsed:        0,
			MemoryUsed:     0,
		}
	}

	// step5: use scheduler to choose instance
	chosenIns := r.scheduler.Schedule(req, reqScheduleInfo, insStatMap)
	if chosenIns == nil {
		r.logger.Warnf("no available instance to handle request")
		return
	}

	// step6: update schedule info and process request
	newReqScheduleInfo := &ReqScheduleInfo{
		ReqId:                req.RequestID,
		ReqPayload:           req.ReqPayload,
		FunctionName:         req.FunctionName,
		PlacedAwsServiceName: chosenIns.PlacedAwsServiceName,
		PlacedInsIpv4:        chosenIns.PlacedInsIpv4,
		RequiredCpu:          req.RequiredCpu,
		RequiredMemory:       req.RequiredMemory,
	}
	err = r.mysql.UpdateReqScheduleInfo(convertToStorageReqScheduleInfo(newReqScheduleInfo))
	if err != nil {
		r.logger.Errorf("update req schedule info failed, %v", err)
		return
	}
	r.reqQueue.Pop()
	r.logger.Infof("schedule request %d(function_name: %s, cpu: %d, memory: %d) to node: %s(%s)",
		req.RequestID, req.FunctionName, req.RequiredCpu, req.RequiredMemory, chosenIns.PlacedAwsServiceName, chosenIns.PlacedInsIpv4)
	go r.CallInstanceFunctionRoutine(req, chosenIns.PlacedInsIpv4)
}

func (r *ReqScheduler) SubmitRequest(req *api.CallFunctionRequest, respChan chan Response) error {
	// step1: construct request
	reqID, err := r.flake.NextID()
	if err != nil {
		return err
	}
	request := &Request{
		FunctionName:    req.GetFunctionName(),
		RequestID:       reqID,
		ReqPayload:      req.Payload,
		RequiredCpu:     req.Cpu,
		RequiredMemory:  req.Memory,
		RespPayloadChan: respChan,
	}
	r.logger.Infof("submit request %d(function_name: %s, cpu: %d, memory: %d), current queue size: %d",
		request.RequestID, request.FunctionName, request.RequiredCpu, request.RequiredMemory, r.reqQueue.Len())

	// step2: submit into request queue
	r.reqQueue.Push(request)
	// 触发调度
	r.triggerCh <- struct{}{}
	return nil
}

// CallFunction 对外暴露的函数调用接口
func (r *ReqScheduler) CallFunction(req *api.CallFunctionRequest) (*api.CallFunctionResponse, error) {
	respChan := make(chan Response)
	for i := 0; i < config.GetConfig().ServerConfig.MaxRetry; i++ {
		if i != 0 {
			r.logger.Warnf("retry to call function %d times", i)
		}
		// step1: submit request
		err := r.SubmitRequest(req, respChan)
		if err != nil {
			if i >= config.GetConfig().ServerConfig.MaxRetry-1 {
				r.logger.Errorf("submit request failed and reach max retry, %v", err)
				return nil, err
			} else {
				r.logger.Warnf("submit request failed, need retry, err: %v", err)
				continue
			}
		}

		// step2: wait response
		resp := <-respChan
		if resp.Err != nil {
			if i >= config.GetConfig().ServerConfig.MaxRetry-1 {
				r.logger.Errorf("get response error and reach max retry, %v", resp.Err)
				return nil, resp.Err
			} else {
				r.logger.Warnf("get response error, need retry, err: %v", resp.Err)
				continue
			}
		}
		return &api.CallFunctionResponse{ErrorCode: 0, Payload: resp.ResponsePayload}, nil
	}
	return nil, fmt.Errorf("call function failed")
}

// CallInstanceFunctionRoutine 调用实例函数的协程
func (r *ReqScheduler) CallInstanceFunctionRoutine(req *Request, instanceIpv4 string) {
	startTime := time.Now()

	respPayload, err := r.CallInstanceFunction(req.ReqPayload, req.RequestID, instanceIpv4)
	resp := Response{
		ResponsePayload: respPayload,
		Err:             err,
	}
	req.RespPayloadChan <- resp
	err = r.mysql.DeleteReqScheduleInfo(req.RequestID)
	if err != nil {
		r.logger.Errorf("delete req schedule info failed, %v", err)
	}
	defer func() {
		elapsedTime := time.Since(startTime).Seconds()
		r.logger.Infof("request %d(function_name: %s, cpu: %d, memory: %d) finished, cost time: %fs, req: %s, resp: %s",
			req.RequestID, req.FunctionName, req.RequiredCpu, req.RequiredMemory, elapsedTime, req.ReqPayload, respPayload)
	}()
}

// CallInstanceFunction 调用实例函数
func (r *ReqScheduler) CallInstanceFunction(reqPayload string, reqID uint64, instanceIpv4 string) (string, error) {
	// TODO: 这里可以做连接复用
	kaParams := keepalive.ClientParameters{
		Time:                20 * time.Second, // 20秒发送一次心跳
		Timeout:             10 * time.Second, // 等待心跳响应的最大时间
		PermitWithoutStream: false,            // 即使没有活动流，也允许发送心跳
	}
	conn, err := grpc.NewClient(fmt.Sprintf("%s:50052", instanceIpv4), grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(kaParams))
	if err != nil {
		r.logger.Errorf("call intance function failed, reqId: %d, connect to instance %s failed, %v", reqID, instanceIpv4, err)
		return "", err
	}
	defer conn.Close() // 确保连接关闭
	workerServiceClient := worker.NewSpikeWorkerServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.GetConfig().ServerConfig.RequestTimeout)*time.Second)
	defer cancel()
	funcServiceResp, err := workerServiceClient.CallWorkerFunction(ctx, &worker.CallWorkerFunctionReq{
		Payload:   reqPayload,
		RequestId: reqID,
	})
	if err != nil {
		r.logger.Errorf("call intance function failed, reqId: %d, call instance %s failed, %v", reqID, instanceIpv4, err)
		return "", err
	}
	return funcServiceResp.Payload, nil
}

// GetReqScheduleInfo 获取函数调度信息
func (r *ReqScheduler) GetReqScheduleInfo(functionName string) ([]*ReqScheduleInfo, error) {
	storageInfo, err := r.mysql.GetReqScheduleInfoByFunctionName(functionName)
	if err != nil {
		return nil, err
	}
	return convertToReqScheduleInfo(storageInfo), nil
}

// CleanReqScheduleInfo 清除函数调度信息，进程重启时调用
func (r *ReqScheduler) CleanReqScheduleInfo() error {
	return r.mysql.DeleteAllReqScheduleInfo()
}

// convertToReqScheduleInfo 将存储层的 ReqScheduleInfo 转换为调度器的 ReqScheduleInfo
func convertToReqScheduleInfo(storageInfo []model.ReqScheduleInfo) []*ReqScheduleInfo {
	result := make([]*ReqScheduleInfo, len(storageInfo))
	for i, info := range storageInfo {
		result[i] = &ReqScheduleInfo{
			ReqId:                info.ReqId,
			ReqPayload:           info.ReqPayload,
			FunctionName:         info.FunctionName,
			PlacedAwsServiceName: info.PlacedAwsServiceName,
			PlacedInsIpv4:        info.PlacedInsIpv4,
			RequiredCpu:          info.RequiredCpu,
			RequiredMemory:       info.RequiredMemory,
		}
	}
	return result
}

// convertToStorageReqScheduleInfo 将调度器的 ReqScheduleInfo 转换为存储层的 ReqScheduleInfo
func convertToStorageReqScheduleInfo(info *ReqScheduleInfo) *model.ReqScheduleInfo {
	return &model.ReqScheduleInfo{
		ReqId:                info.ReqId,
		ReqPayload:           info.ReqPayload,
		FunctionName:         info.FunctionName,
		PlacedAwsServiceName: info.PlacedAwsServiceName,
		PlacedInsIpv4:        info.PlacedInsIpv4,
		RequiredCpu:          info.RequiredCpu,
		RequiredMemory:       info.RequiredMemory,
	}
}
