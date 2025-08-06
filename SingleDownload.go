package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// func LegacyParsing() bool {
// 	args := os.Args[1:]

// 	// Old -B behavior
// 	if contains(args, "-B") {
// 		HandleBackgroundDownload())
// 		return true
// 	}

// 	// Old -i behavior
// 	for i := 0; i < len(args); i++ {
// 		if strings.HasPrefix(args[i], "-i") {
// 			var filePath string
// 			if strings.Contains(args[i], "=") {
// 				parts := strings.SplitN(args[i], "=", 2)
// 				if len(parts) != 2 || parts[1] == "" {
// 					fmt.Fprintln(os.Stderr, "Invalid -i flag format")
// 					return true
// 				}
// 				filePath = parts[1]
// 			} else if i+1 < len(args) {
// 				filePath = args[i+1]
// 			} else {
// 				fmt.Fprintln(os.Stderr, "Missing file after -i flag")
// 				return true
// 			}
// 			HandleMultipleDownloads(filePath)
// 			return true
// 		}
// 	}
// 	return false
// }

func DownloadOneSource(c *FlagsComponents, logger *log.Logger) error {
	for _, link := range c.Links {
		filename := c.OutputFile
		Overide := true
		if c.OutputFile == "" {
			Overide = false
			filename = GetOutputFromUrl(link)
		}

		if c.PathFile != "" {
			filename = filepath.Join(c.PathFile, filename)
			if strings.HasPrefix(filename, "~") {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("failed to get the home directory: %v", err)
				}
				filename = strings.ReplaceAll(filename, "~", homeDir)
			}
			if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
				return fmt.Errorf("failed to create directory: %v", err)
			}
		}

		err := Download(link, c, filename, logger, Overide)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}

	}

	return nil
}

func GetOutputFromUrl(Link string) string {
	sli := strings.Split(Link, "/")
	filename := sli[len(sli)-1]
	if filename == "" {
		filename = "index.html"
	}
	return filename
}

func Download(Link string, c *FlagsComponents, filename string, logger *log.Logger, Overide bool) error {
	// Print timestamp and URL
	logOrPrint(logger, c.Background, fmt.Sprintf("--%s--  %s\n", time.Now().Format("2006-01-02 15:04:05"), Link))

	// Parse URL to get host
	url, err := url.Parse(Link)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %v", err)
	}
	response, err := http.Get(Link)
	if err != nil {
		return err
	}

	// Get the host name

	// Look up IP address
	ips, err := net.LookupIP(url.Host)
	if err != nil {
		logOrPrint(logger, c.Background, "DNS resolution failed")
		return fmt.Errorf("failed to resolve hostname: %v", err)
	}
	// Print all the ips
	var IpsTotal []string
	for _, ip := range ips {
		IpsTotal = append(IpsTotal, ip.String())
	}
	IpStr := strings.Join(IpsTotal, ", ")
	logOrPrint(logger, c.Background, fmt.Sprintf("Resolving %s (%s)... %s\n", url.Host, url.Host, IpStr))
	// if len(ips) > 0 {
	// 	logOrPrint(logger, c.Background, fmt.Sprintf("Resolved to: %s", IpStr))
	// }

	// Print connecting message
	port := "80"
	if url.Scheme == "https" {
		port = "443"
	}
	// if !c.Background {
	// 	fmt.Printf("Connecting to %s (%s)|%s|:%s...", url.Host, url.Host, ips[0].String(), port)
	// } else {
	logOrPrint(logger, c.Background, fmt.Sprintf("Connecting to %s (%s)|%s|:%s...", url.Host, url.Host, ips[0].String(), port))
	// }

	startTime := time.Now()
	defer response.Body.Close()

	// if !c.Background {
	// 	fmt.Printf(" connected.")
	// } else {
	logOrPrint(logger, c.Background, " connected.\n")
	// }

	// Print HTTP request status
	logOrPrint(logger, c.Background, fmt.Sprintf("HTTP request sent, awaiting response... %s\n", response.Status))
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("--%s--  Error %d: %s", time.Now().Format("2006-01-02 15:04:05"), response.StatusCode, response.Status)
	}

	// Print content length
	fileSize := response.ContentLength
	contentType := response.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	if fileSize > 0 {
		logOrPrint(logger, c.Background, fmt.Sprintf("Length: %d [%s]\n", fileSize, contentType))
	} else {
		logOrPrint(logger, c.Background, fmt.Sprintf("Length: unspecified [%s]\n", contentType))
	}

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// Create output file and ovrid the old if needed
	OutputFile, err := Create_Output_file(Overide, filename)
	if err != nil {
		return err
	}
	defer OutputFile.Close()

	logOrPrint(logger, c.Background, fmt.Sprintf("Saving to: '%s'\n", filepath.Base(filename)))

	// Download with progress - ALWAYS show progress unless in background mode
	rate, err := parseRateLimit(c.RateLimite)
	if err != nil {
		return err
	}
	var downloaded int64
	if rate > 0 {
		downloaded, err = copyWithRateLimit(response.Body, OutputFile, rate, fileSize, filename, logger, c.Background, Link)
	} else {
		fmt.Println()
		fmt.Println("wa zaaaabi")
		// Show progress regardless of whether we know file size
		downloaded, err = copyWithProgress(response.Body, OutputFile, fileSize, filepath.Base(filename), logger, c.Background, Link)
	}

	if err != nil {
		return fmt.Errorf("download failed: %v", err)
	}

	// Calculate download speed and time
	duration := time.Since(startTime)
	speed := float64(downloaded) / duration.Seconds() / (1024 * 1024) // MB/s

	logOrPrint(logger, c.Background, fmt.Sprintf("%s (%s) - '%s' saved [%d]\n",
		time.Now().Format("2006-01-02 15:04:05"),
		formatSpeed(speed),
		filepath.Base(filename),
		downloaded))

	return nil
}

