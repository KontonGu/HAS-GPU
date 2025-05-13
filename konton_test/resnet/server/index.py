import flask
from waitress import serve
import json
import io
import time
from PIL import Image
import numpy as np
import argparse

app = flask.Flask(__name__)
model = None
backend = None
args = None
labels_dict = None
log_enabled = True
models_dir = "/models/resnet/" 
RESNET50_SUPPORTED_PROFILE = {
    "tensorflow": {
        "inputs": ["input_tensor:0"],
        "outputs": "ArgMax:0",
        "dataset": "imagenet",
        "backend": "tensorflow",
        "model-name": "resnet50",
        "model-file": "resnet50_v1.pb",
    },
    "pytorch-native": {
        "inputs": "image",
        "outputs": "ArgMax:0",
        "dataset": "imagenet",
        "backend": "tensorflow",
        "model-name": "resnet50",
        "model-file": "resnet50-19c8e357.pth",
    },
    "onnxruntime": {
        "dataset": "imagenet",
        "inputs": None,
        "outputs": None,
        #"outputs": "ArgMax:0",
        "backend": "onnxruntime",
        "model-name": "resnet50",
        "model-file": "resnet50_v1.onnx",
    },
}
def get_args():
    """Parse commandline."""
    parser = argparse.ArgumentParser()
    parser.add_argument("--backend", choices=["tensorflow", "onnxruntime", "pytorch-native"], help="runtime to use")
    parser.add_argument("--log", action="store_true", help="log enabled")
    args = parser.parse_args()
    return args

def get_backend():
    if args.backend == "tensorflow":
        from backend_tf import BackendTensorflow
        backend = BackendTensorflow()
    elif args.backend == "onnxruntime":
        from backend_onnxruntime import BackendOnnxruntime
        backend = BackendOnnxruntime()
    elif args.backend == "pytorch-native":
        from backend_pytorch_native import BackendPytorchNative
        backend = BackendPytorchNative()
    elif args.backend == "tvm":
        from backend_tvm import BackendTVM
        backend = BackendTVM()
    elif args.backend == "tflite":
        from backend_tflite import BackendTflite
        backend = BackendTflite()
    else:
        raise ValueError("unknown backend")
    return backend

def initialize():
    global model, args, labels_dict
    args = get_args()
    backend = get_backend();
    profile = RESNET50_SUPPORTED_PROFILE[args.backend]
    model = backend.load(models_dir + profile["model-file"], inputs = profile["inputs"], outputs = profile["outputs"])
    with open('imagenet_class_index.json') as json_file:
        labels_dict = json.loads(json_file.read())

def transform_image(image):
    image = Image.open(io.BytesIO(image))
    image = image.resize((224, 224))
    image = np.array(image).astype(np.float32)
    image = image/255.
    if args.backend != "tensorflow":
        image = np.moveaxis(image, -1, 0) # NHWC
    image = image[np.newaxis, :]

    return image

def post_process(output):
    if args.backend == "tensorflow":
        labels_ = output 
    elif args.backend == "pytorch-native": 
        scores_, labels_ = output 
    elif args.backend == "onnxruntime":
        labels_, scores_ = output 
    else:
        raise ValueError("unknown backend")
    labels_ = [labels_dict[str(idx)] for idx in labels_.tolist()]
    return labels_

@app.route("/predict", methods=["POST"])
def predict():
    start = time.time()
    data = {"success": False}

    if flask.request.method == "POST":
        image = flask.request.files.get("payload")
        if image:
            image=image.read()
            image = transform_image(image)
            output = model.predict({model.inputs[0]:image})
            output = post_process(output)
            data["success"] = True
            data["output"] = output
            if log_enabled:
                print("output:", output)
                print("elapsed: ", time.time() - start, " with success ", data["success"] )

    return flask.jsonify(data)      

if __name__ == "__main__":
    print("Loading model, and starting Flask server")       
    initialize()
    serve(app, host="0.0.0.0", port=5000, threads=8)
