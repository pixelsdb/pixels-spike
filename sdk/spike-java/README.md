# Spike java

You can use spike-java to develop your functions. spike-java is a Java-based implementation framework that you can use to develop your functions.

---

## 1. Development process:

Step 1: Install spike-java and run the following command in the `spike-java` directory
```bash
mvn clean install
```

Step 2: Implement the `io.pixelsdb.pixels.spike.handler.RequestHandler` interface. You need to implement the `execute` method. You can refer to the implementation of `spike-java-example`.

Step 3: Package the implemented interface into a Docker image. Here, take `spike-java-example` as an example and execute the following command.

in x86_64:
```bash
# docker build --build-arg HANDLER_JAR_FILE={Your handler jar file path} --build-arg IMPL_JAR_FILE={Your handler implement jar file path} --platform linux/amd64,linux/arm64 -t {Your aws ECR}:{version} --push .
docker build --build-arg JAR_FILE=spike-java-example/target/spike-java-example-1.0-SNAPSHOT.jar --platform linux/amd64,linux/arm64 -t 013072238852.dkr.ecr.cn-north-1.amazonaws.com.cn/agentguo/spike-java-worker:1.0 --push .
```

in arm64:
```bash
docker buildx create --use
# docker buildx build --build-arg HANDLER_JAR_FILE={Your handler jar file path} --build-arg IMPL_JAR_FILE={Your handler implement jar file path} --platform linux/amd64,linux/arm64 -t {Your aws ECR}:{version} --push .
docker buildx build --build-arg HANDLER_JAR_FILE=spike-java-handler/target/spike-java-handler-1.0-SNAPSHOT.jar --build-arg IMPL_JAR_FILE=spike-java-example/target/spike-java-example-1.0-SNAPSHOT.jar --platform linux/amd64,linux/arm64 -t 013072238852.dkr.ecr.cn-north-1.amazonaws.com.cn/agentguo/spike-java-worker:1.0 --push .
```


[//]: # (Step 4: Run the Docker image)

[//]: # (```bash)

[//]: # (docker run -d -p 50052:50052 spike-java-worker:1.0)

[//]: # ()
[//]: # (docker exec -it <container_id> /bin/bash)

[//]: # (```)
