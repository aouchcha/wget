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

func HandleMultipleDownloads(filePath string) error{
	file, err := os.Open(filePath)
	if err != nil {
		// fmt.Fprintln(os.Stderr, "Failed to open URL list:", err)
		return fmt.Errorf("%s",err)
	}
	defer file.Close()

	var urls []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		link := strings.TrimSpace(scanner.Text())
		if link != "" {
			urls = append(urls, link)
		}
	}

	var sizes []int64
	for _, link := range urls {
		resp, err := http.Head(link)
		if err != nil {
			sizes = append(sizes, -1)
			continue
		}
		sizes = append(sizes, resp.ContentLength)
		resp.Body.Close()
	}
	fmt.Printf("content size: %v\n", sizes)

	var wg sync.WaitGroup
	for _, link := range urls {
		wg.Add(1)
		go func(l string) error {
			defer wg.Done()
			err := DownloadFile(l, "")
			if err != nil {
				// fmt.Fprintf(os.Stderr, "Error downloading %s: %v\n", l, err)
				return fmt.Errorf("%s",err)
			}
			parts := strings.Split(l, "/")
			fmt.Println("finished", parts[len(parts)-1])
			return nil
		}(link)
	}
	wg.Wait()

	fmt.Println("Download finished: ", urls)
	return nil
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

	return nil
}