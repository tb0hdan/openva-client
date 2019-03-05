package main

import (
	"fmt"
	"strings"
)

func commandDispatcher(commands <-chan string) {
	for cmd := range commands {
		cmd = strings.TrimSpace(cmd)
		fmt.Println(cmd)
		first := strings.Split(cmd, " ")[0]
		switch first {
		case "reboot":
			sysExit("USER_EXIT_REQ")
		default:
			fmt.Println("Unknown command")
		}
	}
}
