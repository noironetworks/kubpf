# statsagent

Kubernetes pod service stats with ebpf

## Prerequisites to build

docker 19.03

## Building

```
cd  scripts
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
