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

package funcmanager

import (
	"container/list"
	"fmt"
	"github.com/AgentGuo/spike/api"
	"github.com/AgentGuo/spike/cmd/server/config"
	"github.com/AgentGuo/spike/pkg/constants"
	"github.com/AgentGuo/spike/pkg/logger"
	"github.com/AgentGuo/spike/pkg/reqscheduler"
	"github.com/AgentGuo/spike/pkg/storage"
	"github.com/AgentGuo/spike/pkg/storage/model"
	"github.com/AgentGuo/spike/pkg/utils"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/sirupsen/logrus"
	"math"
	"sort"
	"strconv"
	"sync"
	"time"
)

type FuncManager struct {
	awsClient         *AwsClient
	mysql             *storage.Mysql
	logger            *logrus.Logger
	reqQueue          *reqscheduler.ReqQueue
	hotResourcePool   constants.InstanceType
	coldResourcePool  constants.InstanceType
	autoScalingStep   int
	autoScalingWindow int
	usePublicIpv4     bool
}

var (
	funcManager *FuncManager
	once        sync.Once
)

func NewFuncManager() *FuncManager {
	once.Do(func() {
		funcManager = &FuncManager{
			awsClient:         NewAwsClient(),
			mysql:             storage.NewMysql(),
			logger:            logger.GetLogger(),
			reqQueue:          reqscheduler.NewReqQueue(),
			hotResourcePool:   config.GetConfig().ServerConfig.HotResourcePool,
			coldResourcePool:  config.GetConfig().ServerConfig.ColdResourcePool,
			autoScalingStep:   config.GetConfig().ServerConfig.AutoScalingStep,
			autoScalingWindow: config.GetConfig().ServerConfig.AutoScalingWindow,
			usePublicIpv4:     config.GetConfig().AwsConfig.UsePublicIpv4,
		}
		go funcManager.FunctionAutoScaling()
		go funcManager.MonitorFunctionStatus()
	})
	return funcManager
}

func (f *FuncManager) CreateFunction(req *api.CreateFunctionRequest) error {
	// step1: check if function has been created
	if hasCreated, err := funcManager.mysql.HasFuncMetaDataByFunctionName(req.FunctionName); err != nil || hasCreated {
		if hasCreated {
			return fmt.Errorf("function has been created")
		} else {
			return err
		}
	}

	// step2: create function definition
	var resourceSpecList []model.ResourceSpec
	for _, res := range req.Resources {
		var family string
		var revision int32
		if awsTaskDef, err := f.mysql.GetAwsTaskDefByFuncCpuMenImg(req.FunctionName, res.Cpu, res.Memory, req.ImageUrl); err == nil && awsTaskDef != nil {
			family = awsTaskDef.TaskFamily
			revision = awsTaskDef.TaskRevision
		} else {
			family, revision, err = f.awsClient.RegTaskDef(req.FunctionName, res.Cpu, res.Memory, req.ImageUrl)
			if err != nil {
				return err
			}
			updateTaskDef := &model.AwsTaskDef{
				TaskFamily:   family,
				TaskRevision: revision,
				FunctionName: req.FunctionName,
				Cpu:          res.Cpu,
				Memory:       res.Memory,
				ImageUrl:     req.ImageUrl,
			}
			err = f.mysql.UpdateAwsTaskDef(updateTaskDef)
			if err != nil {
				f.logger.Errorf("update aws task def failed, err: %v", err)
			}
		}
		resourceSpecList = append(resourceSpecList, model.ResourceSpec{
			Cpu:        res.Cpu,
			Memory:     res.Memory,
			MinReplica: res.MinReplica,
			MaxReplica: res.MaxReplica,
			Family:     family,
			Revision:   revision,
		})
	}
	// TODO: 事务
	err := funcManager.mysql.CreateFuncMetaData(&model.FuncMetaData{
		FunctionName: req.FunctionName,
		ImageUrl:     req.ImageUrl,
		ResSpecList:  resourceSpecList,
	})
	if err != nil {
		f.logger.Errorf("create func meta data failed, err: %v", err)
		return err
	}

	// step3: create function instance
	var funcInstances []model.FuncInstance
	for _, res := range resourceSpecList {
		awsServiceNames, err := f.awsClient.BatchCreateInstance(res.Family, res.Revision, f.hotResourcePool, res.MinReplica)
		if err != nil {
			f.logger.Errorf("create ecs failed, err: %v", err)
			return err
		}
		for _, awsServiceName := range awsServiceNames {
			funcInstances = append(funcInstances, model.FuncInstance{
				AwsServiceName: awsServiceName,
				FunctionName:   req.FunctionName,
				Cpu:            res.Cpu,
				Memory:         res.Memory,
				AwsFamily:      res.Family,
				AwsRevision:    res.Revision,
				LastStatus:     "NOT_CREATE",
				LaunchType:     f.hotResourcePool,
			})
		}
	}
	if err := f.mysql.UpdateFuncInstanceBatch(funcInstances); err != nil {
		f.logger.Errorf("UpdateFuncInstanceBatch failed, err: %v", err)
		return err
	}
	return nil
}

