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

package io.pixelsdb.spike.handler.impl;

import io.pixelsdb.spike.handler.RequestHandler;
import io.pixelsdb.spike.handler.SpikeWorker;

public class Example implements RequestHandler
{
    @Override
    public SpikeWorker.CallWorkerFunctionResp execute(SpikeWorker.CallWorkerFunctionReq request)
    {
        int sleepSeconds = Integer.parseInt(request.getPayload());
        try
        {
            Thread.sleep(sleepSeconds * 1000L);
        }
        catch (InterruptedException e)
        {
            throw new RuntimeException(e);
        }

        // example：reverse the payload string
        return SpikeWorker.CallWorkerFunctionResp.newBuilder()
                .setRequestId(request.getRequestId())
                .setPayload(new StringBuilder(request.getPayload()).reverse().toString())
                .build();
    }
}
