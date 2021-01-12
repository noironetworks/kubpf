#!/bin/sh

if [ -z $BPFTOOL ]
then
	BPFTOOL=/bin/bpftool
fi

if [ -z $EBPF_MOUNT ]
then
	EBPF_MOUNT=/ebpf
fi

if [ -z $CGROUP_MOUNT ]
then
	CGROUP_MOUNT=/cgroup
fi

if [ -z $1 ]
	then
	mkdir -p $EBPF_MOUNT/cgroup
	mkdir -p $EBPF_MOUNT/pinned_maps
	$BPFTOOL prog loadall /bin/bpf_cgroup_kern.o $EBPF_MOUNT/cgroup pinmaps $EBPF_MOUNT/pinned_maps
	$BPFTOOL cgroup attach $CGROUP_MOUNT/unified/kubepods.slice ingress pinned $EBPF_MOUNT/cgroup/cgroup_skb_ingress multi
	$BPFTOOL cgroup attach $CGROUP_MOUNT/unified/kubepods.slice egress pinned $EBPF_MOUNT/cgroup/cgroup_skb_egress multi
elif [ $1 -eq -1 ]
then
	$BPFTOOL cgroup detach $CGROUP_MOUNT/unified/kubepods.slice egress pinned $EBPF_MOUNT/cgroup/cgroup_skb_egress multi
	$BPFTOOL cgroup detach $CGROUP_MOUNT/unified/kubepods.slice ingress pinned $EBPF_MOUNT/cgroup/cgroup_skb_ingress multi
	unlink $EBPF_MOUNT/cgroup/cgroup_skb_ingress
	unlink $EBPF_MOUNT/cgroup/cgroup_skb_egress
elif [ $1 -eq -2 ]
then
	$BPFTOOL cgroup detach $CGROUP_MOUNT/unified/kubepods.slice egress pinned $EBPF_MOUNT/cgroup/cgroup_skb_egress multi
	$BPFTOOL cgroup detach $CGROUP_MOUNT/unified/kubepods.slice ingress pinned $EBPF_MOUNT/cgroup/cgroup_skb_ingress multi
	unlink $EBPF_MOUNT/cgroup/cgroup_skb_ingress
	unlink $EBPF_MOUNT/cgroup/cgroup_skb_egress
	unlink $EBPF_MOUNT/pinned_maps/v4_flow_map
	unlink $EBPF_MOUNT/pinned_maps/v6_flow_map
fi
