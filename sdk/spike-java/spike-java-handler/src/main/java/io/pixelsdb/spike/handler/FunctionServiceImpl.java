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

import io.grpc.stub.StreamObserver;

public class FunctionServiceImpl extends SpikeWorkerServiceGrpc.SpikeWorkerServiceImplBase {

    private final RequestHandler handler;

    public FunctionServiceImpl(RequestHandler handler) {
        this.handler = handler;
    }

    @Override
    public void callWorkerFunction(SpikeWorker.CallWorkerFunctionReq request,
                                   StreamObserver<SpikeWorker.CallWorkerFunctionResp> responseObserver)
    {
        try
        {
            SpikeWorker.CallWorkerFunctionResp resp = handler.execute(request);
            responseObserver.onNext(resp);
            responseObserver.onCompleted();
        }
        catch (Exception e)
        {
            responseObserver.onError(e);
        }
    }
}