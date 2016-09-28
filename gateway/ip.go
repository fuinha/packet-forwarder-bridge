package gateway

import (
	"net"

	"github.com/brocaar/loraserver/api/gw"
)

func addIPToGatewayStatsPacket(stat *gw.GatewayStatsPacket, ip net.IP) {
	var ips []string
	if existingIPs, ok := stat.CustomData["ip"]; ok {
		if existingIPs, ok := existingIPs.([]string); ok {
			for _, existingIP := range existingIPs {
				if ip.String() == existingIP {
					return
				}
			}
			ips = append(existingIPs, ip.String())
		}
	}
	stat.CustomData["ip"] = ips
}
