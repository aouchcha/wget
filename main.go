package main

import (
	"fmt"
	"os"
	// "os/signal"
	// "syscall"
)

type FlagsComponents struct {
	Links       []string
	InputFile  string
	OutputFile string
	PathFile   string
	RateLimite string
	Exclude    []string
	Reject     []string
	isMirror   bool
	Background bool
	Convert    bool
}

func main() {
	// // Handle SIGPIPE gracefully
	// signal.Ignore(syscall.SIGPIPE)

	// Your existing code...
	args := os.Args[1:]
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage go run . link to download \n go run -O=filename link")
		return
	}
	components := FlagsComponents{}
	err := parsing(args, &components)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	err = DownloadFiles(&components)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
}
