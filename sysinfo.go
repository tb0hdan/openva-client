package main

import (
	"crypto/sha1"
	"fmt"
	"github.com/satori/go.uuid"
	"github.com/shirou/gopsutil/cpu"
	"io"
	"log"
	"net"
	"strings"
)

func GetSysUUID() string {
	var firstIface string
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Fatal(err)
	}
	for _, iface := range interfaces {
		if len(iface.HardwareAddr.String()) == 0 {
			continue
		}
		firstIface = strings.Replace(iface.HardwareAddr.String(), ":", "", -1)
		break

	}
	h := sha1.New()

	cpuinfo, err := cpu.Info()
	if err != nil {
		return ""
	}

	_, err = io.WriteString(h, cpuinfo[0].ModelName)
	if err != nil {
		return ""
	}

	uuid_str := firstIface + fmt.Sprintf("%x", h.Sum(nil))
	result, err := uuid.FromString(uuid_str[:32])
	if err != nil {
		return ""
	}

	return result.String()
}
