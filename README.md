# pixels-spike
A cloud-native computing framework for serverless query processing.

## 1. How to get started

step1: install go(requires Go version 1.22.7 or higher)

step2: start MySQL, for example, start MySQL with Docker, and then create the `spike` database

```shell
# Download the mysql:8.0.31 image
docker pull mysql:8.0.31

# Start the MySQL instance
docker run --name spike-mysql -e MYSQL_ROOT_PASSWORD=spikepassword -p 3306:3306 -d mysql:8.0.31

# After the MySQL instance starts, enter the MySQL container
docker exec -it mysql-spike mysql -uroot -pspikepassword -e "CREATE DATABASE spike;"
```

step3: build spike

```shell
sh build.sh
```

step4: run spike

```shell
cd build
./spike-server -f spike.yaml
```