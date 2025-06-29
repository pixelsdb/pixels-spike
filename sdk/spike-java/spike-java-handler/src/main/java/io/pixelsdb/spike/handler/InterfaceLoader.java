/*
 * Copyright 2024 PixelsDB.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package io.pixelsdb.spike.handler;

import io.grpc.Server;
import io.grpc.ServerBuilder;

import java.io.File;
import java.net.URL;
import java.net.URLClassLoader;
import java.util.ServiceLoader;

public class InterfaceLoader
{
    public static void main(String[] args)
    {
        if (args.length < 1)
        {
            System.err.println("Please provide JAR file path as argument");
            System.exit(1);
        }

        try
        {
            // Get JAR file path from args
            File jarFile = new File(args[0]);
            if (!jarFile.exists() || !jarFile.isFile())
            {
                System.err.println("The provided path is not a valid JAR file");
                System.exit(1);
            }
//            // Print JAR file contents
//            try (JarFile jar = new JarFile(jarFile)) {
//                jar.stream().forEach(entry -> System.out.println(entry.getName()));
//            } catch (IOException e) {
//                System.err.println("Failed to read JAR file contents: " + e.getMessage());
//                e.printStackTrace();
//                System.exit(1);
//            }

            URL jarUrl = jarFile.toURI().toURL();
            URLClassLoader classLoader = new URLClassLoader(new URL[]{jarUrl}, InterfaceLoader.class.getClassLoader());
            Thread.currentThread().setContextClassLoader(classLoader);

            // Use ServiceLoader to load classes implementing MyInterface
            ServiceLoader<RequestHandler> serviceLoader = ServiceLoader.load(RequestHandler.class, classLoader);
            if (!serviceLoader.iterator().hasNext())
            {
                System.err.println("ServiceLoader failed: No class implementing RequestHandler interface found");
                System.exit(1);
            }
            RequestHandler myInterface = serviceLoader.iterator().next();
            // Configure keepalive options
            Server server = ServerBuilder.forPort(50052)
                    // Set keepalive options
                    .keepAliveTime(10, java.util.concurrent.TimeUnit.SECONDS) // Send heartbeat every 10 seconds
                    .keepAliveTimeout(5, java.util.concurrent.TimeUnit.SECONDS) // Maximum time to wait for heartbeat response is 5 seconds
                    .permitKeepAliveWithoutCalls(true) // Allow sending heartbeat even without requests
                    .addService(new FunctionServiceImpl(myInterface))  // Add service implementation
                    .build()
                    .start();

            System.out.println("Server started, listening on " + server.getPort());
            server.awaitTermination();
        }
        catch (Exception e)
        {
            System.err.println("ClassLoader failed: " + e.getMessage());
            e.printStackTrace();
            System.exit(1);
        }
    }
}