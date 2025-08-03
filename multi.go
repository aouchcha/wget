package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

func HandleMultipleDownloads(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to open URL list:", err)
		return
	}
	defer file.Close()

	var wg sync.WaitGroup
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		url := strings.TrimSpace(scanner.Text())
		if url == "" {
			continue
		}

		wg.Add(1)
		go func(link string) {
			defer wg.Done()
			err := DownloadFile(link, "")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error downloading %s: %v\n", link, err)
			}
		}(url)
	}

	wg.Wait()
	fmt.Println("Download finished from list:", filePath)
}

func DownloadFile(url, outputPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// extract filename from URL
	parts := strings.Split(url, "/")
	fileName := parts[len(parts)-1]
	if outputPath != "" {
		fileName = filepath.Join(outputPath, fileName)
	}

	out, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Println("Downloaded:", url)
	return nil
}
