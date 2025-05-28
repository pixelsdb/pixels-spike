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

package config

import (
	"log"
	"os"
	"sync"

	"github.com/AgentGuo/spike/pkg/constants"
	"gopkg.in/yaml.v3"
)

var (
	configPath     string
	configInstance *SpikeConfig
	configOnce     sync.Once
)

type SpikeConfig struct {
	ServerConfig ServerConfig `yaml:"server_config"`
	AwsConfig    AwsConfig    `yaml:"aws_config"`
}

type ServerConfig struct {
	GrpcPort       int `yaml:"grpc_port"`
	HttpPort       int `yaml:"http_port"`
	RequestTimeout int `yaml:"request_timeout"`
	MaxRetry       int `yaml:"max_retry"`

	// scheduler config
	SchedulerType string `yaml:"scheduler_type"`

	// log config
	LogLevel  string `yaml:"log_level"`
	LogToFile bool   `yaml:"log_to_file"`
	LogToStd  bool   `yaml:"log_to_std"`

	// resource pool config
	HotResourcePool  constants.InstanceType `yaml:"hot_resource_pool"`
	ColdResourcePool constants.InstanceType `yaml:"cold_resource_pool"`

	// mysql config
	MysqlDsn string `yaml:"mysql_dsn"`

	// auto-scaling config
	AutoScalingStep      int     `yaml:"auto_scaling_step"`
	AutoScalingWindow    int     `yaml:"auto_scaling_window"`
	AutoScalingThreshold float64 `yaml:"auto_scaling_threshold"`
}

type AwsConfig struct {
	AwsCluster        string   `yaml:"aws_cluster"`
	AwsSubnets        []string `yaml:"aws_subnets"`
	AwsSecurityGroups []string `yaml:"aws_security_groups"`
	TaskRole          string   `yaml:"task_role"`
	UsePublicIpv4     bool     `yaml:"use_public_ipv4"`
	EC2Provider       string   `yaml:"ec2_provider"`
}

func SetConfigPath(path string) {
	configPath = path
	GetConfig()
}

func GetConfig() *SpikeConfig {
	configOnce.Do(func() {
		configInstance = &SpikeConfig{
			ServerConfig: ServerConfig{
				GrpcPort:             13306,
				HttpPort:             8080,
				RequestTimeout:       600,
				MaxRetry:             3,
				SchedulerType:        "base",
				LogLevel:             "debug",
				LogToFile:            false,
				LogToStd:             true,
				HotResourcePool:      constants.Fargate,
				ColdResourcePool:     constants.Fargate,
				MysqlDsn:             "root:spikepassword@tcp(127.0.0.1:3306)/spike?charset=utf8mb4&parseTime=True&loc=Local",
				AutoScalingStep:      5,
				AutoScalingWindow:    60,
				AutoScalingThreshold: 0.8,
			},
			AwsConfig: AwsConfig{
				AwsCluster:        "spike_cluster_mini",
				AwsSubnets:        []string{"subnet-01930cb57dbc12f7e", "subnet-0c77aae8c226d039c", "subnet-02bd39d1f8b337c22"},
				AwsSecurityGroups: []string{"sg-02221dbcd555d5277"},
				TaskRole:          "PixelsFaaSRole",
				UsePublicIpv4:     true,
				EC2Provider:       "Infra-ECS-Cluster-spikeclustermini-d985e674-EC2CapacityProvider-FufGynLGFE0q",
			},
		}
		if configPath != "" {
			if fileContent, e := os.ReadFile(configPath); e == nil {
				if e := yaml.Unmarshal(fileContent, configInstance); e != nil {
					log.Fatal("Failed to unmarshal config file: ", e)
				}
			} else {
				log.Fatal("Failed to read config file: ", e)
			}
		}
	})
	return configInstance
}
