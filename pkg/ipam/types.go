package ipam

import "net"

type IPAM interface {
	GetGatewayAddress() (net.IP, *net.IPNet, error)
	AllocateIpv4Address() (net.IP, *net.IPNet, error)
	DeAllocateIpv4Address(net.IP) error
}
