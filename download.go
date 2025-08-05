package main

import (
	"fmt"
	"log"
	"os"
	"time"
)

func DownloadFiles(args *FlagsComponents) error {
	if err := args.Validate(); err != nil {
		return err
	}
	// Setup logging for background mode
	var logger *log.Logger
	var logFile *os.File

	if args.Background {
		var err error
		logFile, err = os.Create("wget-log")
		if err != nil {
			return fmt.Errorf("failed to create log file: %v", err)
		}
		defer logFile.Close()

		logger = log.New(logFile, "", 0)
		fmt.Println(`Output will be written to "wget-log".`)

		// Log start time
		logger.Printf("start at %s", time.Now().Format("2006-01-02 15:04:05"))
		
	}

	// Choose execution path based on flags
	if args.InputFile != "" {
		// Batch download from file
		// return args.executeBatchDownload(logger)
	} else if args.isMirror {
		// Mirror website
		// return args.executeMirrorDownload(logger)
	} else {
		// Single file download
		return DownloadOneSource(args, logger)
	}
	return nil
}
