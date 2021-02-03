# statsagent

Kubernetes pod service stats with ebpf

## Prerequisites to build

docker 19.03

## Building

```
cd scripts
./build.sh
```

This will build the container with the required object files.
Go binary is called statsagent. Currently built container is named stalactite/statsagent:latest.

## Running

To get a daemonset in Kubernetes with the container, use the statsagent.yaml file in the scripts directory.
You can directly use the yaml file to try the latest image without building.

```
kubectl apply -f scripts/statsagent.yaml

```

### Kernel runtime dependencies

 [eBPF features by Linux version](https://github.com/iovisor/bcc/blob/master/docs/kernel-versions.md)

Has been tested on Linux 5.4. Ebpf cgroup attachment depends on linux kernel > 4.10.
By default cgroupv2 file system is mounted at /sys/fs/cgroup/unified. Default kubernetes installation
uses the cgroup heirarchy at /sys/fs/cgroup/unified/kubepods.slice. Right now these paths are hardcoded.
Additionally cgroupv1 net controllers cause issues with cgroupv2 attachment and these need to be disabled
with the boot option cgroup_no_v1=net_prio,net_cls.

### Kind

Boot option mentioned previously is also required for kind.
In the deployment file set CGROUP_ROOT to the cgroup root that docker uses for the specific kind node. This is not
easy to do manually and helm support needs to be added. Best to try on a single node cluster for now.
To obtain the cgroup root for a single node.

```
docker ps
CONTAINER ID   IMAGE                   COMMAND                  CREATED       STATUS          PORTS                       NAMES
64522c5fcf55   kindest/node:v1.18.15   "/usr/local/bin/entr…"   3 hours ago   Up 27 minutes   127.0.0.1:37029->6443/tcp   kind-control-plane
docker inspect 64522c5fcf55 | grep Pid
            "Pid": 1165,
sudo cat /proc/1165/cgroup
<..>
11:pids:/system.slice/docker-64522c5fcf554c142320fd43883b93329dc833a26c7d6397d28a6b106ddb308e.scope
<..>

```
In this case CGROUP_ROOT environment variable should be set to
`/sys/fs/cgroup/unified/system.slice/docker-64522c5fcf554c142320fd43883b93329dc833a26c7d6397d28a6b106ddb308e.scope`

## Prometheus

Metrics are exported on the container port 8010 for now. Following metrics are available: 

| pod_svc_stats | Pod to service stats |
| ------------- | -------------------- |
| statsagent_pod_svc_stats_pod_to_svc_bytes | pod to service bytes |
| statsagent_pod_svc_stats_pod_to_svc_packets | pod to service packets |
| statsagent_pod_svc_stats_svc_to_pod_bytes | service to pod bytes |
| statsagent_pod_svc_stats_svc_to_pod_packets | service to pod packets |

| pod_stats | Pod stats |
| --------- | --------- |
| statsagent_pod_stats_pod_tx_bytes | pod egress bytes |
| statsagent_pod_stats_pod_tx_packets | pod egress packets |
| statsagent_pod_stats_pod_rx_bytes | pod ingress bytes |
| statsagent_pod_stats_pod_rx_packets | pod ingress packets |

| svc_stats | Service stats |
| --------- | ------------ |
| statsagent_svc_stats_svc_tx_bytes | service egress bytes | 
| statsagent_svc_stats_svc_tx_packets | service egress packets |
| statsagent_svc_stats_svc_rx_bytes | service ingress packets |
| statsagent_svc_stats_svc_rx_packets | service ingress packets |

![pod_svc_stats](images/pod_svc_stats.png)
![svc_stats](images/svc_stats.png)
![pod_stats](images/pod_stats.png)

