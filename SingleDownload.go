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
	port := "80"
	if url.Scheme == "https" {
		port = "443"
	}
	logOrPrint(logger, c.Background, fmt.Sprintf("Connecting to %s (%s)|%s|:%s...", url.Host, url.Host, ips[0].String(), port))

	startTime := time.Now()
	defer response.Body.Close()

	logOrPrint(logger, c.Background, " connected.\n")
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

	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	OutputFile, err := Create_Output_file(Overide, filename)
	if err != nil {
		return err
	}
	defer OutputFile.Close()

	logOrPrint(logger, c.Background, fmt.Sprintf("Saving to: '%s'\n", filepath.Base(filename)))
	rate, err := parseRateLimit(c.RateLimite)
	if err != nil {
		return err
	}
	var downloaded int64
	if rate > 0 {
		downloaded, err = copyWithRateLimit(response.Body, OutputFile, rate, fileSize, filename, logger, c.Background, Link)
	} else {
		downloaded, err = copyWithProgress(response.Body, OutputFile, fileSize, filepath.Base(filename), logger, c.Background, Link)
	}

	if err != nil {
		return fmt.Errorf("download failed: %v", err)
	}

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
			now := time.Now()
			// here
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
	speed := float64(downloaded) / duration.Seconds() / (1024 * 1024) // MB/s
	barWidth := 80
	var progressBar string

	if total > 0 {
		percentage := float64(downloaded) / float64(total) * 100
		filled := int(percentage / 100 * float64(barWidth))
		progressBar = strings.Repeat("=", filled)
		if filled < barWidth {
			progressBar += ">"
			progressBar += strings.Repeat(" ", barWidth-filled-1)
		}
	} else {
		pos := int(time.Now().UnixMilli()/100) % (barWidth - 6)
		progressBar = strings.Repeat(" ", pos) + "  <=>  " + strings.Repeat(" ", barWidth-pos-6)
	}

	var filesize string
	if downloaded/(1024*1024) > 1 {
		filesize = fmt.Sprintf("%.2fM", float64(downloaded)/(1024*1024))
	} else {
		filesize = fmt.Sprintf("%.2fK", float64(downloaded)/(1024))
	}

	if background {
		remaining := total - downloaded
		remainSeconds := float64(remaining) / (speed * 1024 * 1024) // Convert MB/s to bytes/s
		remainDuration := time.Duration(remainSeconds * float64(time.Second))
		remainingStr := formatETA(remainDuration)

		logOrPrint(logger, background, fmt.Sprintf("%dK %s %.0f%% %s %s", downloaded/1024, strings.ReplaceAll(progressBar, "=", "."), float64(downloaded*100)/float64(total), formatSpeed(speed), remainingStr))
	} else {
		logOrPrint(logger, background, fmt.Sprintf("\r\033[K%s %.0f%% [%s] %s %s ", filename, float64(downloaded*100)/float64(total), progressBar, filesize, formatSpeed(speed)))
	}

	if total > 0 && downloaded >= total {
		logOrPrint(logger, background, fmt.Sprintf(" in %.2fs", duration.Seconds()))
	} else if total > 0 && downloaded < total && speed > 0 {
		remainSeconds := float64(total-downloaded) / (speed * 1024 * 1024)
		remainDuration := time.Duration(remainSeconds * float64(time.Second))
		logOrPrint(logger, background, fmt.Sprintf(" at %s", formatETA(remainDuration)))
	}
	// os.Stdout.Sync()
}

