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
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
)

type FuncMetaData struct {
	gorm.Model
	FunctionName string           `gorm:"primaryKey;column:function_name;unique;index:idx_function_name"`
	ImageUrl     string           `gorm:"column:image_url"`
	ResSpecList  ResourceSpecList `gorm:"column:resource_spec_list;type:json"`
}

func (r *ResourceSpecList) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan ResourceSpecList, expected []byte, got %T", value)
	}
	return json.Unmarshal(b, r)
}

func (r ResourceSpecList) Value() (driver.Value, error) {
	return json.Marshal(r)
}

type ResourceSpecList []ResourceSpec

type ResourceSpec struct {
	Cpu        int32
	Memory     int32
	MinReplica int32
	MaxReplica int32
	Family     string
	Revision   int32
}

func (FuncMetaData) TableName() string {
	return "func_metadata"
}
