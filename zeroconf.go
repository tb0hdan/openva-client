package main

import (
	"context"
	"net"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/tb0hdan/openva-server/netutils"

	"github.com/grandcat/zeroconf"
)

const (
	connectionTimeOut = 3
	discoveryWaitTime = 10
	service           = "_openva._tcp"
	domain            = ".local"
)

func CheckHostPort(hostPort string) bool {
	d := net.Dialer{Timeout: time.Duration(connectionTimeOut) * time.Second}
	_, err := d.Dial("tcp", hostPort)
	return err == nil
}

func CheckServerIP(ip net.IP) bool {
	ifaces, err := netutils.GetValidIfaces()
	if err != nil {
		log.Println("Could not get list of valid interfaces", err)
		return false
	}
	for _, iface := range ifaces {
		for _, addr := range iface.Addrs {
			_, ipnet, err := net.ParseCIDR(addr.Addr)
			if err != nil {
				log.Println("Could not parse CIDR", err)
			}
			if ipnet.Contains(ip) {
				return true
			}
		}
	}
	return false
}

func DiscoverOpenVAServers() (hostPort string, err error) {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Println("Failed to initialize resolver:", err.Error())
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(discoveryWaitTime))

	entries := make(chan *zeroconf.ServiceEntry)
	go func(results <-chan *zeroconf.ServiceEntry) {
		defer cancel()
		for entry := range results {
			for _, ip := range entry.AddrIPv4 {
				hostPort = net.JoinHostPort(ip.String(), strconv.Itoa(entry.Port))
				if CheckServerIP(ip) && CheckHostPort(hostPort) {
					break
				}
			}
			if len(hostPort) > 0 {
				break
			}
		}
	}(entries)

	defer cancel()
	err = resolver.Browse(ctx, service, domain, entries)
	if err != nil {
		log.Println("Failed to browse:", err.Error())
		return "", err
	}

	<-ctx.Done()
	// Wait some additional time to see debug messages on go routine shutdown.
	time.Sleep(1 * time.Second)
	log.Debug(hostPort)
	return hostPort, nil
}
