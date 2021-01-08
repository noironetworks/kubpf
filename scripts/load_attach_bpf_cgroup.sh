#!/bin/sh
if [ -z $1 ]
	then
	mkdir -p /sys/fs/bpf/cgroup
	mkdir -p /sys/fs/bpf/pinned_maps
	bpftool prog loadall bpf_cgroup_kern.o /sys/fs/bpf/cgroup pinmaps /sys/fs/bpf/pinned_maps
	bpftool cgroup attach /sys/fs/cgroup/unified/kubepods.slice ingress pinned /sys/fs/bpf/cgroup/cgroup_skb_ingress multi
	bpftool cgroup attach /sys/fs/cgroup/unified/kubepods.slice egress pinned /sys/fs/bpf/cgroup/cgroup_skb_egress multi
elif [$1 -eq -1]
then
	bpftool cgroup detach /sys/fs/cgroup/unified/kubepods.slice egress pinned /sys/fs/bpf/cgroup/cgroup_skb_egress multi
	bpftool cgroup detach /sys/fs/cgroup/unified/kubepods.slice ingress pinned /sys/fs/bpf/cgroup/cgroup_skb_ingress multi
	unlink /sys/fs/bpf/cgroup/cgroup_skb_ingress
	unlink /sys/fs/bpf/cgroup/cgroup_skb_egress
elif [$1 -eq -2]
then
	bpftool cgroup detach /sys/fs/cgroup/unified/kubepods.slice egress pinned /sys/fs/bpf/cgroup/cgroup_skb_egress multi
	bpftool cgroup detach /sys/fs/cgroup/unified/kubepods.slice ingress pinned /sys/fs/bpf/cgroup/cgroup_skb_ingress multi
	unlink /sys/fs/bpf/cgroup/cgroup_skb_ingress
	unlink /sys/fs/bpf/cgroup/cgroup_skb_egress
	unlink /sys/fs/bpf/pinned_maps/v4_flow_map
	unlink /sys/fs/bpf/pinned_maps/v6_flow_map
fi
