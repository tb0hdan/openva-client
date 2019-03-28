package main

import (
	"context"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/tb0hdan/openva-server/netutils"

	"github.com/grandcat/zeroconf"
)

const (
	discoveryWaitTime = 10
	service           = "_openva._tcp"
	domain            = ".local"
)

func CheckServerIP(ip net.IP) bool {
	// FIXME: Add TCP connection check
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

	entries := make(chan *zeroconf.ServiceEntry)
	go func(results <-chan *zeroconf.ServiceEntry) {
		for entry := range results {
			for _, ip := range entry.AddrIPv4 {
				if CheckServerIP(ip) {
					hostPort = net.JoinHostPort(ip.String(), strconv.Itoa(entry.Port))
					break
				}
			}
			if len(hostPort) > 0 {
				break
			}
		}
	}(entries)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(discoveryWaitTime))
	defer cancel()
	err = resolver.Browse(ctx, service, domain, entries)
	if err != nil {
		log.Println("Failed to browse:", err.Error())
		return "", err
	}

	<-ctx.Done()
	// Wait some additional time to see debug messages on go routine shutdown.
	time.Sleep(1 * time.Second)
	log.Println(hostPort)
	return hostPort, nil
}
