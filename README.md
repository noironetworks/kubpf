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
