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

import (
	"github.com/AgentGuo/spike/pkg/constants"
	"gorm.io/gorm"
)

type FuncInstance struct {
	gorm.Model
	AwsServiceName string                 `gorm:"primaryKey;column:aws_service_name;index:idx_service_name"`
	AwsTaskArn     string                 `gorm:"column:aws_task_arn;index:idx_task_arn"`
	FunctionName   string                 `gorm:"column:function_name;index:idx_function_name"`
	Ipv4           string                 `gorm:"column:ipv4"`
	Cpu            int32                  `gorm:"column:cpu"`
	Memory         int32                  `gorm:"column:memory"`
	AwsFamily      string                 `gorm:"column:aws_family"`
	AwsRevision    int32                  `gorm:"column:aws_revision"`
	LastStatus     string                 `gorm:"column:last_status"`
	DesiredStatus  string                 `gorm:"column:desired_status"`
	LaunchType     constants.InstanceType `gorm:"column:launch_type;type:varchar(20)"`
}

func (FuncInstance) TableName() string {
	return "func_instance"
}
