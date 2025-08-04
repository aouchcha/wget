package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Downloader struct {
	URL        string
	Output     string
	DestDir    string
	RateLimit  int64 // bytes per second
	Background bool
	Client     *http.Client
}

func NewDownloader(url string) *Downloader {
	return &Downloader{
		URL:        url,
		DestDir:    ".",
		RateLimit:  0,
		Background: false,
		Client:     &http.Client{Timeout: 30 * time.Second},
	}
}

func (d *Downloader) SetOutput(name string) {
	d.Output = name
}

func (d *Downloader) SetDir(dir string) {
	d.DestDir = dir
}

func (d *Downloader) SetRateLimit(rateStr string) error {
	re := regexp.MustCompile(`^(\d+)([kKmM])?$`)
	parts := re.FindStringSubmatch(rateStr)
	if len(parts) != 3 {
		return fmt.Errorf("invalid rate limit: %s", rateStr)
	}
	val, _ := strconv.ParseInt(parts[1], 10, 64)
	unit := strings.ToLower(parts[2])
	switch unit {
	case "k":
		d.RateLimit = val * 1024
	case "m":
		d.RateLimit = val * 1024 * 1024
	case "":
		d.RateLimit = val
	default:
		return fmt.Errorf("unsupported unit: %s", unit)
	}
	return nil
}

func (d *Downloader) buildOutputPath() (string, error) {
	u, err := url.Parse(d.URL)
	if err != nil {
		return "", err
	}

	// If -O was used, save exactly as specified, inside DestDir
	if d.Output != "" {
		return filepath.Join(d.DestDir, d.Output), nil
	}

	// Extract path from URL
	path := u.Path

	// If path is empty or ends with '/', treat as index.html
	if path == "" || path[len(path)-1] == '/' {
		path += "index.html"
	}

	// Clean the path to resolve any '..' or '.' segments safely
	path = filepath.Clean("/" + path)

	// Construct final path: <DestDir>/<domain>/<cleaned_path>
	return filepath.Join(d.DestDir, u.Host, path), nil
}

func (d *Downloader) Download() error {
	resp, err := d.Client.Get(d.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logRequest(fmt.Sprintf("%d %s", resp.StatusCode, resp.Status))
		return fmt.Errorf("bad status: %s", resp.Status)
	}
	logRequest("200 OK")

	contentLength := resp.ContentLength
	logSize(contentLength)

	outputPath, err := d.buildOutputPath()
	if err != nil {
		return err
	}
	logSaving(outputPath)

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	var written int64
	startTime := time.Now()
	ticker := time.NewTicker(100 * time.Millisecond)
	done := make(chan bool)

	go func() {
		defer close(done)
		for {
			select {
			case <-ticker.C:
				elapsed := time.Since(startTime).Seconds()
				var speed float64
				if elapsed > 0 {
					speed = float64(written) / elapsed
				}
				logProgress(written, contentLength, speed, time.Since(startTime))
			case <-done:
				return
			}
		}
	}()

	buffer := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			if d.RateLimit > 0 {
				delay := time.Second * time.Duration(n) / time.Duration(d.RateLimit)
				time.Sleep(delay)
			}
			if _, werr := outFile.Write(buffer[:n]); werr != nil {
				ticker.Stop()
				done <- true
				return werr
			}
			written += int64(n)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			ticker.Stop()
			done <- true
			return err
		}
	}

	ticker.Stop()
	done <- true

	// Final update
	elapsed := time.Since(startTime).Seconds()
	var speed float64
	if elapsed > 0 {
		speed = float64(written) / elapsed
	}
	logProgress(written, contentLength, speed, time.Since(startTime))
	logFinish(d.URL)

	return nil
}
