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
	"container/list"
	"github.com/AgentGuo/spike/pkg/logger"
	"github.com/sirupsen/logrus"
	"sync"
)

type ReqQueue struct {
	mutex  sync.RWMutex
	qu     *list.List
	logger *logrus.Logger
}

type FuncStat struct {
	FunctionName string
	CpuMax       int32
	MemoryMax    int32
	CpuSum       int32
	MemorySum    int32
}

var reqQueueInstance *ReqQueue
var once sync.Once

func NewReqQueue() *ReqQueue {
	once.Do(func() {
		reqQueueInstance = &ReqQueue{
			qu:     list.New(),
			logger: logger.GetLogger(),
		}
	})
	return reqQueueInstance
}

func (r *ReqQueue) Len() int {
	return r.qu.Len()
}

// Peek get the top request from the queue
func (r *ReqQueue) Peek() *Request {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	if r.qu.Len() == 0 {
		return nil
	}
	topItem := r.qu.Front()
	req := topItem.Value.(*Request)
	return req
}

func (r *ReqQueue) Pop() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.qu.Len() != 0 {
		r.qu.Remove(r.qu.Front())
	}
}

func (r *ReqQueue) Push(request *Request) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.qu.PushBack(request)
}

func (r *ReqQueue) Snapshot() []*Request {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	var slice []*Request
	// deep copy
	for e := r.qu.Front(); e != nil; e = e.Next() {
		if v, ok := e.Value.(*Request); ok {
			copy := &Request{
				FunctionName:   v.FunctionName,
				RequestID:      v.RequestID,
				ReqPayload:     v.ReqPayload,
				RequiredCpu:    v.RequiredCpu,
				RequiredMemory: v.RequiredMemory,
			}
			slice = append(slice, copy)
		}
	}
	return slice
}

// GetStat 获取队列统计信息
func (r *ReqQueue) GetStat() map[string]*FuncStat {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	stat := map[string]*FuncStat{}
	for item := r.qu.Front(); item != nil; item = item.Next() {
		req := item.Value.(*Request)
		if stat[req.FunctionName] == nil {
			stat[req.FunctionName] = &FuncStat{
				FunctionName: req.FunctionName,
				CpuMax:       0,
				MemoryMax:    0,
				CpuSum:       0,
				MemorySum:    0,
			}
		}
		stat[req.FunctionName].CpuMax = max(stat[req.FunctionName].CpuMax, req.RequiredCpu)
		stat[req.FunctionName].MemoryMax = max(stat[req.FunctionName].MemoryMax, req.RequiredMemory)
		stat[req.FunctionName].CpuSum += req.RequiredCpu
		stat[req.FunctionName].MemorySum += req.RequiredMemory
	}
	return stat
}
