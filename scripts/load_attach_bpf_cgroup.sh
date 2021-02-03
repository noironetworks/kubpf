#!/bin/sh

if [ -z $BPFTOOL ]
then
	BPFTOOL=/bin/bpftool
fi

if [ -z $EBPF_MOUNT ]
then
	EBPF_MOUNT=/ebpf
fi

if [ -z $EBPF_MAP_DIR]
then
	EBPF_MAP_DIR=$EBPF_MOUNT/pinned_maps
fi

if [ -z $EBPF_PROG_DIR]
then
	EBPF_PROG_DIR=$EBPF_MOUNT/prog
fi

if [ -z $CGROUP_MOUNT ]
then
	CGROUP_MOUNT=/cgroup/unified/kubepods.slice
fi

if [ -z $1 ]
	then
	mkdir -p $EBPF_MAP_DIR
	mkdir -p $EBPF_PROG_DIR
	$BPFTOOL prog loadall /bin/bpf_cgroup_kern.o $EBPF_PROG_DIR pinmaps $EBPF_MAP_DIR
	$BPFTOOL cgroup attach $CGROUP_MOUNT ingress pinned $EBPF_PROG_DIR/cgroup_skb_ingress multi
	$BPFTOOL cgroup attach $CGROUP_MOUNT egress pinned $EBPF_PROG_DIR/cgroup_skb_egress multi
elif [ $1 -eq -1 ]
then
	$BPFTOOL cgroup detach $CGROUP_MOUNT egress pinned $EBPF_PROG_DIR/cgroup_skb_egress multi
	$BPFTOOL cgroup detach $CGROUP_MOUNT ingress pinned $EBPF_PROG_DIR/cgroup_skb_ingress multi
	unlink $EBPF_PROG_DIR/cgroup_skb_ingress
	unlink $EBPF_PROG_DIR/cgroup_skb_egress
elif [ $1 -eq -2 ]
then
	$BPFTOOL cgroup detach $CGROUP_MOUNT egress pinned $EBPF_PROG_DIR/cgroup_skb_egress multi
	$BPFTOOL cgroup detach $CGROUP_MOUNT ingress pinned $EBPF_PROG_DIR/cgroup_skb_ingress multi
	unlink $EBPF_PROG_DIR/cgroup_skb_ingress
	unlink $EBPF_PROG_DIR/cgroup_skb_egress
	unlink $EBPF_MAP_DIR/v4_flow_map
	unlink $EBPF_MAP_DIR/v6_flow_map
fi
