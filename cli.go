package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strings"
)

func ClearScreen() {
	switch runtime.GOOS {
	case "darwin", "linux":
		os.Stdout.WriteString("\x1b[3;J\x1b[H\x1b[2J")
	default:
	}
}

func PrintHeader() {
	fmt.Println("OpenVA Shell")
	fmt.Println("---------------------")
}

func RunCLI(commands chan string) {

	reader := bufio.NewReader(os.Stdin)

	PrintHeader()
	for {
		fmt.Print("-> ")

		text, err := reader.ReadString('\n')
		// CTRL+D
		if err != nil && err.Error() == "EOF" {
			break
		}
		// CRLF -> "\n"
		text = strings.Replace(text, "\n", "", -1)

		if strings.TrimSpace(text) == "" {
			continue
		}
		switch text {
		case "clear":
			ClearScreen()
			PrintHeader()
		default:
			commands <- text
		}
	}

}
