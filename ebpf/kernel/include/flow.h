#ifndef __FLOW_H
#define __FLOW_H

enum cgroup_direction {
    CGROUP_INGRESS,
    CGROUP_EGRESS
};

#define IPPROTO_UDP 17
#define IPPROTO_TCP 6

struct proto_port {
    __u8 ip_proto;
    __be16 sport;
    __be16 dport;    
};

struct inet_v4_flow {
    __be32 src_ip;
    __be32 dst_ip;
    struct proto_port l4;
};

struct flow_stats {
    __u64 out_bytes;
    __u64 out_packets;
    __u64 in_bytes;
    __u64 in_packets;
};

struct inet_v6_flow {
    __be32 src_ip[4];
    __be32 dst_ip[4];
    struct proto_port l4;
};

__always_inline void normalize_v4_flow(struct inet_v4_flow *v4_flow, enum cgroup_direction dir)
{
    if(dir == CGROUP_INGRESS) {
        return;
    }
    __be32 addr = v4_flow->src_ip;
    __be16 port = v4_flow->l4.sport;
    v4_flow->src_ip = v4_flow->dst_ip;
    v4_flow->dst_ip = addr;
    v4_flow->l4.sport = v4_flow->l4.dport;   
    v4_flow->l4.dport = port;
}

__always_inline void normalize_v6_flow(struct inet_v6_flow *v6_flow, enum cgroup_direction dir)
{
#define assign_v6(a,b) { a[0] = b[0]; a[1] = b[1]; a[2] = b[2]; a[3] = b[3];}
    if(dir == CGROUP_INGRESS) {
        return;
    } 
    __be32 addr[4];
    __be16 port = v6_flow->l4.sport;
    assign_v6(addr,v6_flow->src_ip);
    assign_v6(v6_flow->src_ip, v6_flow->dst_ip);
    assign_v6(v6_flow->dst_ip, addr);
    v6_flow->l4.sport = v6_flow->l4.dport;   
    v6_flow->l4.dport = port;
}
#endif /*__FLOW_H*/
