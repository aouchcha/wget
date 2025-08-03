package main

import (
	"fmt"
	"os"
	"os/exec"
)

func HandleBackgroundDownload(link string) {
	// Use current executable to fork a new process
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to locate binary:", err)
		return
	}

	cmd := exec.Command(exe, link)
	logFile, err := os.Create("wget-log")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to create wget-log:", err)
		return
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	err = cmd.Start()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to run in background:", err)
		return
	}

	fmt.Println(`Output will be written to "wget-log".`)
}
