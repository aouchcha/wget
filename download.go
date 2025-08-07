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
		logFile, err = Create_Output_file(false, "wget-log")
		if err != nil {
			return fmt.Errorf("failed to create log file: %v", err)
		}
		defer logFile.Close()

		logger = log.New(logFile, "", 0)
		fmt.Printf(`Output will be written to '%s'.`,logFile.Name())
		logger.Printf("start at %s", time.Now().Format("2006-01-02 15:04:05"))
	}

	if args.InputFile != "" {
		// Batch download from file
		err := HandleMultipleDownloads(args.InputFile)
		if err != nil {
			return err
		}
	} else if args.isMirror {
		for _, link := range args.Links {

			args.NewMirrorConfig(link)

			if !args.Background {
				logStart()
			}
			if err := args.crawl(link,0); err != nil {
				logError(err.Error())
				os.Exit(1)
			}
			if !args.Background {
				logFinish(link)
			}
		}
	} else {
		// Single file download
		return DownloadOneSource(args, logger)
	}
	return nil
}