func copyWithProgress(src io.Reader, dst io.Writer, total int64, filename string, logger *log.Logger, background bool, Link string) (int64, error) {
	var written int64
	buf := make([]byte, 32*1024)

	startTime := time.Now()
	lastUpdate := time.Now()

	for {
		number_of_bytes_readed, err := src.Read(buf)
		if number_of_bytes_readed > 0 {
			number_of_byte_writed, err2 := dst.Write(buf[0:number_of_bytes_readed])
			if number_of_byte_writed > 0 {
				written += int64(number_of_byte_writed)
			}
			if err2 != nil {
				return written, err2
			}
			if number_of_bytes_readed != number_of_byte_writed {
				return written, io.ErrShortWrite
			}

			// Update progress more frequently - every 10ms or when finished
			now := time.Now()
			//here
			if now.Sub(lastUpdate) > 10*time.Millisecond || err == io.EOF {
				showProgress(written, total, filename, time.Since(startTime), logger, background, Link)
				lastUpdate = now
			}
		}
		if err != nil {
			if err != io.EOF {
				return written, err
			}

			break
		}
	}

	// Final progress update
	if !background {
		showProgress(written, total, filename, time.Since(startTime), logger, background, Link)
	}
	fmt.Println()
	return written, nil
}

func showProgress(downloaded, total int64, filename string, duration time.Duration, logger *log.Logger, background bool, Link string) {
	// If in background mode, log progress periodically instead of showing progress bar
	if background {
		fmt.Println(background)
		fmt.Println("hqqqqqqqqqqqqni")
		// Log progress every MB or when complete
		HandleBackgroundDownloaded(Link, logger)
		return
	}

	speed := float64(downloaded) / duration.Seconds() / (1024 * 1024)

	// Create progress bar similar to wget
	barWidth := 80 // Reduced width to fit better
	var progressBar string

	if total > 0 {
		// Known file size - show normal progress bar
		percentage := float64(downloaded) / float64(total) * 100
		filled := int(percentage / 100 * float64(barWidth))

		progressBar = strings.Repeat("=", filled)
		if filled < barWidth {
			progressBar += ">"
			progressBar += strings.Repeat(" ", barWidth-filled-1)
		}
	} else {
		// Unknown file size - show indeterminate progress (like wget's <=>)
		// Create a moving indicator
		pos := int(time.Now().UnixMilli()/100) % (barWidth - 6)
		progressBar = strings.Repeat(" ", pos) + "  <=>  " + strings.Repeat(" ", barWidth-pos-6)
	}

	// downloaded file size
	var filesize string
	if downloaded/(1024*1024) > 1 {
		filesize = fmt.Sprintf("%.2fM", float64(downloaded)/(1024*1024))
	} else {
		filesize = fmt.Sprintf("%.2fK", float64(downloaded)/(1024))
	}
	// fmt.Println("bbbbbbbbbb", background)
	// if !background {
	// 	fmt.Printf("\r%-20s [%s] %s %s", filename, progressBar, filesize, formatSpeed(speed))
	// 	// fmt.Printf("\r %s %s %s %s", filename, progressBar, filesize, formatSpeed(speed))
	// } else {
	// fmt.Println("hanni")
	// fmt.Println("")
	if background {
		remaining := total - downloaded
		// remaining_Sec := time.Duration(float64(remaining)/speed) * time.Second
		// fmt.Println(time.Duration(float64(remaining)/speed).Seconds() * 10)
		remainingStr := formatETA(time.Duration(float64(remaining)/speed) * 10)
		logOrPrint(logger, background, fmt.Sprintf("%dK %s %.0f%% %s %s", downloaded, strings.ReplaceAll(progressBar, "=", "."), float64(downloaded*100)/float64(total), formatSpeed(speed), remainingStr))
	} else {
		logOrPrint(logger, background, fmt.Sprintf("\r%s %.0f%% [%s] %s %s", filename, float64(downloaded*100)/float64(total), progressBar, filesize, formatSpeed(speed)))
	}
	// }

	// Add timing info
	if total > 0 && downloaded >= total {
		// Complete - show "in Xs"
		// if !background {
		// 	fmt.Printf(" in %.1fs", duration.Seconds())
		// } else {
		// if {
		logOrPrint(logger, background, fmt.Sprintf(" in %.2fs", duration.Seconds()))
		// }
		// }
	}
	// else if total > 0 && downloaded < total && speed > 0 {
	// 	// Show ETA
	// 	// remaining := total - downloaded
	// 	// eta := time.Duration(float64(remaining)/speed) * 1000
	// 	// if !background {
	// 	// 	fmt.Printf(" eta %.1f", eta.Seconds())
	// 	// } else {
	// 	// logOrPrint(logger, background, fmt.Sprintf(" in %.1fs", duration.Seconds()))
	// 	// if background {
	// 	// 	logOrPrint(logger, background, fmt.Sprintf(" %.2f", eta.Seconds()))
	// 	// }
	// 	// }
	// }

	os.Stdout.Sync()
}
