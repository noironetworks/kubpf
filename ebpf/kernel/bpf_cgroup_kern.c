#include "linux/bpf.h"
#include "linux/bpf_helpers.h"
#include "linux/bpf_endian.h"
#include "ebpf_maps.h"
#include "flow.h"
#include "ip.h"
#include "ipv6.h"
#include "tcp.h"
#include "udp.h"
#include <stddef.h>


#if 0
static inline int ip_is_fragment(struct __sk_buff *ctx, __u64 nhoff)
{
#define IP_MF                   0x2000
#define IP_OFFSET               0x1FFF
        return load_half(ctx, nhoff + offsetof(struct iphdr, frag_off))
                & (IP_MF | IP_OFFSET);
}
#endif


static __always_inline int bpf_flow_reader(struct __sk_buff *skb, enum cgroup_direction dir)
{
	struct inet_v4_flow v4_key = {
            .src_ip = 0,
            .dst_ip = 0,
            .l4.ip_proto = 0,
            .l4.sport = 0,
            .l4.dport = 0,
        };
	struct inet_v6_flow v6_key = {
            .src_ip = 0,
            .dst_ip = 0,
            .l4.ip_proto = 0,
            .l4.sport = 0,
            .l4.dport = 0,
        };

	struct flow_stats *value = NULL;
        struct flow_stats init_cgroup_ingress_stats = {
	    .out_packets = 1,
	    .out_bytes = skb->len,
            .in_packets = 0,
            .in_bytes = 0,
        };
        struct flow_stats init_cgroup_egress_stats = {
	    .out_packets = 0,
	    .out_bytes = 0,
            .in_packets = 1,
            .in_bytes = skb->len,
        };
        struct iphdr *iph = (struct iphdr *)((void *)(long)skb->data);
	if((void *)(iph+1) > (void *)(long)(skb->data_end)) {
		return 1;
	}
        if (iph->version == 4) {
            v4_key.l4.ip_proto = iph->protocol;
	    v4_key.src_ip = iph->saddr;
	    v4_key.dst_ip = iph->daddr;
            __u16 l3_offset = ((iph->ihl)<<2);
            if(v4_key.l4.ip_proto == IPPROTO_TCP) {
                struct tcphdr *tcph = (struct tcphdr *)((__u8 *)(long)(skb->data) + l3_offset);
                if(((void *)(tcph + 1) > (void *)(long)(skb->data_end))) {
                    return 1;
                }
		v4_key.l4.sport = tcph->source;
		v4_key.l4.dport = tcph->dest;
            } else if (v4_key.l4.ip_proto == IPPROTO_UDP) {
                struct udphdr *udph = (struct udphdr *)((__u8 *)(long)skb->data + l3_offset);
                if(((void *)(udph + 1) > (void *)(long)(skb->data_end))) {
                    return 1;
                }
		v4_key.l4.sport = udph->source;
		v4_key.l4.dport = udph->dest;
            }
	    normalize_v4_flow(&v4_key, dir); 
            value = bpf_map_lookup_elem(&v4_flow_map, &v4_key);
            if(!value) {
                if( dir == CGROUP_INGRESS) {
                    value = &init_cgroup_ingress_stats;
                } else {
                    value = &init_cgroup_egress_stats;
                }
                int ret = bpf_map_update_elem(&v4_flow_map, &v4_key, value, BPF_ANY);
                if(ret) {
                    char fmt[] = "Unable to update v4_flow_map:%d";
                    bpf_trace_printk(fmt,sizeof(fmt),ret);
                }
                return 1;
            }
        } else {
	    struct ipv6hdr *ip6h = (struct ipv6hdr *)((__u8 *)(long)skb->data);
	    if(((void *)(ip6h + 1) > (void *)(long)(skb->data_end))) {
                return 1;
	    }
            v6_key.l4.ip_proto = ip6h->nexthdr;
	    __builtin_memcpy(&v6_key.src_ip, &ip6h->saddr, 16);
	    __builtin_memcpy(&v6_key.dst_ip, &ip6h->daddr, 16);
            __u16 l3_offset = sizeof(struct ipv6hdr);
            if(v6_key.l4.ip_proto == IPPROTO_TCP) {    
                struct tcphdr *tcph = (struct tcphdr *)((__u8 *)(long)(skb->data) + l3_offset);
                if(((void *)(tcph + 1) > (void *)(long)(skb->data_end))) {
                    return 1;
                }
		v6_key.l4.sport = tcph->source;
		v6_key.l4.dport = tcph->dest;
            } else if (v6_key.l4.ip_proto == IPPROTO_UDP) {
                struct udphdr *udph = (struct udphdr *)((__u8 *)(long)skb->data + l3_offset);
                if(((void *)(udph + 1) > (void *)(long)(skb->data_end))) {
                    return 1;
                }
		v6_key.l4.sport = udph->source;
		v6_key.l4.dport = udph->dest;
            } 
	    normalize_v6_flow(&v6_key, dir); 
            value = bpf_map_lookup_elem(&v6_flow_map, &v6_key);
            if(!value) {
                if( dir == CGROUP_INGRESS) {
                    value = &init_cgroup_ingress_stats;
                } else {
                    value = &init_cgroup_egress_stats;
                }
                int ret = bpf_map_update_elem(&v6_flow_map, &v6_key, value, BPF_ANY);
                if(ret) {
                    char fmt[] = "Unable to update v6_flow_map:%d";
                    bpf_trace_printk(fmt,sizeof(fmt),ret);
                }
                return 1;
            }
            
        } 
	
	if (value) {
            if(dir == CGROUP_INGRESS ) {
                __sync_fetch_and_add(&value->out_bytes, skb->len);
                __sync_fetch_and_add(&value->out_packets, 1);
            } else {
                __sync_fetch_and_add(&value->in_bytes, skb->len);
                __sync_fetch_and_add(&value->in_packets, 1);
            }
        } 
	return 1;
}

SEC("cgroup_skb/ingress") int _bpf_ingress_flow_reader(struct __sk_buff *skb) {
	return bpf_flow_reader(skb,CGROUP_INGRESS);
}

SEC("cgroup_skb/egress") int bpf_egress_flow_reader(struct __sk_buff *skb) {
	return bpf_flow_reader(skb,CGROUP_EGRESS);
}

char _license[] SEC("license") = "GPL";
