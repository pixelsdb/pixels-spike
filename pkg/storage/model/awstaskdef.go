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

type AwsTaskDef struct {
	gorm.Model
	TaskFamily   string `gorm:"column:task_family;index:idx_task_family_revision,unique"`
	TaskRevision int32  `gorm:"column:task_revision;index:idx_task_family_revision,unique"`
	FunctionName string `gorm:"column:function_name;index:idx_func_cpu_mem_img,unique"`
	Cpu          int32  `gorm:"column:cpu;index:idx_func_cpu_mem_img,unique"`
	Memory       int32  `gorm:"column:memory;index:idx_func_cpu_mem_img,unique"`
	ImageUrl     string `gorm:"column:image_url;index:idx_func_cpu_mem_img,unique"`
}

func (AwsTaskDef) TableName() string {
	return "aws_task_def"
}
