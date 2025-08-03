package main

import (
	"errors"
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


func (c *FlagsComponents) DownloadOneSource(logger *log.Logger) error {
	filename := c.OutputFile
	Overide := true
	if c.OutputFile == "" {
		Overide = false
		filename = GetOutputFromUrl(c.Link)
	}

	if c.PathFile != "" && c.OutputFile == "" {
		filename = filepath.Join(c.PathFile, filename)
		if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
			return fmt.Errorf("failed to create directory: %v", err)
		}
	}
	err := Download(c, filename, logger, Overide)
	if err != nil {
		return err
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

func Download(c *FlagsComponents, filename string, logger *log.Logger, Overide bool) error {
	// Print timestamp and URL
	logOrPrint(logger, c.Background, fmt.Sprintf("--%s--  %s", time.Now().Format("2006-01-02 15:04:05"), c.Link))

	// Parse URL to get host
	url, err := url.Parse(c.Link)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %v", err)
	}
	response, err := http.Get(c.Link)
	if err != nil {
		return err
	}

	if response.StatusCode != 200 {
		return errors.New("bad status")
	}
	
	// Resolve hostname (simulate DNS lookup)
	if !c.Background {
		fmt.Printf("Resolving %s (%s)... ", url.Host, url.Host)
	} else {
		logOrPrint(logger, c.Background, fmt.Sprintf("Resolving %s (%s)...", url.Host, url.Host))
	}

	// Look up IP address
	ips, err := net.LookupIP(url.Host)
	if err != nil {
		if !c.Background {
			fmt.Printf("failed\n")
		}
		logOrPrint(logger, c.Background, "DNS resolution failed")
		return fmt.Errorf("failed to resolve hostname: %v", err)
	}

	// Print first IP
	if len(ips) > 0 {
		if !c.Background {
			fmt.Printf("%s\n", ips[0].String())
		} else {
			logOrPrint(logger, c.Background, fmt.Sprintf("Resolved to: %s", ips[0].String()))
		}
	}

	// Print connecting message
	port := "80"
	if url.Scheme == "https" {
		port = "443"
	}
	
	if !c.Background {
		fmt.Printf("Connecting to %s (%s)|%s|:%s... ", url.Host, url.Host, ips[0].String(), port)
	} else {
		logOrPrint(logger, c.Background, fmt.Sprintf("Connecting to %s (%s)|%s|:%s...", url.Host, url.Host, ips[0].String(), port))
	}

	startTime := time.Now()
	defer response.Body.Close()
	
	if !c.Background {
		fmt.Printf("connected.\n")
	} else {
		logOrPrint(logger, c.Background, "Connected successfully")
	}

	// Print HTTP request status
	logOrPrint(logger, c.Background, fmt.Sprintf("HTTP request sent, awaiting response... %s", response.Status))

	// Print content length
	fileSize := response.ContentLength
	contentType := response.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	if fileSize > 0 {
		logOrPrint(logger, c.Background, fmt.Sprintf("Length: %d [%s]", fileSize, contentType))
	} else {
		logOrPrint(logger, c.Background, fmt.Sprintf("Length: unspecified [%s]", contentType))
	}

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// Create output file and ovrid the old
	var out *os.File
	if Overide {
		out, err = os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create file: %v", err)
		}
	} else {
		_, err := os.Stat(filename)
		if err != nil {
			if os.IsNotExist(err) {
				out, err = os.Create(filename)
				if err != nil {
					return fmt.Errorf("failed to create file: %v", err)
				}
			} else {
				return err
			}
		} else {
			// File exists - create with number suffix
			dir := filepath.Dir(filename)
			ext := filepath.Ext(filename)
			base := strings.TrimSuffix(filepath.Base(filename), ext)
			i := 1
			for {
				if ext != "" {
					filename = filepath.Join(dir, fmt.Sprintf("%s.%d%s", base, i, ext))
				} else {
					filename = filepath.Join(dir, fmt.Sprintf("%s.%d", base, i))
				}

				// Check if this numbered version exists
				if _, err := os.Stat(filename); os.IsNotExist(err) {
					// This name is available
					out, err = os.Create(filename)
					if err != nil {
						return fmt.Errorf("failed to create file: %v", err)
					}
					logOrPrint(logger, c.Background, fmt.Sprintf("File renamed to: '%s'", filepath.Base(filename)))
					break
				}
				i += 1
			}
		}
	}
	defer out.Close()

	logOrPrint(logger, c.Background, fmt.Sprintf("Saving to: '%s'\n", filepath.Base(filename)))

	// Download with progress - ALWAYS show progress unless in background mode
	var downloaded int64
	if c.RateLimite > 0 {
		// If you have rate limiting implementation
		// downloaded, err = io.Copy(out, response.Body)
	} else {
		if !c.Background {
			// Show progress regardless of whether we know file size
			downloaded, err = copyWithProgress(response.Body, out, fileSize, filepath.Base(filename), logger, c.Background)
		} else {
			// implement the code for -B flag
			downloaded, err = io.Copy(out, response.Body)
		}
	}

	if err != nil {
		return fmt.Errorf("download failed: %v", err)
	}

	// Calculate download speed and time
	duration := time.Since(startTime)
	speed := float64(downloaded) / duration.Seconds() / (1024 * 1024) // MB/s

	// Print completion message
	logOrPrint(logger, c.Background, fmt.Sprintf("\n%s (%s) - '%s' saved [%d]",
		time.Now().Format("2006-01-02 15:04:05"),
		formatSpeed(speed),
		filepath.Base(filename),
		downloaded))

	return nil
}

