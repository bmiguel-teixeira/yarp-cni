package ipam

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"sync"
	"yarp-cni/pkg/utils"

	"github.com/sirupsen/logrus"
)

type LocalIpamClientConfig struct {
	IpamDbPath string
}

type LocalIpamConfigFile struct {
	CIDR           string   `json:"CIDR"`
	GatewayAddress string   `json:"gateway,omitempty"`
	AllocatedIps   []string `json:"AllocatedIps"`
}

type LocalIpamClient struct {
	Config *LocalIpamClientConfig
	Log    *logrus.Logger
}

var fileMutex sync.Mutex

func NewLocalIpamClient(logger *logrus.Logger, config *LocalIpamClientConfig) *LocalIpamClient {
	return &LocalIpamClient{
		Config: config,
		Log:    logger,
	}
}

func (ipamManager *LocalIpamClient) GetGatewayAddress() (net.IP, *net.IPNet, error) {
	return ipamManager.gatewayAddress()
}

func (ipamManager *LocalIpamClient) AllocateIpv4Address() (net.IP, *net.IPNet, error) {
	return ipamManager.allocateIP()
}

func (ipamManager *LocalIpamClient) DeAllocateIpv4Address(ip net.IP) error {
	return ipamManager.deAllocateIP(ip)
}

func (ipamManager *LocalIpamClient) gatewayAddress() (net.IP, *net.IPNet, error) {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	// Load current DB
	ips, err := ipamManager.loadDB()
	if err != nil {
		return nil, nil, err
	}

	if ips.GatewayAddress != "" {
		_, ipNet, err := net.ParseCIDR(ips.CIDR)
		if err != nil {
			return nil, nil, err
		}
		ip := net.ParseIP(ips.GatewayAddress)
		ipNet.IP = ip

		return ip, ipNet, nil
	}

	// Generate next valid IP
	fileMutex.Unlock()
	ip, ipNet, err := ipamManager.allocateIP()
	if err != nil {
		return nil, nil, err
	}

	// Write new DB
	fileMutex.Lock()
	ips, err = ipamManager.loadDB()
	if err != nil {
		return nil, nil, err
	}

	ips.GatewayAddress = ip.To4().String()
	err = ipamManager.writeDB(ips)
	if err != nil {
		return nil, nil, err
	}

	return ip, ipNet, nil
}

func (ipamManager *LocalIpamClient) allocateIP() (net.IP, *net.IPNet, error) {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	// Load current DB
	ips, err := ipamManager.loadDB()
	if err != nil {
		return nil, nil, err
	}

	// Generate next valid IP
	ip, ipNet, err := net.ParseCIDR(ips.CIDR)
	if err != nil {
		return nil, nil, err
	}
	//Skip reserved IP
	ip, err = nextIpInCidr(ip, *ipNet)
	if err != nil {
		return nil, nil, err
	}

	for utils.Contains(ips.AllocatedIps, ip.To4().String()) {
		ip, err = nextIpInCidr(ip, *ipNet)
		if err != nil {
			return nil, nil, err
		}
	}

	// Write new DB
	ips.AllocatedIps = append(ips.AllocatedIps, ip.To4().String())
	err = ipamManager.writeDB(ips)
	if err != nil {
		return nil, nil, err
	}

	ipamManager.Log.Debug(fmt.Sprintf("Allocated [%s]", ip.To4().String()))
	return ip, ipNet, nil
}

func (ipamManager *LocalIpamClient) deAllocateIP(ip net.IP) error {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	// Load current DB
	ips, err := ipamManager.loadDB()
	if err != nil {
		return err
	}

	s := utils.Remove(ips.AllocatedIps, ip.To4().String())
	ips.AllocatedIps = s

	// Write new DB
	err = ipamManager.writeDB(ips)
	if err != nil {
		return err
	}

	ipamManager.Log.Debug(fmt.Sprintf("DeAllocated [%s]", ip.To4().String()))
	return nil
}

func nextIpInCidr(ip net.IP, cidr net.IPNet) (net.IP, error) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}

	if !cidr.Contains(ip) {
		return ip, fmt.Errorf("next IP is over the CIDR range")
	}

	return ip, nil
}

func (ipamManager *LocalIpamClient) loadDB() (*LocalIpamConfigFile, error) {
	db, err := os.Open(ipamManager.Config.IpamDbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	content, err := ioutil.ReadAll(db)
	if err != nil {
		return nil, err
	}

	dbFile := &LocalIpamConfigFile{}
	err = json.Unmarshal([]byte(content), &dbFile)
	if err != nil {
		return nil, err
	}

	return dbFile, nil
}

func (ipamManager *LocalIpamClient) writeDB(db *LocalIpamConfigFile) error {
	content, err := json.Marshal(db)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(ipamManager.Config.IpamDbPath, content, 0644)
	if err != nil {
		return err
	}

	return nil
}
