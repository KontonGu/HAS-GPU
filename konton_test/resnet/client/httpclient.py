import requests

url = "http://localhost:5000/predict"

image = open("car.jpg", "rb")

files = { "payload": image }
print(files)

response = requests.post(url, files=files)

print(response.text)
