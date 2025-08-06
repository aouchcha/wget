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
		HandleBackgroundDownloaded(logFile, )
		// var err error
		// logFile, err = Create_Output_file(false, "wget-log")
		// if err != nil {
		// 	return fmt.Errorf("failed to create log file: %v", err)
		// }
		// defer logFile.Close()

		// logger = log.New(logFile, "", 0)
		
		// fmt.Printf(`Output will be written to '%s'.`,logFile.Name())

		// // Log start time
		// logger.Printf("start at %s", time.Now().Format("2006-01-02 15:04:05"))
	}else if args.InputFile != "" {
		// Batch download from file
		// return args.executeBatchDownload(logger)
	} else if args.isMirror {
		fmt.Println("++++++++++++++++++++++++++++++",args)
		for _, link := range args.Links {

			args.NewMirrorConfig(link)

			if !args.Background {
				logStart(link)
			}
			if err := args.ParseAndDownload(link); err != nil {
				logError(err.Error())
				os.Exit(1)
			}
			if !args.Background {
				logFinish(link)
			}
		}
		// return nil
	} else {
	fmt.Println("hhhhhhhhhhhhhhhhhhhhhhhhhhh")

		// Single file download
		return DownloadOneSource(args, logger)
	}
	// if args.Background {
	// 	fmt.Printf("Output will be written to '%s'.\n", logFile.Name())
	// }
	return nil
}
