# pixels-spike
A cloud-native computing framework for serverless query processing.

[English](README.md) | [中文](README_CN.md)

## 1. How to get started

### step1: Install Go
Requires Go version 1.22.7 or higher.

### step2: Start MySQL
For example, start MySQL with Docker, and then create the `spike` database:

```shell
# Download the mysql:8.0.31 image
docker pull mysql:8.0.31

# Start the MySQL instance
docker run --name spike-mysql -e MYSQL_ROOT_PASSWORD=spikepassword -p 3306:3306 -d mysql:8.0.31

# After the MySQL instance starts, enter the MySQL container
docker exec -it mysql-spike mysql -uroot -pspikepassword -e "CREATE DATABASE spike;"
```

### step3: Initialize submodules
This project uses Git submodules. You need to initialize and update them:

```shell
# Initialize submodules
git submodule init

# Update submodules to their latest commits
git submodule update
```

### step4: Build spike
```shell
sh build.sh
```

### step5: Setup AWS credentials
To use AWS computing resources, you need to configure AWS credentials. You can use either [environment variables](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html#envvars-set) or [credential files](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html).

### step6: Modify spike config file
Edit `spike.yaml`, mainly modify the `aws_config` section to match your AWS cluster configuration.

Parameter description:
* `aws_cluster`: AWS ECS cluster name
* `aws_subnets`: List of AWS VPC subnet IDs for deploying function instances
* `aws_security_groups`: List of AWS security group IDs for controlling network access permissions
* `task_role`: AWS IAM role name for granting function instances access to AWS resources
* `ec2_provider`: AWS EC2 capacity provider name for providing EC2 instance resources

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

### step7: Run spike
```shell
cd build
./spike-server -f spike.yaml
```

## 2. How to call function

### step1: Create a function

Parameter description:
* `function_name`: Function name, used to uniquely identify a function
* `image_url`: Container image URL for the function
* `resources`: List of function resource configurations, multiple specifications can be configured
  * `cpu`: CPU resource size in millicores (1 core = 1000 millicores)
  * `memory`: Memory resource size in MB
  * `min_replica`: Minimum number of replicas, i.e., minimum number of function instances
  * `max_replica`: Maximum number of replicas, i.e., maximum number of function instances

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

### step2: Call function (test function will sleep for payload seconds)

Parameter description:
* `function_name`: Name of the function to call
* `payload`: Function input parameters, here representing sleep seconds
* `cpu`: Required CPU resource size in millicores (1 core = 1000 millicores)
* `memory`: Required memory resource size in MB, set to 0 for default value

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