package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	termcolor "github.com/fatih/color"
)

// Expected log line output
// [LOG] 0:00.003 INFO a.package body

// TODO: Perhaps error deserves a better coloring
var (
	errColorP      = termcolor.New(termcolor.FgGreen).Add(termcolor.Underline).PrintlnFunc()
	logSqBracketSP = termcolor.New(termcolor.FgWhite, termcolor.Bold).SprintFunc()
	logSP          = termcolor.New(termcolor.FgYellow, termcolor.Bold).SprintFunc()

	bracketSP = termcolor.New(termcolor.FgBlue).SprintFunc()
	quoteSP   = termcolor.New(termcolor.FgBlue, termcolor.Bold, termcolor.Italic).SprintFunc()
	kvNumSP   = termcolor.New(termcolor.FgYellow, termcolor.Bold).SprintFunc()
	kvSepSP   = termcolor.New(termcolor.FgWhite, termcolor.Bold, termcolor.Italic).SprintFunc()
	strSP     = termcolor.New(termcolor.FgWhite, termcolor.Bold).SprintFunc()
	dateSP    = termcolor.New(termcolor.FgRed, termcolor.Italic).SprintFunc()
	packSP    = termcolor.New(termcolor.FgWhite, termcolor.Italic, termcolor.Bold).SprintFunc()
	escapeSP  = termcolor.New(termcolor.FgCyan, termcolor.Bold).SprintFunc()
	escapedSP = termcolor.New(termcolor.FgMagenta, termcolor.Bold).SprintFunc()

	logHeading = fmt.Sprintf("%s%s%s", logSqBracketSP("["), logSP("LOG"), logSqBracketSP("]"))
	debugL     = termcolor.CyanString("DEBUG")
	errorL     = termcolor.RedString("ERROR")
	infoL      = termcolor.GreenString("INFO")
	warnL      = termcolor.MagentaString("WARN")
)

type nologArgs struct {
	outToFile   bool
	outFileName string
	color       bool
	verbose     bool
	gocheck     string
	args        []string
}

var nlArgs nologArgs

func init() {
	flag.BoolVar(&nlArgs.outToFile, "f", false, "setting this flag will output the logs to a file.")
	flag.StringVar(&nlArgs.outFileName, "name", "tests.log", "this is an alternative file name for the ouput.")
	flag.BoolVar(&nlArgs.color, "c", false, "setting this flag will color the output logs.")
	flag.BoolVar(&nlArgs.verbose, "v", false, "setting this flag will use -test.v=true on the test run.")
	flag.StringVar(&nlArgs.gocheck, "filter", "", "this will be used to filter tests with -gocheck.f (requires gocheck).")
	flag.Parse()
	nlArgs.args = flag.Args()
}

