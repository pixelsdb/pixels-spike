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
	"github.com/AgentGuo/spike/api"
	"testing"
	"time"
)

func TestFuncManager_CreateFunction(t *testing.T) {
	type args struct {
		req *api.CreateFunctionRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"test1", args{&api.CreateFunctionRequest{
			FunctionName: "test",
			ImageUrl:     "013072238852.dkr.ecr.cn-north-1.amazonaws.com.cn/agentguo/spike-java-worker:1.0",
			Resources: []*api.ResourceSpec{{
				Cpu:        1024,
				Memory:     3072,
				MinReplica: 2,
				MaxReplica: 5,
			}},
		}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFuncManager()
			if err := f.CreateFunction(tt.args.req); (err != nil) != tt.wantErr {
				t.Errorf("CreateFunction() error = %v, wantErr %v", err, tt.wantErr)
			}
			time.Sleep(150 * time.Second)
		})
	}
}

func TestFuncManager_DeleteFunction(t *testing.T) {
	type args struct {
		req *api.DeleteFunctionRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"test1", args{&api.DeleteFunctionRequest{FunctionName: "test"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFuncManager()
			if err := f.DeleteFunction(tt.args.req); (err != nil) != tt.wantErr {
				t.Errorf("DeleteFunction() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
