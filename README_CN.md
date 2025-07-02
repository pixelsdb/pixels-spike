# pixels-spike
一个用于无服务器查询处理的云原生计算框架。

[English](README.md) | [中文](README_CN.md)

## 1. 快速开始

### 步骤1：安装 Go
需要 Go 1.22.7 或更高版本。

还需要为生成protobuf和grpc的go文件安装一些插件:
```bash
# 安装 protoc-gen-go（生成普通 Go protobuf 代码）
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
# 安装 protoc-gen-go-grpc（生成 gRPC 服务代码）
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
# 安装 protoc-gen-grpc-gateway（grpc-gateway 支持 HTTP 到 gRPC 的转发）
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
# 安装 protoc-gen-swagger（生成OpenAPI文档）
go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger@latest
```

### 步骤2：启动 MySQL
例如，使用 Docker 启动 MySQL，然后创建 `spike` 数据库：

```shell
# 下载 mysql:8.0.31 镜像
docker pull mysql:8.0.31

# 启动 MySQL 实例
docker run --name spike-mysql -e MYSQL_ROOT_PASSWORD=spikepassword -p 3306:3306 -d mysql:8.0.31

# MySQL 实例启动后，进入 MySQL 容器
docker exec -it spike-mysql mysql -uroot -pspikepassword -e "CREATE DATABASE spike;"
```

### 步骤3：初始化子模块
本项目使用了 Git 子模块，需要初始化和更新它们：

```shell
# 初始化子模块
git submodule init

# 更新子模块到最新提交
git submodule update
```

### 步骤4：构建 spike
```shell
sh build.sh
```

### 步骤5：配置 AWS 凭证
为了使用 AWS 的计算资源，我们需要配置 AWS 凭证。你可以使用[环境变量](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html#envvars-set)或[凭证文件](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html)。

### 步骤6：修改 spike 配置文件
编辑 `spike.yaml`，主要修改 `aws_config` 部分以匹配你的 AWS 集群配置。

参数说明：
* `aws_cluster`：AWS ECS 集群名称
* `aws_subnets`：用于部署函数实例的 AWS VPC 子网 ID 列表
* `aws_security_groups`：用于控制网络访问权限的 AWS 安全组 ID 列表
* `task_role`：用于授予函数实例访问 AWS 资源权限的 AWS IAM 角色名称
* `ec2_provider`：用于提供 EC2 实例资源的 AWS EC2 容量提供商名称

```yaml
aws_config:
  aws_cluster: spike_cluster_mini
  aws_subnets:
    - subnet-01930cb57dbc12f7e
    - subnet-0c77aae8c226d039c
    - subnet-02bd39d1f8b337c22
  aws_security_groups:
    - sg-02221dbcd555d5277
  task_role: PixelsFaaSRole
  ec2_provider: Infra-ECS-Cluster-spikeclustermini-d985e674-EC2CapacityProvider-FufGynLGFE0q
```

### 步骤7：运行 spike
```shell
cd build
./spike-server -f spike.yaml
```

## 2. 如何调用函数

### 步骤1：创建函数

参数说明：
* `function_name`：函数名称，用于唯一标识一个函数
* `image_url`：函数的容器镜像地址
* `resources`：函数资源配置列表，可以配置多种规格
  * `cpu`：CPU 资源大小，单位为毫核（1 核 = 1000 毫核）
  * `memory`：内存资源大小，单位为 MB
  * `min_replica`：最小副本数，即函数实例的最小数量
  * `max_replica`：最大副本数，即函数实例的最大数量

```bash
curl --location 'http://127.0.0.1:8080/v1/create_function' \
--header 'Content-Type: application/json' \
--data '{
    "function_name": "test",
    "image_url": "013072238852.dkr.ecr.cn-north-1.amazonaws.com.cn/agentguo/test:1.1",
    "resources": [
        {
            "cpu": 8192,
            "memory": 32768,
            "min_replica": 1,
            "max_replica": 5
        },
        {
            "cpu": 4096,
            "memory": 16384,
            "min_replica": 1,
            "max_replica": 5
        }
    ]
}'
```

### 步骤2：调用函数（test 函数会 sleep payload 秒）

参数说明：
* `function_name`：要调用的函数名称
* `payload`：函数的输入参数，这里表示 sleep 的秒数
* `cpu`：请求所需的 CPU 资源大小，单位为毫核（1 核 = 1000 毫核）
* `memory`：请求所需的内存资源大小，单位为 MB，设为 0 表示使用默认值

```bash
curl --location 'http://127.0.0.1:8080/v1/call_function' \
--header 'Content-Type: application/json' \
--data '{
    "function_name": "test",
    "payload": "3",
    "cpu": 8192,
    "memory": 0
}'
``` 