func main() {
	args := nlArgs.args
	var (
		f   *os.File
		err error
	)
	if nlArgs.outToFile {
		f, err = os.OpenFile(nlArgs.outFileName, os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening file %q: %v", nlArgs.outFileName, err)
			os.Exit(1)
		}
		defer f.Close()

	}
	testFlags := []string{}

	if nlArgs.verbose || nlArgs.gocheck != "" {
		testFlags = append(testFlags, "-test.v=true")
	}

	if nlArgs.gocheck != "" {
		testFlags = append(testFlags, fmt.Sprintf("-gocheck.f=%s", nlArgs.gocheck))
	}
	args = append([]string{"test"}, args...)
	args = append(args, testFlags...)
	if len(args) == 1 {
		args = append(args, "./...")
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
	var colorOut, colorErr chan string
	if nlArgs.color {
		colorOut = make(chan string)
		colorErr = make(chan string)
		wg.Add(2)
		go colorizeOut(colorOut, wg)
		go colorizeErr(colorErr, wg)
	}
	go printOut(f, out, colorOut, wg)
	go printErr(f, stderr, colorErr, wg)
	wg.Wait()
}

func tokenize(s string) string {
	tokenized := []string{" "}
	bracketCount := 0
	inStr := false
	escaped := false
	for k, _ := range s {
		// TODO (perrito666) make this more rune friendly
		c := s[k : k+1] // This will break a lot with unicode chars
		if escaped {
			escaped = false
			tokenized = append(tokenized, escapedSP(c))
			continue
		}
		switch c {
		case "{":
			bracketCount += 1
			tokenized = append(tokenized, bracketSP(c))
		case "}":
			bracketCount -= 1
			tokenized = append(tokenized, bracketSP(c))
		case "\\":
			escaped = true
			tokenized = append(tokenized, escapeSP(c))
		case "\"":
			inStr = !inStr
			tokenized = append(tokenized, quoteSP(c))
		case "1", "2", "3", "4", "5", "6", "7", "8", "9", "0":
			// I know, regexes would be better right? NO
			switch {
			case bracketCount > 0 && !inStr:
				tokenized = append(tokenized, kvNumSP(c))
			case inStr:
				tokenized = append(tokenized, strSP(c))
			default:
				tokenized = append(tokenized, c)
			}
		case ":":
			if bracketCount > 0 {
				tokenized = append(tokenized, kvSepSP(c))
			} else {
				tokenized = append(tokenized, c)
			}
		default:
			if inStr {
				tokenized = append(tokenized, strSP(c))
			} else {
				tokenized = append(tokenized, c)
			}

		}
	}
	return strings.Join(tokenized, "")
}

func nextWord(s string) (string, string) {
	s = strings.TrimLeft(s, " ")
	words := strings.SplitN(s, " ", 2)
	return words[0], words[1]
}

func extractDateInto(s, out string) (string, string) {
	date, s := nextWord(s)
	out = fmt.Sprintf("%s %s", out, dateSP(date))
	return out, s
}

func extractLogLevel(s, out string) (string, string) {
	logLevel, s := nextWord(s)
	switch {
	case strings.HasPrefix(logLevel, "DEBUG"):
		logLevel = debugL
	case strings.HasPrefix(logLevel, "ERROR"):
		logLevel = errorL
	case strings.HasPrefix(logLevel, "INFO"):
		logLevel = infoL
	case strings.HasPrefix(logLevel, "WARN"):
		logLevel = warnL
	}
	out = fmt.Sprintf("%s %s", out, logLevel)
	return out, s
}

func extractPackage(s, out string) (string, string) {
	pack, s := nextWord(s)
	out = fmt.Sprintf("%s %s", out, packSP(pack))
	return out, s
}

func colorizeOut(c chan string, wg *sync.WaitGroup) {
	for {
		s, ok := <-c
		if !ok {
			wg.Done()
			return
		}
		var out string
		if strings.HasPrefix(s, "[LOG]") {
			out = logHeading
			_, s = nextWord(s)
			out, s = extractDateInto(s, out)
			out, s = extractLogLevel(s, out)
			out, s = extractPackage(s, out)
		}
		out += tokenize(s)
		fmt.Println(out)
	}

}

func colorizeErr(c chan string, wg *sync.WaitGroup) {
	for {
		s, ok := <-c
		if !ok {
			wg.Done()
			return
		}
		errColorP(s)
	}
}

func printOut(f *os.File, r io.Reader, c chan string, wg *sync.WaitGroup) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		s := string(scanner.Bytes())
		if f != nil {
			fmt.Fprintln(f, s)
		}
		if !strings.HasPrefix(s, "[LOG]") && c == nil {
			fmt.Println(s)
		}
		if c != nil {
			c <- s
		}
	}
	if scanner.Err() != nil {
		fmt.Fprintf(os.Stderr, "Error reading output of go test: %v", scanner.Err())
		os.Exit(1)
	}
	close(c)
	wg.Done()
}

func printErr(f *os.File, r io.Reader, c chan string, wg *sync.WaitGroup) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		s := string(scanner.Bytes())
		if f != nil {
			fmt.Fprintln(f, s)
		}
		fmt.Fprintln(os.Stderr, s)
		if c != nil {
			c <- s
		}

	}
	if scanner.Err() != nil {
		fmt.Fprintf(os.Stderr, "Error reading output of go test: %v", scanner.Err())
		os.Exit(1)
	}
	close(c)
	wg.Done()
}
