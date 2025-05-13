import os
import torch
from transformers import AutoModelForCausalLM

model_path = os.path.join(os.getcwd(), "./gpt-j/model/")
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