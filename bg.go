package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func HandleBackgroundDownload(args []string) {
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to locate binary:", err)
		return
	}

	// Remove the -B flag so the child doesn’t re-trigger background logic
	cleanArgs := []string{}
	for _, arg := range args {
		if arg != "-B" && !strings.HasPrefix(arg, "-B=") {
			cleanArgs = append(cleanArgs, arg)
		}
	}

	cmd := exec.Command(exe, cleanArgs...)
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

	fmt.Printf("Continuing in background, pid %d.\n", cmd.Process.Pid)
	fmt.Println(`Output will be written to "wget-log".`)
}
