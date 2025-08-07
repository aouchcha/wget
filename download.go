package main

import (
	"log"
	"os"
)

func DownloadFiles(args *FlagsComponents) error {
	if err := args.Validate(); err != nil {
		return err
	}
	var logger *log.Logger
	if args.Background {
		for _, Link := range args.Links {
			err := HandleBackgroundDownload(Link)
			if err != nil {
				return err
			}
		}
		return nil
	} else if args.InputFile != "" {
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
			if err := args.crawl(link, 0); err != nil {
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
