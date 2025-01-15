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

package constants

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
)

type InstanceType int

const (
	UnKnow InstanceType = iota
	EC2
	Fargate
	FargateSpot
)

var str2InstanceType = map[string]InstanceType{
	"EC2":         EC2,
	"Fargate":     Fargate,
	"FargateSpot": FargateSpot,
}

var instanceType2Str = map[InstanceType]string{
	EC2:         "EC2",
	Fargate:     "Fargate",
	FargateSpot: "FargateSpot",
}

// 实现字符串到枚举的解析
func (i *InstanceType) UnmarshalYAML(value *yaml.Node) error {
	var str string
	if err := value.Decode(&str); err != nil {
		return fmt.Errorf("failed to decode InstanceType: %v", err)
	}
	if val, ok := str2InstanceType[str]; ok {
		*i = val
		return nil
	}
	return fmt.Errorf("invalid InstanceType: %s", str)
}

// 实现枚举到字符串的转换
func (i InstanceType) String() string {
	if str, ok := instanceType2Str[i]; ok {
		return str
	}
	return "UnKnow"
}

func (i *InstanceType) Scan(value interface{}) error {
	switch v := value.(type) {
	case string:
		// 如果数据库直接返回字符串
		if val, ok := str2InstanceType[v]; ok {
			*i = val
			return nil
		}
		return fmt.Errorf("invalid InstanceType value: %s", v)
	case []byte:
		// 如果数据库返回字节切片，将其转换为字符串
		str := string(v)
		if val, ok := str2InstanceType[str]; ok {
			*i = val
			return nil
		}
		return fmt.Errorf("invalid InstanceType value: %s", str)
	default:
		// 如果值类型不匹配，返回错误
		return fmt.Errorf("InstanceType should be a string or []byte, got %T", value)
	}
}

func (i InstanceType) Value() (driver.Value, error) {
	str, ok := instanceType2Str[i]
	if !ok {
		return nil, errors.New("invalid InstanceType value")
	}
	return str, nil
}