func (f *FuncManager) ScaleFunction(req *api.ScaleFunctionRequest) error {
	// step1: 获取当前函数信息
	metaData, err := f.mysql.GetFuncMetaDataByFunctionName(req.FunctionName)
	if err != nil {
		f.logger.Errorf("scale function failed, get func meta data failed, functionName: %s, err: %v", req.FunctionName, err)
		return err
	}
	currentInsList, err := f.mysql.GetFuncInstanceByCondition(map[string]interface{}{"function_name": req.FunctionName,
		"cpu":    req.Cpu,
		"memory": req.Memory,
	})
	if err != nil {
		f.logger.Errorf("get func instance failed, functionName: %s, err: %v", req.FunctionName, err)
		return err
	}
	awsTaskDef, err := f.mysql.GetAwsTaskDefByFuncCpuMenImg(req.FunctionName, req.Cpu, req.Memory, metaData.ImageUrl)
	if err != nil {
		f.logger.Errorf("scale function failed, get aws task def failed, functionName: %s, err: %v", req.FunctionName, err)
		return err
	}

	// step2: 检查是否超出最大实例数
	maxReplica, minReplica := int32(-1), int32(-1)
	for _, res := range metaData.ResSpecList {
		if res.Cpu == req.Cpu && res.Memory == req.Memory {
			maxReplica, minReplica = res.MaxReplica, res.MinReplica
		}
	}
	if maxReplica == -1 {
		f.logger.Errorf("no such resource spec")
		return fmt.Errorf("no such resource spec")
	}
	realScaleCnt := req.ScaleCnt
	if req.ScaleCnt > 0 {
		if int32(len(currentInsList))+req.ScaleCnt > maxReplica {
			f.logger.Warnf("scale cnt is too large, scale cnt: %d, max replica: %d, current replica: %d", req.ScaleCnt, maxReplica, len(currentInsList))
			realScaleCnt = maxReplica - int32(len(currentInsList))
		} else {
			realScaleCnt = req.ScaleCnt
		}
	} else if req.ScaleCnt < 0 {
		if int32(len(currentInsList))+req.ScaleCnt < minReplica {
			f.logger.Warnf("scale cnt is too small, scale cnt: %d, current replica: %d", req.ScaleCnt, len(currentInsList))
			realScaleCnt = minReplica - int32(len(currentInsList))
		} else {
			realScaleCnt = req.ScaleCnt
		}
	}

	if realScaleCnt > 0 {
		awsServiceNames, err := f.awsClient.BatchCreateInstance(awsTaskDef.TaskFamily, awsTaskDef.TaskRevision, f.coldResourcePool, realScaleCnt)
		if err != nil {
			f.logger.Errorf("create ecs failed, err: %v", err)
			return err
		}
		var funcInstances []model.FuncInstance
		for _, awsServiceName := range awsServiceNames {
			funcInstances = append(funcInstances, model.FuncInstance{
				AwsServiceName: awsServiceName,
				FunctionName:   req.FunctionName,
				Cpu:            req.Cpu,
				Memory:         req.Memory,
				AwsFamily:      awsTaskDef.TaskFamily,
				AwsRevision:    awsTaskDef.TaskRevision,
				LastStatus:     "NOT_CREATE",
				LaunchType:     f.coldResourcePool,
			})
		}
		if err := f.mysql.UpdateFuncInstanceBatch(funcInstances); err != nil {
			f.logger.Errorf("UpdateFuncInstanceBatch failed, err: %v", err)
			return err
		}
		f.logger.Infof("scale function %s success, scale cnt: %d", req.FunctionName, realScaleCnt)
	} else if realScaleCnt < 0 {
		f.logger.Debugf("skip delete instance, scale cnt: %d", realScaleCnt)
		for _, instance := range currentInsList {
			if realScaleCnt >= 0 {
				break
			}
			if instance.LaunchType == f.coldResourcePool {
				if err := f.mysql.DeleteFuncInstanceServiceName(instance.AwsServiceName); err != nil {
					f.logger.Errorf("delete mysql func instance failed, serviceName: %s, err: %v", instance.AwsServiceName, err)
				}
				// 避免热实例删除
				go f.AsyncDeleteFuncInstance(instance.AwsServiceName)
				realScaleCnt++
			}
		}
	}
	return nil
}

