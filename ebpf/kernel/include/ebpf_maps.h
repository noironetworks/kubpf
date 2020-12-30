#ifndef __EBPF_MAPS_H
#define __EBPF_MAPS_H

#include "flow.h"

#define V4_FLOW_MAP_SIZE 65535
#define V6_FLOW_MAP_SIZE 65535

struct bpf_map_def SEC("maps") v4_flow_map = {
    .type = BPF_MAP_TYPE_HASH,
    .key_size = sizeof(struct inet_v4_flow),
    .value_size = sizeof(struct flow_stats),
    .max_entries = V4_FLOW_MAP_SIZE,
};

BPF_ANNOTATE_KV_PAIR(v4_flow_map, struct inet_v4_flow, struct flow_stats);

struct bpf_map_def SEC("maps") v6_flow_map = {
    .type = BPF_MAP_TYPE_HASH,
    .key_size = sizeof(struct inet_v6_flow),
    .value_size = sizeof(struct flow_stats),
    .max_entries = V6_FLOW_MAP_SIZE,
};

BPF_ANNOTATE_KV_PAIR(v6_flow_map, struct inet_v6_flow, struct flow_stats);

#endif /*__EBPF_MAPS_H*/
