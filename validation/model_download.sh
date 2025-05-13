#!/bin/bash

# Create base directory for models
BASE_DIR="/models"
mkdir -p $BASE_DIR/{bert,gpt-j,resnet,mobilenet,shufflenet,convnext}

# Create a temporary directory for download scripts
mkdir -p $BASE_DIR/scripts

# GPT-J download script
cat > $BASE_DIR/scripts/download_gptj.py << 'EOL'
import os
import torch
from transformers import AutoModelForCausalLM

model_path = os.path.join(os.getcwd(), "/models/gpt-j/")
if not os.path.exists(model_path):
    os.makedirs(model_path)

model_name = "EleutherAI/gpt-j-6B"
print(f"Downloading {model_name}...")
model = AutoModelForCausalLM.from_pretrained(
    model_name, 
    device_map="auto", 
    torchscript=True
)
model.save_pretrained(model_path)
print(f"Model saved to: {model_path}")
EOL

# Download models
echo "Starting model downloads..."

# GPT-J
#echo "Downloading GPT-J model..."
#python3 $BASE_DIR/scripts/download_gptj.py

# BERT
echo "Downloading BERT model..."
URL="https://zenodo.org/records/3733896/files/model.pytorch"
wget -O $BASE_DIR/bert/bert_qa.pt $URL

# ResNet50 model
echo "Downloading ResNet50 model..."
URL="https://download.pytorch.org/models/resnet50-19c8e357.pth"
wget -O $BASE_DIR/resnet50/resnet50-19c8e357.pth $URL

# MobileNet model
echo "Downloading MobileNet model..."
URL="https://download.pytorch.org/models/mobilenet_v3_large-8738ca79.pth"
wget -O $BASE_DIR/mobilenet/mobilenet_v3_large-8738ca79.pth $URL

# ShuffleNet model
echo "Downloading ShuffleNet 0.5 model..."
URL="https://download.pytorch.org/models/shufflenetv2_x0.5-f707e7126e.pth"
wget -O $BASE_DIR/shufflenet/shufflenetv2_x0.5-f707e7126e.pth $URL

echo "Downloading ShuffleNet 1.0 model..."
URL="https://download.pytorch.org/models/shufflenetv2_x1-5666bf0f80.pth"
wget -O $BASE_DIR/shufflenet/shufflenetv2_x1-5666bf0f80.pth $URL


# ConvNext model
echo "Downloading ConvNext model..."
URL="https://download.pytorch.org/models/convnext_large-ea097f82.pth"
wget -O $BASE_DIR/convnext/convnext_large-ea097f82.pth $URL

echo "Model downloads complete!"

# Cleanup
rm -rf $BASE_DIR/scripts

# Set permissions
chmod -R 755 $BASE_DIR