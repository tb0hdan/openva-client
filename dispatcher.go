package main

import (
	"fmt"
	"strings"
)

func commandDispatcher(commands <-chan string) {
	for cmd := range commands {
		cmd = strings.TrimSpace(cmd)
		fmt.Println(cmd)
	}
}
