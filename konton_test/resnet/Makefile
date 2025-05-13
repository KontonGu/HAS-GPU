MODEL_DIR := "/models/resnet"
TENSORFLOW_MODEL := $(MODEL_DIR)/resnet50_v1.pb
ONNX_MODEL := $(MODEL_DIR)/resnet50_v1.onnx
PYTORCH_MODEL := $(MODEL_DIR)/resnet50-19c8e357.pth
ZENODO_TENSORFLOW := https://zenodo.org/record/2535873/files/resnet50_v1.pb 
ZENODO_ONNX := https://zenodo.org/record/2592612/files/resnet50_v1.onnx
ZENODO_PYTORCH := https://zenodo.org/record/4588417/files/resnet50-19c8e357.pth

default:
	@echo "run one of the following command: \n\
	1.1 make download_tensorflow_model \n\
	1.2 make download_pytorch_model \n\
	1.3 make download_onnx_model \n\
	2.1 make build_docker_pytoch \n\
	2.2 make build_docker_tensorflow \n\
	2.3 make build_docker_onnxruntime \
	"

.PHONY: create_directories
create_directories:
	@echo "Preparing directories..."
	@if [ ! -e $(MODEL_DIR) ]; then \
                mkdir -p $(MODEL_DIR) ; \
        fi

.PHONY: download_tensorflow_model
download_tensorflow_model: create_directories
	@echo "Downloading TensorFlow model from Zenodo..."
	@if [ ! -e $(TENSORFLOW_MODEL) ]; then \
		wget -O $(TENSORFLOW_MODEL) $(ZENODO_TENSORFLOW); \
	fi

.PHONY: download_pytorch_model
download_pytorch_model: create_directories
	@echo "Downloading PyTorch model from Zenodo..."
	@if [ ! -e $(PYTORCH_MODEL) ]; then \
		wget -O $(PYTORCH_MODEL) $(ZENODO_PYTORCH); \
	fi

.PHONY: download_onnx_model
download_onnx_model: create_directories
	@echo "Downloading ONNX model from Zenodo..."
	@if [ ! -e $(ONNX_MODEL) ]; then \
		wget -O $(ONNX_MODEL) $(ZENODO_ONNX); \
	fi

.PHONY: build_docker_pytoch
build_docker_pytoch: 
	@cp dockerfiles/Dockerfile.pytorch Dockerfile
	@docker build -t kontonpuku666/mlperf-resnet:pytorch-32 .
        
.PHONY: build_docker_tensorflow
build_docker_tensorflow: 
	@cp dockerfiles/Dockerfile.tensorflow Dockerfile
	@docker build -t kontonpuku666/mlperf-resnet:tensorflow .

.PHONY: build_docker_onnxruntime
build_docker_onnxruntime: 
	@cp dockerfiles/Dockerfile.onnxruntime Dockerfile
	@docker build -t kontonpuku666/mlperf-resnet:onnx .

.PHONY: build_docker_pytoch_test
build_docker_pytoch_test: 
	@cp dockerfiles/Dockerfile_test.pytorch Dockerfile
	@docker build -t kontonpuku666/mlperf-resnet:pytorch-test-32 .
