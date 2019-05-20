package main // nolint typecheck

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
		_, _ = os.Stdout.WriteString("\x1b[3;J\x1b[H\x1b[2J")
	default:
	}
}

func PrintHeader() {
	_ = BuildDate
	fmt.Printf("OpenVA (%s) v%s-%s (Go %s) UUID:%s Shell\n", ProjectURL, Version, Build[len(Build)-6:], GoVersion, UUID[len(UUID)-12:])
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
