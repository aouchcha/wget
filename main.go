package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

func main() {
	var (
		outputFlag    = flag.String("O", "", "output file name")
		dirFlag       = flag.String("P", ".", "save directory")
		bgFlag        = flag.Bool("B", false, "run in background")
		rateLimitFlag = flag.String("rate-limit", "", "limit download speed (e.g. 200k, 2M)")
		inputFlag     = flag.String("i", "", "input file with URLs")
		mirrorFlag    = flag.Bool("mirror", false, "mirror website")
		rejectFlag    = flag.String("reject", "", "comma-separated extensions to reject")
		excludeFlag   = flag.String("X", "", "comma-separated paths to exclude")
		convertFlag   = flag.Bool("convert-links", false, "convert links for offline viewing")
	)

	flag.Parse()

	args := flag.Args()
	if len(args) == 0 && *inputFlag == "" && !*mirrorFlag {
		fmt.Fprintln(os.Stderr, "Usage: go-wget [options] <url>")
		os.Exit(1)
	}

	if *bgFlag {
		logFile, err := openLogFile()
		if err != nil {
			logError(err.Error())
			os.Exit(1)
		}
		defer logFile.Close()
		Stdout = io.MultiWriter(logFile)
		Stderr = io.MultiWriter(logFile)
		logBackground()
	}

	// Handle -i: download multiple URLs from file
	if *inputFlag != "" {
		lines, err := readLines(*inputFlag)
		if err != nil {
			logError(err.Error())
			os.Exit(1)
		}
		var wg sync.WaitGroup
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			wg.Add(1)
			go func(url string) {
				defer wg.Done()
				dl := NewDownloader(url)
				dl.DestDir = *dirFlag
				if *rateLimitFlag != "" {
					dl.SetRateLimit(*rateLimitFlag)
				}
				if !*bgFlag {
					logStart(url)
				}
				if err := dl.Download(); err != nil {
					logError(err.Error())
				}
			}(line)
		}
		wg.Wait()
		fmt.Fprintf(Stdout, "\nDownload finished: %v\n", lines)
		return
	}

	// Handle --mirror
	if *mirrorFlag && len(args) > 0 {
		config, err := NewMirrorConfig(args[0])
		if err != nil {
			logError(err.Error())
			os.Exit(1)
		}
		config.Reject = strings.Split(*rejectFlag, ",")
		if *rejectFlag == "" {
			config.Reject = []string{}
		}
		config.Exclude = strings.Split(*excludeFlag, ",")
		if *excludeFlag == "" {
			config.Exclude = []string{}
		}
		config.Convert = *convertFlag

		if !*bgFlag {
			logStart(args[0])
		}
		if err := config.ParseAndDownload(args[0]); err != nil {
			logError(err.Error())
			os.Exit(1)
		}
		if !*bgFlag {
			logFinish(args[0])
		}
		return
	}

	// Normal single download
	if len(args) == 0 {
		return
	}
	url := args[0]

	dl := NewDownloader(url)
	dl.SetOutput(*outputFlag)
	dl.SetDir(*dirFlag)
	if *rateLimitFlag != "" {
		if err := dl.SetRateLimit(*rateLimitFlag); err != nil {
			logError(err.Error())
			os.Exit(1)
		}
	}
	dl.Background = *bgFlag

	if !*bgFlag {
		logStart(url)
	}
	if err := dl.Download(); err != nil {
		logError(err.Error())
		os.Exit(1)
	}
}

func readLines(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	buf := make([]byte, 4096)
	var lineBuilder strings.Builder

	for {
		n, err := file.Read(buf)
		if n > 0 {
			lineBuilder.Write(buf[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	content := lineBuilder.String()
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
}