func (f *FuncManager) AsyncDeleteFuncInstance(awsServiceName string) {
	for {
		reqList, err := f.mysql.GetReqScheduleInfoByAwsServiceName(awsServiceName)
		if err != nil {
			f.logger.Errorf("AsyncDeleteFuncInstance get req schedule info failed, awsServiceName: %s, err: %v", awsServiceName, err)
			continue
		}
		if len(reqList) == 0 {
			err = f.awsClient.DeleteInstance(awsServiceName)
			if err != nil {
				f.logger.Errorf("delete mysql func instance failed, serviceName: %s, err: %v", awsServiceName, err)
			}
			f.logger.Debugf("Async delete func instance success, serviceName: %s", awsServiceName)
			return
		}
		time.Sleep(5 * time.Second)
	}
}

func (f *FuncManager) MonitorFunctionStatus() {
	for {
		funcMetaDataList, err := f.mysql.GetFuncMetaDataByCondition(map[string]interface{}{})
		if err != nil {
			f.logger.Errorf("monitor instance status get func meta data failed, err: %v", err)
			continue
		}
		for _, funcMetaData := range funcMetaDataList {
			funcInstances, err := f.mysql.GetFuncInstanceByFunctionName(funcMetaData.FunctionName)
			if err != nil {
				f.logger.Errorf("monitor instance status get func instance failed, functionName: %s, err: %v",
					funcMetaData.FunctionName, err)
				continue
			}
			f.UpdateInstancesStatus(funcInstances)
		}
		time.Sleep(time.Second)
	}
}

