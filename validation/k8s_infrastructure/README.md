## Install the Kubernetes Cluster Infrastructure and Configure the Cloud Environment for NVIDIA GPUs
- Ubuntu 20.04; (Only tested)
- NVIDIA Driver 535.54.03 , CUDA 12.2;
- Helm v3.17.0;
- Kubernetes 1.32.1;
### 1. Install NVIDIA Driver, Runtime and Container Toolkit
```
$ bash install_nvidia_driver.sh
$ bash install_nvidia_container.sh
```

### 2. Install Kubernetes Cluster 
```
$ bash install_base.sh
```
Switch to superuser privileges to get containerd configuration file：
```
$ sudo su -
$ containerd config default>/etc/containerd/config.toml
$ exit
```
Edit the file `/etc/containerd/config.toml` and modify the following parameters/specification:
```
$ sudo vim /etc/containerd/config.toml
```
Set all specs of `SystemdCgroup` to `SystemdCgroup = true`; 

Set `default_runtime_name` to be `nvidia`:
```
 [plugins."io.containerd.grpc.v1.cri".containerd]
      default_runtime_name = "nvidia"
```
Then restart the containerd service and add nvidia container runtime configuration:
```
$ sudo systemctl restart containerd
$ sudo nvidia-ctk runtime configure --runtime containerd --config /etc/containerd/config.toml 
$ sudo systemctl restart containerd
```
- Install the K8s in the **Master Node** (Important: **Next steps only for master node**):
    ```
    $ bash install_master.sh
    ```
    Deploy the Kubernetes NVIDIA GPU Device plugin:
    ```
    $ kubectl create -f https://raw.githubusercontent.com/NVIDIA/k8s-device-plugin/v0.13.0/nvidia-device-plugin.yml
    ```
    Get the woker node join command (will be used in worker node configuration to join the master node):
    ```
    $ kubeadm token create --print-join-command
    ```
- Install the K8s in the **Worker Node**  (Important: **Next steps only for worker node**):
    ```
    $ bash install_worker.sh
    ```
    Use the join command printed in the master node:
    ```
    $ sudo kubeadm join xxxxxx....
    ```
