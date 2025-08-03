package main

import (
	"fmt"
	"os"
)

type FlagsComponents struct {
	Link       string
	InputFile  string
	OutputFile string
	PathFile   string
	RateLimite int
	Exclude    []string
	Reject     []string
	isMirror   bool
	Background bool
	Convert    bool
}

func main() {
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
	err = DownloadFile(&components)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
}