func (f *FuncManager) UpdateInstancesStatus(funcInstances []model.FuncInstance) {
	// step1: 检查是否所有实例已经就绪
	var taskList []string
	taskMap := make(map[int][]int)
	var updateIns []model.FuncInstance
	for _, instance := range funcInstances {
		if instance.LastStatus != instance.DesiredStatus || instance.UpdatedAt.Add(10*time.Second).Before(time.Now()) {
			tasks, err := f.awsClient.GetAllTasks(instance.AwsServiceName)
			if err != nil {
				f.logger.Errorf("get %s's all tasks failed, err: %v", instance.AwsServiceName, err)
				continue
			}
			updateIns = append(updateIns, instance)
			updateIdx := len(updateIns) - 1
			if tasks == nil || len(tasks) == 0 {
				updateIns[updateIdx].LastStatus = "NOT_CREATED"
			} else {
				// 如果task多于1个，说明有的task挂了，重新启动了其他的task
				for _, task := range tasks {
					taskList = append(taskList, task)
					if taskMap[updateIdx] == nil {
						taskMap[updateIdx] = []int{}
					}
					taskMap[updateIdx] = append(taskMap[updateIdx], len(taskList)-1)
				}
			}
		}
	}
	if len(taskList) == 0 {
		// no need to update
		return
	}

	// step2: 获取实例当前状态
	output, err := f.awsClient.DescribeTasks(taskList)
	if err != nil {
		f.logger.Errorf("describe tasks failed, taskList: %v, err: %v", taskList, err)
		return
	}

	for i, _ := range updateIns {
		var updateTask *types.Task = nil
		for _, taskIdx := range taskMap[i] {
			task := output.Tasks[taskIdx]
			if updateTask == nil {
				updateTask = &task
			} else {
				if task.CreatedAt.After(*updateTask.CreatedAt) {
					updateTask = &task
				}
			}
		}
		if updateTask == nil {
			updateIns[i].LastStatus = "NOT_CREATED"
		} else {
			cpu, _ := strconv.Atoi(*updateTask.Cpu)
			memory, _ := strconv.Atoi(*updateTask.Memory)
			publicIpv4, privateIpv4 := "", ""
			if updateTask.Attachments != nil && len(updateTask.Attachments) != 0 {
				for _, d := range updateTask.Attachments[0].Details {
					if d.Name != nil && *d.Name == "networkInterfaceId" {
						publicIpv4, _ = f.awsClient.GetPublicIpv4(*d.Value)
					}
					if d.Name != nil && *d.Name == "privateIPv4Address" {
						privateIpv4 = *d.Value
					}
				}
			}
			updateIns[i].AwsTaskArn = *updateTask.TaskArn
			if f.usePublicIpv4 {
				updateIns[i].Ipv4 = publicIpv4
			} else {
				updateIns[i].Ipv4 = privateIpv4
			}
			updateIns[i].Cpu = int32(cpu)
			updateIns[i].Memory = int32(memory)
			updateIns[i].LastStatus = *updateTask.LastStatus
			updateIns[i].DesiredStatus = *updateTask.DesiredStatus
		}
	}
	if err := f.mysql.UpdateFuncInstanceBatch(updateIns); err != nil {
		f.logger.Errorf("update task status failed, err: %v", err)
	}
}

func (f *FuncManager) DeleteFunction(req *api.DeleteFunctionRequest) error {
	//step1: check function exist
	_, err := f.mysql.GetFuncMetaDataByFunctionName(req.FunctionName)
	if err != nil {
		f.logger.Errorf("get func meta data failed, functionName: %s, err: %v", req.FunctionName, err)
		return err
	}
	funcInstances, err := f.mysql.GetFuncInstanceByFunctionName(req.FunctionName)
	if err != nil {
		f.logger.Errorf("get func instance failed, functionName: %s, err: %v", req.FunctionName, err)
		return err
	}

	//step2: delete task
	for _, instance := range funcInstances {
		if err := f.awsClient.DeleteInstance(instance.AwsServiceName); err != nil {
			f.logger.Errorf("delete ecs failed, serviceName: %s, err: %v", instance.AwsServiceName, err)
		}
	}
	err = f.mysql.DeleteFuncInstanceFunctionName(req.FunctionName)
	if err != nil {
		f.logger.Errorf("mysql DeleteFuncTaskDataServiceName failed, err:%v", err)
	}
	err = f.mysql.DeleteFuncMetaDataByFunctionName(req.FunctionName)
	if err != nil {
		f.logger.Errorf("mysql DeleteFuncTaskDataServiceName failed, err:%v", err)
	}
	return nil
}

