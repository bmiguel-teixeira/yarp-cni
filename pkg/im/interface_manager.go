package im

import (
	"fmt"
	"runtime"
	"yarp-cni/pkg/cni"
	"yarp-cni/pkg/ipam"
	"yarp-cni/pkg/utils"

	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

type InterfaceManager struct {
	IpamClient    ipam.IPAM
	Configuration InterfaceConfiguration
	Log           *logrus.Logger
}

func NewInterfaceManager(logger *logrus.Logger, ic InterfaceConfiguration, ipamClient ipam.IPAM) *InterfaceManager {
	return &InterfaceManager{
		IpamClient:    ipamClient,
		Configuration: ic,
		Log:           logger,
	}
}

func (im *InterfaceManager) DeleteInterface(containerId string, namespacePath string, containerInterfaceName string) (*cni.ResultSuccess, *cni.ResultError) {
	networkNsHandle, err := netns.GetFromPath(namespacePath)
	if err != nil {
		return nil, &cni.ResultError{
			ExitCode: 1,
			Message:  err.Error(),
			Details:  fmt.Sprintf("unable to load network namespace [%s]", namespacePath),
		}
	}

	result, cniErr := im.runInNetworkNamespace(
		networkNsHandle,
		func() (*cni.ResultSuccess, *cni.ResultError) {
			containerVirtualInterface, err := netlink.LinkByName(containerInterfaceName)
			if err != nil {
				return nil, &cni.ResultError{
					ExitCode: 1,
					Message:  err.Error(),
					Details:  fmt.Sprintf("unable to fetch link [%s]", containerInterfaceName),
				}
			}

			v4addr, err := netlink.AddrList(containerVirtualInterface, netlink.FAMILY_V4)
			if err != nil {
				return nil, &cni.ResultError{
					ExitCode: 1,
					Message:  err.Error(),
					Details:  fmt.Sprintf("unable to fetch addr in [%s]", containerInterfaceName),
				}
			}

			ips := []cni.Ip{}
			for _, ip := range v4addr {
				err = im.IpamClient.DeAllocateIpv4Address(ip.IP)
				if err != nil {
					return nil, &cni.ResultError{
						ExitCode: 1,
						Message:  err.Error(),
						Details:  fmt.Sprintf("unable to deallocate ip [%s]", ip.IP),
					}
				}

				ips = append(ips, cni.Ip{
					Address: ip.IP.String(),
				})
			}

			err = netlink.LinkDel(containerVirtualInterface)
			if err != nil {
				return nil, &cni.ResultError{
					ExitCode: 1,
					Message:  err.Error(),
					Details:  fmt.Sprintf("unable to delete link [%s]", containerVirtualInterface),
				}
			}

			cniResponse := cni.ResultSuccess{
				Interfaces: []cni.Interface{
					{
						Name:             containerInterfaceName,
						NetworkNamespace: namespacePath,
					},
				},
				Ips: ips,
				Routes: []cni.Routes{
					{
						Destination: "0.0.0.0/0",
					},
				},
			}
			return &cniResponse, nil
		})
	if cniErr != nil {
		return nil, cniErr
	}

	return result, nil
}

func (im *InterfaceManager) CreateInterface(containerId string, namespacePath string, containerInterfaceName string) (*cni.ResultSuccess, *cni.ResultError) {
	err := im.ensureBridgeIsPresent()
	if err != nil {
		return nil, &cni.ResultError{
			ExitCode: 1,
			Message:  err.Error(),
			Details:  "unable to create bridge",
		}
	}

	networkNsHandle, err := netns.GetFromPath(namespacePath)
	if err != nil {
		return nil, &cni.ResultError{
			ExitCode: 1,
			Message:  err.Error(),
			Details:  fmt.Sprintf("unable to load network namespace [%s]", namespacePath),
		}
	}

	hostVirtualInterfaceName := utils.TruncateString(containerId, 8)
	hostVirtualInterfaceAttrs := netlink.NewLinkAttrs()
	hostVirtualInterfaceAttrs.Name = hostVirtualInterfaceName

	containerVirtualInterfaceAttrs := netlink.NewLinkAttrs()
	containerVirtualInterfaceAttrs.Namespace = networkNsHandle
	containerVirtualInterfaceAttrs.Name = containerInterfaceName

	virtualLinkInterface := &netlink.Veth{
		LinkAttrs: hostVirtualInterfaceAttrs,
		PeerName:  containerInterfaceName,
	}
	err = netlink.LinkAdd(virtualLinkInterface)
	if err != nil {
		return nil, &cni.ResultError{
			ExitCode: 1,
			Message:  err.Error(),
			Details:  fmt.Sprintf("unable to add paired veth [%s<->%s]", hostVirtualInterfaceName, containerInterfaceName),
		}
	}

	bridgeLink, err := netlink.LinkByName(im.Configuration.BridgeName)
	if err != nil {
		return nil, &cni.ResultError{
			ExitCode: 1,
			Message:  err.Error(),
			Details:  fmt.Sprintf("unable to fetch link [%s]", im.Configuration.BridgeName),
		}
	}

	hostLink, err := netlink.LinkByName(hostVirtualInterfaceName)
	if err != nil {
		return nil, &cni.ResultError{
			ExitCode: 1,
			Message:  err.Error(),
			Details:  fmt.Sprintf("unable to fetch link [%s]", hostVirtualInterfaceName),
		}
	}

	err = netlink.LinkSetMaster(hostLink, bridgeLink.(*netlink.Bridge))
	if err != nil {
		return nil, &cni.ResultError{
			ExitCode: 1,
			Message:  err.Error(),
			Details:  fmt.Sprintf("unable to link [%s] to bridge [%s]", hostVirtualInterfaceName, im.Configuration.BridgeName),
		}
	}

	err = netlink.LinkSetUp(hostLink)
	if err != nil {
		return nil, &cni.ResultError{
			ExitCode: 1,
			Message:  err.Error(),
			Details:  fmt.Sprintf("unable to enable link [%s]", hostVirtualInterfaceName),
		}
	}

	containerVirtualInterface, err := netlink.LinkByName(containerInterfaceName)
	if err != nil {
		return nil, &cni.ResultError{
			ExitCode: 1,
			Message:  err.Error(),
			Details:  fmt.Sprintf("unable to fetch link [%s]", containerInterfaceName),
		}
	}

	err = netlink.LinkSetNsFd(containerVirtualInterface, int(networkNsHandle))
	if err != nil {
		return nil, &cni.ResultError{
			ExitCode: 1,
			Message:  err.Error(),
			Details:  fmt.Sprintf("unable to move link [%s] to network namespace [%s]", containerInterfaceName, namespacePath),
		}
	}

	result, cniErr := im.runInNetworkNamespace(
		networkNsHandle,
		func() (*cni.ResultSuccess, *cni.ResultError) {
			err = netlink.LinkSetUp(containerVirtualInterface)
			if err != nil {
				return nil, &cni.ResultError{
					ExitCode: 1,
					Message:  err.Error(),
					Details:  fmt.Sprintf("unable to enable link [%s]", containerInterfaceName),
				}
			}

			ip, ipNet, err := im.IpamClient.AllocateIpv4Address()
			if err != nil {
				return nil, &cni.ResultError{
					ExitCode: 1,
					Message:  err.Error(),
					Details:  "unable to allocate ip",
				}
			}

			ipNet.IP = ip
			ipAddr := &netlink.Addr{IPNet: ipNet, Label: ""}
			err = netlink.AddrAdd(containerVirtualInterface, ipAddr)
			if err != nil {
				return nil, &cni.ResultError{
					ExitCode: 1,
					Message:  err.Error(),
					Details:  fmt.Sprintf("unable to attach ip [%s] to interface [%s]", ip.To4().String(), containerInterfaceName),
				}
			}

			gwIp, _, err := im.IpamClient.GetGatewayAddress()
			if err != nil {
				return nil, &cni.ResultError{
					ExitCode: 1,
					Message:  err.Error(),
					Details:  "unable to get gateway address",
				}
			}

			route := &netlink.Route{
				Scope: netlink.SCOPE_UNIVERSE,
				Gw:    gwIp,
			}
			err = netlink.RouteAdd(route)
			if err != nil {
				return nil, &cni.ResultError{
					ExitCode: 1,
					Message:  err.Error(),
					Details:  fmt.Sprintf("unable to create route in network namespace [%s]", namespacePath),
				}
			}

			cniResponse := cni.ResultSuccess{
				Interfaces: []cni.Interface{
					{
						Name:             containerInterfaceName,
						NetworkNamespace: namespacePath,
					},
				},
				Ips: []cni.Ip{
					{
						Address:   ipNet.String(),
						Gateway:   gwIp.String(),
						Interface: 1, // We can assume its 1 in this implementation. 0 for loopback device, 1 for eth0
					},
				},
				Routes: []cni.Routes{
					{
						Destination: "0.0.0.0/0",
						Gateway:     gwIp.String(),
					},
				},
			}

			return &cniResponse, nil
		})

	if cniErr != nil {
		return nil, cniErr
	}

	cniResponse := cni.ResultSuccess{
		Interfaces: []cni.Interface{
			{
				Name: hostVirtualInterfaceName,
			},
		},
	}

	cniResponse.Interfaces = append(cniResponse.Interfaces, result.Interfaces...)
	cniResponse.Ips = append(cniResponse.Ips, result.Ips...)
	cniResponse.Routes = append(cniResponse.Routes, result.Routes...)
	return &cniResponse, nil
}

func (im *InterfaceManager) ensureBridgeIsPresent() error {
	_, err := netlink.LinkByName(im.Configuration.BridgeName)
	if err == nil {
		im.Log.Warn("Bridge already created. Skipping.")
		return nil
	}
	im.Log.Warn(fmt.Sprintf("Bridge [%s] does not exist", im.Configuration.BridgeName))

	la := netlink.NewLinkAttrs()
	la.Name = im.Configuration.BridgeName
	bridge := &netlink.Bridge{LinkAttrs: la}
	err = netlink.LinkAdd(bridge)
	if err != nil {
		return err
	}

	ip, ipNet, err := im.IpamClient.GetGatewayAddress()
	if err != nil {
		return err
	}
	ipNet.IP = ip
	ipAddr := &netlink.Addr{IPNet: ipNet, Label: ""}

	err = netlink.AddrAdd(bridge, ipAddr)
	if err != nil {
		return err
	}

	err = netlink.LinkSetUp(bridge)
	if err != nil {
		return err
	}

	im.Log.Warn(fmt.Sprintf("Bridge [%s] created with IP: [%s]", im.Configuration.BridgeName, ip.To4().String()))
	return nil
}

func (im *InterfaceManager) runInNetworkNamespace(networkNamespace netns.NsHandle, f func() (*cni.ResultSuccess, *cni.ResultError)) (*cni.ResultSuccess, *cni.ResultError) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hostNs, _ := netns.Get()
	defer hostNs.Close()

	err := netns.Set(networkNamespace)
	if err != nil {
		return nil, &cni.ResultError{
			ExitCode: 1,
			Message:  err.Error(),
			Details:  fmt.Sprintf("unable change network namespace to [%s]", networkNamespace),
		}
	}

	result, callErr := f()

	err = netns.Set(hostNs)
	if err != nil {
		return nil, &cni.ResultError{
			ExitCode: 1,
			Message:  err.Error(),
			Details:  fmt.Sprintf("unable change network namespace to [%s]", hostNs),
		}
	}

	return result, callErr
}
