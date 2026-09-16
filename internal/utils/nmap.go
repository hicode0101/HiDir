package utils

import (
	"encoding/xml"
	"fmt"
	"os"
)

// nmapReport 是 nmap XML 报告中我们关心的结构子集。
type nmapReport struct {
	Hosts []nmapHost `xml:"host"`
}

type nmapHost struct {
	Hostnames []nmapHostname `xml:"hostnames>hostname"`
	Addresses []nmapAddress  `xml:"address"`
	Ports     []nmapPort     `xml:"ports>port"`
}

type nmapHostname struct {
	Name string `xml:"name,attr"`
}

type nmapAddress struct {
	Addr string `xml:"addr,attr"`
}

type nmapPort struct {
	Protocol string      `xml:"protocol,attr"`
	PortID   string      `xml:"portid,attr"`
	State    nmapState   `xml:"state"`
	Service  nmapService `xml:"service"`
}

type nmapState struct {
	State string `xml:"state,attr"`
}

type nmapService struct {
	Name string `xml:"name,attr"`
}

// ParseNmapReport 从 nmap XML 报告中提取 HTTP 目标列表，
// 仅保留 protocol=tcp、state=open 且 service 为 http/unknown 的端口。
// 与 dirsearch 的 parse_nmap 行为一致。
func ParseNmapReport(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var report nmapReport
	if err := xml.Unmarshal(data, &report); err != nil {
		return nil, err
	}
	var targets []string
	for _, host := range report.Hosts {
		hostname := ""
		if len(host.Hostnames) > 0 && host.Hostnames[0].Name != "" {
			hostname = host.Hostnames[0].Name
		} else if len(host.Addresses) > 0 {
			hostname = host.Addresses[0].Addr
		}
		for _, port := range portOrder(host.Ports) {
			if port.Protocol != "tcp" {
				// UDP 不是可靠传输，不用于 HTTP
				continue
			}
			if port.State.State != "open" {
				continue
			}
			if port.Service.Name == "http" || port.Service.Name == "unknown" {
				targets = append(targets, fmt.Sprintf("%s:%s", hostname, port.PortID))
			}
		}
	}
	return targets, nil
}

// portOrder 保持端口在报告中出现的顺序。
func portOrder(ports []nmapPort) []nmapPort {
	return ports
}
