curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.32/deb/Release.key | sudo gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
sudo chmod 644 /etc/apt/keyrings/kubernetes-apt-keyring.gpg # allow unprivileged APT programs to read this keyring
# This overwrites any existing configuration in /etc/apt/sources.list.d/kubernetes.list
echo 'deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.32/deb/ /' | sudo tee /etc/apt/sources.list.d/kubernetes.list
sudo chmod 644 /etc/apt/sources.list.d/kubernetes.list   # helps tools such as command-not-found to work correctly

### can use apt-cache madison kubectl get check available versions
sudo apt-get update
sudo apt-get install -y kubelet=1.32.1-1.1 kubeadm=1.32.1-1.1 kubectl=1.32.1-1.1
sudo apt-mark hold kubelet kubeadm kubectl 


sudo kubeadm init --pod-network-cidr=10.244.0.0/16 --cri-socket unix:///var/run/containerd/containerd.sock

mkdir -p $HOME/.kube
## [konton]: if you don't have the sudo permssion to copy the config file under /etc/kubernetes, execute followng command
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config


kubectl apply -f https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml --kubeconfig=$HOME/.kube/config

kubectl taint node `hostname` node-role.kubernetes.io/control-plane:NoSchedule-
kubectl taint node `hostname` node-role.kubernetes.io/master:NoSchedule-
kubectl label node `hostname` node-role.kubernetes.io/master=true