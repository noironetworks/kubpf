#!/bin/bash
# Set up the Docker daemon
sudo apt-get update && sudo apt-get install -y apt-transport-https ca-certificates curl software-properties-common gnupg2
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key --keyring /etc/apt/trusted.gpg.d/docker.gpg add -
sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"
sudo apt-get update && sudo apt-get install -y docker-ce docker-ce-cli containerd.io
cat <<EOF | sudo tee /etc/docker/daemon.json
{
  "exec-opts": ["native.cgroupdriver=systemd"],
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "100m"
  },
  "storage-driver": "overlay2"
}
EOF
sudo mkdir -p /etc/systemd/system/docker.service.d
sudo systemctl daemon-reload
sudo systemctl restart docker
sudo systemctl enable docker
sudo usermod -G docker vagrant
sudo systemctl restart docker
curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -
sudo apt-add-repository "deb http://apt.kubernetes.io/ kubernetes-xenial main"
sudo apt-get install -y kubectl
sudo swapoff -a
sudo apt-get install -y golang curl
sed -i s/GRUB_CMDLINE_LINUX=\"\"/GRUB_CMDLINE_LINUX=\"cgroup_no_v1=net_prio,net_cls\"/g /etc/default/grub
update-grub
cd /home/vagrant
cat <<EOF | tee kind-cluster.yaml 
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  image: kindest/node:v1.18.15@sha256:5c1b980c4d0e0e8e7eb9f36f7df525d079a96169c8a8f20d8bd108c0d0889cc4
EOF
sudo chown vagrant:vagrant kind-cluster.yaml
cat <<EOF | tee run.sh
#!/bin/bash
GO111MODULE=on go get sigs.k8s.io/kind@v0.10.0
mkdir -p .kube
export PATH=$PATH:/home/vagrant/go/bin
kind create cluster --config kind-cluster.yaml
curl https://raw.githubusercontent.com/GoogleCloudPlatform/microservices-demo/master/release/kubernetes-manifests.yaml -o /home/vagrant/kubernetes-manifests.yaml
sed -i s/-\ name\:\ ENV_PLATFORM//g kubernetes-manifests.yaml
sed -i s/value\:\ \"gcp\"//g kubernetes-manifests.yaml
kubectl apply -f ./kubernetes-manifests.yaml
curl https://raw.githubusercontent.com/noironetworks/kubpf/main/scripts/statsagent.yaml -o statsagent.yaml
curl https://raw.githubusercontent.com/noironetworks/kubpf/main/scripts/deploy_kind.sh -o deploy_kind.sh
sudo chown vagrant:vagrant ./statsagent.yaml
sudo chown vagrant:vagrant ./deploy_kind.sh
chmod +x deploy_kind.sh; ./deploy_kind.sh
EOF
sudo chown vagrant:vagrant run.sh;chmod +x run.sh
