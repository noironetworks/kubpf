package statsagent

import (
	"bytes"
	"encoding/binary"
	"net"
)

type inet_v6_flow struct {
	Src_ip [4]uint32
	Dst_ip [4]uint32
	L4     proto_port
}

func (flow *inet_v6_flow) GetSrcIp() string {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, flow.Src_ip)
	return net.IP(buf.Bytes()).String()
}

func (flow *inet_v6_flow) GetDstIp() string {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, flow.Dst_ip)
	return net.IP(buf.Bytes()).String()
}

func (flow *inet_v6_flow) GetIpProto() string {
	return flow.L4.GetIpProto()
}

func (flow *inet_v6_flow) GetSPort() string {
	return flow.L4.GetSPort()
}

func (flow *inet_v6_flow) GetDPort() string {
	return flow.L4.GetDPort()
}
