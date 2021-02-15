#!/bin/bash
docker ps | grep kindest | awk '{print "ls -ld /sys/fs/cgroup/systemd/system.slice/docker* | grep "$1 "|cut -d \" \" -f 9"}' > cgroup_root.sh;chmod +x cgroup_root.sh; KIND_CGROUP_ROOT=`./cgroup_root.sh`; rm -f cgroup_root.sh; 
sed  -i '67s/value:\s.*/value: KIND_CGROUP_ROOT/' ./statsagent.yaml
echo -e "sed -i 's/KIND_CGROUP_ROOT/\""${KIND_CGROUP_ROOT//\//\\/}"\"/g' statsagent.yaml" > temp.sh;chmod +x temp.sh; ./temp.sh;rm -f ./temp.sh
kubectl apply -f ./statsagent.yaml
