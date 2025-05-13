# Introduction
This is the resnet50 work in the FaST-GShare MLPerf Benchmark. Run one of the following command to download the models and build the docker image: 
```
$ make download_tensorflow_model
$ make download_pytorch_model 
$ make download_onnx_model 
$ make build_docker_pytoch 
$ make build_docker_tensorflow 
$ make build_docker_onnxruntime 
```
# Run
- Run the image: 
```
  $ docker run --network host --rm -v /models/resnet:/models/resnet docker.io/kontonpuku666/mlperf-resnet:[tensorflow/pytorch/onnxruntime]
```
- Run the perf script in the client folder: 
```
  $ k6 run k6.js
```

  Note: the inference result generated from the pretrained model tensorflow/onnxruntime provided by NVIDIA might be wrong, but it does not impact the performance test which we focus on
