package main

import (
	"fmt"
	"os"
	"os/exec"
)

func HandleBackgroundDownload(Link string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	

	cmd := exec.Command(exe, Link)
	logFile, err := Create_Output_file(false, "wget-log")
	if err != nil {
		// fmt.Fprintln(os.Stderr, "Failed to create wget-log:", err)
		return fmt.Errorf("%s", err)
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	err = cmd.Start()
	if err != nil {
		// fmt.Fprintln(os.Stderr, "Failed to run in background:", err)
		return fmt.Errorf("%s", err)

	}

	fmt.Printf("Continuing in background, pid %d.\n", cmd.Process.Pid)
	fmt.Printf("Output will be written to '%s'.\n", logFile.Name())
	return nil
}
