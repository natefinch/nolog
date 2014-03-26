package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	args := append([]string{"test"}, os.Args[1:]...)

	cmd := exec.Command("go", args...)
	p, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting StdoutPipe from command: %v", err)
		os.Exit(1)
	}
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running command: %v", err)
		os.Exit(1)
	}
	scanner := bufio.NewScanner(p)
	for scanner.Scan() {
		if !strings.HasPrefix(string(scanner.Bytes()), "[LOG]") {
			fmt.Println(string(scanner.Bytes()))
		}
	}
	if scanner.Err() != nil {
		fmt.Fprintf(os.Stderr, "Error reading output of go test: %v", scanner.Err())
		os.Exit(1)
	}
}