func copyWithProgress(src io.Reader, dst io.Writer, total int64, filename string, logger *log.Logger, background bool) (int64, error) {
	var written int64
	buf := make([]byte, 32*1024) // 32KB buffer
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
			if now.Sub(lastUpdate) > 1000*time.Millisecond || err == io.EOF {
				showProgress(written, total, filename, time.Since(startTime), logger, background)
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
	showProgress(written, total, filename, time.Since(startTime), logger, background)
	return written, nil
}

// Helper function to show progress
func showProgress(downloaded, total int64, filename string, duration time.Duration, logger *log.Logger, background bool) {
	// If in background mode, log progress periodically instead of showing progress bar
	if background {
		// Log progress every MB or when complete
		if downloaded%(1024*1024) == 0 || (total > 0 && downloaded >= total) {
			speed := float64(downloaded) / duration.Seconds() / (1024 * 1024)
			logOrPrint(logger, background, fmt.Sprintf("Downloaded: %.2fMB, Speed: %.2fMB/s", 
				float64(downloaded)/(1024*1024), speed))
		}
		return
	}

	// Foreground mode - show progress bar
	// Avoid division by zero
	if duration.Seconds() <= 0 {
		duration = 1 * time.Millisecond
	}

	// Calculate speed in MB/s with proper bounds checking
	speed := float64(downloaded) / duration.Seconds() / (1024 * 1024)
	
	// Cap extremely high speeds that cause scientific notation
	if speed > 999.99 {
		speed = 999.99
	}
	if speed < 0.01 {
		speed = 0.01
	}

	// Truncate filename if too long (like wget does)
	displayName := filename
	if len(displayName) > 20 {
		displayName = displayName[:17] + "..."
	}

	// Create progress bar similar to wget
	barWidth := 70  // Reduced width to fit better
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

	// Format similar to wget output - FIXED the size display
	fmt.Printf("\r%-20s [%s] %6.2fM %6.2fMB/s",
		displayName,
		progressBar,
		float64(downloaded)/(1024*1024), // FIXED: Show downloaded size in MB
		speed)

	// Add timing info
	if total > 0 && downloaded >= total {
		// Complete - show "in Xs"
		fmt.Printf(" in %.1fs", duration.Seconds())
	} else if total > 0 && downloaded < total && speed > 0 {
		// Show ETA
		remaining := total - downloaded
		eta := time.Duration(float64(remaining)/speed/1024/1024) * time.Second
		fmt.Printf(" eta %.1fs", eta.Seconds())
	}

	// Force flush - FIXED: Use proper flush
	os.Stdout.Sync()
}

func formatSpeed(speedMBps float64) string {
	// Cap extremely high speeds to avoid scientific notation
	if speedMBps > 999.99 {
		speedMBps = 999.99
	}
	if speedMBps < 0.001 {
		speedMBps = 0.001
	}
	
	// Fixed logic: show KB/s when speed is LESS than 1 MB/s
	if speedMBps < 1 {
		return fmt.Sprintf("%.0f KB/s", speedMBps*1024)
	}
	return fmt.Sprintf("%.2f MB/s", speedMBps)
}