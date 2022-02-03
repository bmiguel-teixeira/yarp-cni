package cni

type ResultSuccess struct {
	CniVersion string      `json:"cniVersion"`
	Interfaces []Interface `json:"interfaces"`
	Ips        []Ip        `json:"ips"`
	Routes     []Routes    `json:"routes"`
	Dns        Dns         `json:"dns"`
}

type Interface struct {
	Name             string `json:"name"`
	Mac              string `json:"mac"`
	NetworkNamespace string `json:"sandbox"`
}

type Ip struct {
	Address   string `json:"address"`
	Gateway   string `json:"gateway"`
	Interface int    `json:"interface"`
}

type Routes struct {
	Destination string `json:"dst"`
	Gateway     string `json:"gw"`
}

type Dns struct {
	Nameservers []string `json:"nameservers"`
	Domain      string   `json:"domain"`
	Search      []string `json:"search"`
	Options     []string `json:"options"`
}

type ResultError struct {
	CniVersion string `json:"cniVersion"`
	ExitCode   int    `json:"code"`
	Message    string `json:"msg"`
	Details    string `json:"details"`
}
