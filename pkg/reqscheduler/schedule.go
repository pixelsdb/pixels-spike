package reqscheduler

import (
	"math/rand"
	"sort"
	"time"
)

// InstanceStat 表示函数实例的统计信息
type InstanceStat struct {
	AwsServiceName  string
	Ipv4            string
	Cpu             int32
	Memory          int32
	CpuUsed         int32
	MemoryUsed      int32
	CpuUsageRate    float64
	MemoryUsageRate float64
	AvgUsageRate    float64
}

// ReqScheduleInfo 表示请求调度信息
type ReqScheduleInfo struct {
	ReqId                uint64
	ReqPayload           string
	FunctionName         string
	PlacedAwsServiceName string
	PlacedInsIpv4        string
	RequiredCpu          int32
	RequiredMemory       int32
}

// Scheduler 调度器接口
type Scheduler interface {
	Schedule(req *Request, reqScheduleInfo []*ReqScheduleInfo, insStatMap map[string]*InstanceStat) *ReqScheduleInfo
}

// BaseScheduler 基础调度器实现
type BaseScheduler struct{}

// Schedule 实现基础调度算法
func (s *BaseScheduler) Schedule(req *Request, reqScheduleInfo []*ReqScheduleInfo, insStatMap map[string]*InstanceStat) *ReqScheduleInfo {
	// 计算每个实例的资源使用情况
	for _, reqInfo := range reqScheduleInfo {
		if stat, ok := insStatMap[reqInfo.PlacedAwsServiceName]; ok {
			stat.CpuUsed += reqInfo.RequiredCpu
			stat.MemoryUsed += reqInfo.RequiredMemory
		}
	}

	// 计算使用率并创建实例列表
	insStatList := make([]*InstanceStat, 0, len(insStatMap))
	for _, stat := range insStatMap {
		stat.CpuUsageRate = float64(stat.CpuUsed) / float64(stat.Cpu)
		stat.MemoryUsageRate = float64(stat.MemoryUsed) / float64(stat.Memory)
		// 仅考虑 CPU 使用率
		stat.AvgUsageRate = stat.CpuUsageRate
		insStatList = append(insStatList, stat)
	}

	// 按 CPU、内存和平均使用率排序
	sort.Slice(insStatList, func(i, j int) bool {
		if insStatList[i].Cpu != insStatList[j].Cpu {
			return insStatList[i].Cpu < insStatList[j].Cpu
		} else if insStatList[i].Memory != insStatList[j].Memory {
			return insStatList[i].Memory < insStatList[j].Memory
		} else {
			return insStatList[i].AvgUsageRate > insStatList[j].AvgUsageRate
		}
	})

	// 选择第一个满足资源要求的实例
	for _, stat := range insStatList {
		if stat.CpuUsed+req.RequiredCpu <= stat.Cpu && stat.MemoryUsed+req.RequiredMemory <= stat.Memory {
			return &ReqScheduleInfo{
				ReqId:                req.RequestID,
				ReqPayload:           req.ReqPayload,
				FunctionName:         req.FunctionName,
				PlacedAwsServiceName: stat.AwsServiceName,
				PlacedInsIpv4:        stat.Ipv4,
				RequiredCpu:          req.RequiredCpu,
				RequiredMemory:       req.RequiredMemory,
			}
		}
	}

	return nil
}

// OneTaskPerNodeScheduler 单任务调度器实现，确保每个节点只执行一个任务
type OneTaskPerNodeScheduler struct{}

// Schedule 实现单任务随机调度算法
func (s *OneTaskPerNodeScheduler) Schedule(req *Request, reqScheduleInfo []*ReqScheduleInfo, insStatMap map[string]*InstanceStat) *ReqScheduleInfo {
	// 创建一个map来记录已经调度的节点
	scheduledNodes := make(map[string]bool)
	for _, reqInfo := range reqScheduleInfo {
		scheduledNodes[reqInfo.PlacedAwsServiceName] = true
	}

	// 找出所有未调度的节点
	var availableNodes []*InstanceStat
	for _, stat := range insStatMap {
		// 如果节点未被调度且满足资源要求
		if !scheduledNodes[stat.AwsServiceName] &&
			stat.Cpu >= req.RequiredCpu &&
			stat.Memory >= req.RequiredMemory {
			availableNodes = append(availableNodes, stat)
		}
	}

	// 如果没有可用节点，返回 nil
	if len(availableNodes) == 0 {
		return nil
	}

	// 随机选择一个可用节点
	rand.Seed(time.Now().UnixNano())
	selectedNode := availableNodes[rand.Intn(len(availableNodes))]

	return &ReqScheduleInfo{
		ReqId:                req.RequestID,
		ReqPayload:           req.ReqPayload,
		FunctionName:         req.FunctionName,
		PlacedAwsServiceName: selectedNode.AwsServiceName,
		PlacedInsIpv4:        selectedNode.Ipv4,
		RequiredCpu:          req.RequiredCpu,
		RequiredMemory:       req.RequiredMemory,
	}
}
