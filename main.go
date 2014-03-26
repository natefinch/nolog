package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
)

func main() {
	args := []string{}
	var f *os.File
	if len(os.Args) > 1 && os.Args[1] == "-f" {
		var err error
		f, err = os.OpenFile("tests.out", os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening file tests.out: %v", err)
			os.Exit(1)
		}
		defer f.Close()
		args = append([]string{"test"}, os.Args[2:]...)
	} else {
		args = append([]string{"test"}, os.Args[1:]...)
	}

	cmd := exec.Command("go", args...)
	out, err := cmd.StdoutPipe()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting StdoutPipe from command: %v", err)
		os.Exit(1)
	}
	stderr, err := cmd.StderrPipe()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting StderrPipe from command: %v", err)
		os.Exit(1)
	}
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running command: %v", err)
		os.Exit(1)
	}
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go printOut(f, out, wg)
	go printErr(f, stderr, wg)
	wg.Wait()
}

func printOut(f *os.File, r io.Reader, wg *sync.WaitGroup) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		s := string(scanner.Bytes())
		if f != nil {
			fmt.Fprintln(f, s)
		}
		if !strings.HasPrefix(s, "[LOG]") {
			fmt.Println(s)
		}
	}
	if scanner.Err() != nil {
		fmt.Fprintf(os.Stderr, "Error reading output of go test: %v", scanner.Err())
		os.Exit(1)
	}
	wg.Done()
}

func printErr(f *os.File, r io.Reader, wg *sync.WaitGroup) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		s := string(scanner.Bytes())
		if f != nil {
			fmt.Fprintln(f, s)
		}
		fmt.Fprintln(os.Stderr, s)
	}
	if scanner.Err() != nil {
		fmt.Fprintf(os.Stderr, "Error reading output of go test: %v", scanner.Err())
		os.Exit(1)
	}
	wg.Done()
}
