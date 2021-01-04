#include <stdio.h>
#include <unistd.h>
#include <linux/bpf.h>
#include "bpf.h"
#include "libbpf.h"
#include "stdlib.h"
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include "errno.h"
#include "string.h"
#include <iostream>
#include "flow.h"

int main() {
    int obj, map_fd;
    struct inet_v4_flow v4_key,v4_next_key;
    struct  flow_stats value;

    map_fd = bpf_obj_get("/sys/fs/bpf/pinned_maps/v4_flow_map");
    if (map_fd < 0) {
            fprintf(stderr, "bpf_obj_get(v4_flow_map): %s(%d)\n",
                    strerror(errno), errno);
            return -1;
    }
    char pktstr[100];
    char bytestr[100];
    if(bpf_map_get_next_key(map_fd, NULL, &v4_key) != 0) {
	    return 0;
    }
    v4_next_key = v4_key;
    do{
	    v4_key = v4_next_key;
            bpf_map_lookup_elem(map_fd, &v4_key, &value);
	    struct in_addr sip,dip;
	    sip.s_addr = v4_key.src_ip;
	    dip.s_addr = v4_key.dst_ip;
	    snprintf(pktstr,99,"%llu bytes ",value.out_bytes);
	    snprintf(bytestr,99,"%llu packets",value.out_packets);
	    std::cout<< inet_ntoa(sip) << "(:" <<ntohs(v4_key.l4.sport)<< ")->" << inet_ntoa(dip) <<"(:"<<ntohs(v4_key.l4.dport)<< "):::: " << pktstr << bytestr <<std::endl;
	    snprintf(pktstr,99,"%llu bytes ",value.in_bytes);
	    snprintf(bytestr,99,"%llu packets",value.in_packets);
	    std::cout<< inet_ntoa(dip) << "(:" <<ntohs(v4_key.l4.dport)<< ")->" << inet_ntoa(sip) <<"(:"<<ntohs(v4_key.l4.sport)<< "):::: " << pktstr << bytestr <<std::endl;
    } while(bpf_map_get_next_key(map_fd, &v4_key, &v4_next_key) == 0);

}
