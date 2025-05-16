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

package model

import "gorm.io/gorm"

type ReqScheduleInfo struct {
	gorm.Model
	ReqId                uint64 `gorm:"column:req_id;primaryKey"`
	ReqPayload           string `gorm:"column:req_payload"`
	FunctionName         string `gorm:"column:function_name;index:idx_function_name"`
	PlacedAwsServiceName string `gorm:"column:placed_aws_service_name;index:idx_aws_service_name"`
	PlacedInsIpv4        string `gorm:"column:placed_ins_ipv4"`
	RequiredCpu          int32  `gorm:"column:required_cpu"`
	RequiredMemory       int32  `gorm:"column:required_memory"`
}

func (ReqScheduleInfo) TableName() string {
	return "req_scheduler_info"
}
