package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

func HandleBackgroundDownloaded(Link string, logger *log.Logger) {
	fmt.Println("hqnni")
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to locate binary:", err)
		return
	}
	cmd := exec.Command(exe, Link)
	
	cmd.Stdout = logger.Writer()
	cmd.Stderr = logger.Writer()

	err = cmd.Start()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to run in background:", err)
		return
	}

	fmt.Printf("Continuing in background, pid %d.\n", cmd.Process.Pid)
	// fmt.Println(`Output will be written to "wget-log".`)
}