func (f *FuncManager) GetAllFunction() (*api.GetAllFunctionsResponse, error) {
	FuncMetaDataList, err := f.mysql.GetFuncMetaDataByCondition(map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	resp := &api.GetAllFunctionsResponse{}
	for _, data := range FuncMetaDataList {
		var resSpecList []*api.ResourceSpec
		for _, res := range data.ResSpecList {
			resSpecList = append(resSpecList, &api.ResourceSpec{
				Cpu:        res.Cpu,
				Memory:     res.Memory,
				MinReplica: res.MinReplica,
				MaxReplica: res.MaxReplica,
			})
		}
		resp.Functions = append(resp.Functions, &api.FunctionMetaData{
			FunctionName: data.FunctionName,
			ImageUrl:     data.ImageUrl,
			Resources:    resSpecList,
		})
	}
	return resp, nil
}

func (f *FuncManager) GetFunctionResources(req *api.GetFunctionResourcesRequest) (*api.GetFunctionResourcesResponse, error) {
	taskDataList, err := f.mysql.GetFuncInstanceByFunctionName(req.FunctionName)
	if err != nil {
		return nil, err
	}
	resp := &api.GetFunctionResourcesResponse{
		FunctionName: req.FunctionName,
	}
	for _, taskData := range taskDataList {
		resp.Resources = append(resp.Resources, &api.ResourceStatus{
			Ipv4:          taskData.Ipv4,
			Cpu:           taskData.Cpu,
			Memory:        taskData.Memory,
			LaunchType:    taskData.LaunchType.String(),
			LastStatus:    taskData.LastStatus,
			DesiredStatus: taskData.DesiredStatus,
		})
	}
	return resp, nil
}

func (f *FuncManager) FunctionAutoScaling() {
	windowLen := f.autoScalingWindow / f.autoScalingStep
	ticker := time.NewTicker(time.Duration(f.autoScalingStep) * time.Second)
	defer ticker.Stop()
	hisReqs := list.New()
	for {
		select {
		case <-ticker.C:
			break
		}
		allReqs, err := f.GetAllReq()
		if err != nil {
			f.logger.Errorf("get all req failed, err: %v", err)
			continue
		}
		hisReqs.PushFront(allReqs)
		if hisReqs.Len() < windowLen {
			continue
		}
		for hisReqs.Len() > windowLen {
			hisReqs.Remove(hisReqs.Back())
		}
		resDemandMap, err := f.CalResScale(hisReqs)
		if err != nil {
			f.logger.Errorf("cal res demand failed, err: %v", err)
			continue
		}
		for funcName, resDemandList := range resDemandMap {
			for _, resScale := range resDemandList {
				if resScale.ScaleCnt == 0 {
					continue
				}
				err := f.ScaleFunction(&api.ScaleFunctionRequest{
					FunctionName: funcName,
					Cpu:          resScale.Cpu,
					Memory:       resScale.Memory,
					ScaleCnt:     resScale.ScaleCnt,
				})
				if err != nil {
					f.logger.Errorf("scale function failed, resScale: %v, err: %v", resScale, err)
					continue
				}
				f.logger.Infof("scale function success, resScale: %v", resScale)
			}
		}
	}
}

type ResScale struct {
	Cpu      int32
	Memory   int32
	ScaleCnt int32
}

func (f *FuncManager) GetAllReq() ([]*reqscheduler.Request, error) {
	queuedReqs := f.reqQueue.Snapshot()
	scheduledReqs, err := f.mysql.GetReqScheduleInfoByCondition(map[string]interface{}{})
	if err != nil {
		f.logger.Errorf("get mysql req schedule info failed, err: %v", err)
		return nil, err
	}
	var allReq []*reqscheduler.Request
	for _, req := range queuedReqs {
		allReq = append(allReq, &reqscheduler.Request{
			FunctionName:   req.FunctionName,
			RequestID:      req.RequestID,
			RequiredCpu:    req.RequiredCpu,
			RequiredMemory: req.RequiredMemory,
		})
	}
	for _, req := range scheduledReqs {
		allReq = append(allReq, &reqscheduler.Request{
			FunctionName:   req.FunctionName,
			RequestID:      req.ReqId,
			RequiredCpu:    req.RequiredCpu,
			RequiredMemory: req.RequiredMemory,
		})
	}
	f.logger.Debugf("scheduled req num: %d, queued req num: %d", len(scheduledReqs), len(queuedReqs))
	return allReq, nil
}

func (f *FuncManager) CalResScale(hisReqQueue *list.List) (map[string][]*ResScale, error) {
	funcMetaDataList, err := f.mysql.GetFuncMetaDataByCondition(map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	funcMetaDataMap := make(map[string]model.FuncMetaData)
	// [ts]map[funcName][cpu+memory]required_cnt
	resDemandMap := make([]map[string]map[[2]int32]float64, hisReqQueue.Len())
	currentResMap := make(map[string]map[[2]int32]int32)
	for i := 0; i < hisReqQueue.Len(); i++ {
		resDemandMap[i] = make(map[string]map[[2]int32]float64)
	}
	for _, metaData := range funcMetaDataList {
		sort.Slice(metaData.ResSpecList, func(i, j int) bool {
			if metaData.ResSpecList[i].Cpu != metaData.ResSpecList[j].Cpu {
				return metaData.ResSpecList[i].Cpu < metaData.ResSpecList[j].Cpu
			} else {
				return metaData.ResSpecList[i].Memory < metaData.ResSpecList[j].Memory
			}
		})
		funcMetaDataMap[metaData.FunctionName] = metaData
		for i := 0; i < hisReqQueue.Len(); i++ {
			resDemandMap[i][metaData.FunctionName] = make(map[[2]int32]float64)
		}
		currentResMap[metaData.FunctionName] = make(map[[2]int32]int32)
	}

	allInsList, err := f.mysql.GetFuncInstanceByCondition(map[string]interface{}{})
	f.logger.Debugf("cuurent ins num: %d", len(allInsList))
	if err != nil {
		return nil, err
	}
	for _, ins := range allInsList {
		currentResMap[ins.FunctionName][[2]int32{ins.Cpu, ins.Memory}]++
	}

	it := hisReqQueue.Front()
	for i := 0; i < hisReqQueue.Len(); i++ {
		reqQueue := it.Value.([]*reqscheduler.Request)
		for _, req := range reqQueue {
			if metaData, ok := funcMetaDataMap[req.FunctionName]; ok {
				maxRes := metaData.ResSpecList[len(metaData.ResSpecList)-1]
				resCpu, resMemory, requireCnt := maxRes.Cpu, maxRes.Memory, max(float64(req.RequiredCpu)/float64(maxRes.Cpu), float64(req.RequiredMemory)/float64(maxRes.Memory))
				for _, res := range metaData.ResSpecList {
					if req.RequiredCpu <= res.Cpu && req.RequiredMemory <= res.Memory {
						resCpu, resMemory, requireCnt = res.Cpu, res.Memory, max(float64(req.RequiredCpu)/float64(res.Cpu), float64(req.RequiredMemory)/float64(res.Memory))
						break
					}
				}
				resDemandMap[i][metaData.FunctionName][[2]int32{resCpu, resMemory}] += requireCnt
			}
		}
		it = it.Next()
	}

	avgResDemandMap := make(map[string]map[[2]int32]float64)
	for _, metaData := range funcMetaDataList {
		avgResDemandMap[metaData.FunctionName] = make(map[[2]int32]float64)
		for i := 0; i < hisReqQueue.Len(); i++ {
			for res, demand := range resDemandMap[i][metaData.FunctionName] {
				avgResDemandMap[metaData.FunctionName][res] += demand / (float64(hisReqQueue.Len()) * config.GetConfig().ServerConfig.AutoScalingThreshold)
			}
		}
	}

	ret := make(map[string][]*ResScale)
	for funcName, metaData := range funcMetaDataMap {
		ret[funcName] = []*ResScale{}
		for _, res := range metaData.ResSpecList {
			expectResDemand := min(res.MaxReplica, max(res.MinReplica, int32(math.Round(avgResDemandMap[funcName][[2]int32{res.Cpu, res.Memory}]))))
			ret[funcName] = append(ret[funcName], &ResScale{
				Cpu:      res.Cpu,
				Memory:   res.Memory,
				ScaleCnt: expectResDemand - currentResMap[funcName][[2]int32{res.Cpu, res.Memory}],
			})
		}
	}
	//for funcName, resMap := range currentResMap {
	//	ret[funcName] = []*ResScale{}
	//	for res, cnt := range resMap {
	//		ret[funcName] = append(ret[funcName], &ResScale{
	//			Cpu:      res[0],
	//			Memory:   res[1],
	//			ScaleCnt: int32(math.Round(avgResDemandMap[funcName][res])) - cnt,
	//		})
	//	}
	//}
	f.logger.Debugf("avg res demand: %v, history res demand: %v, res scale: %s", avgResDemandMap, resDemandMap, utils.GetJson(ret))
	return ret, nil
}
