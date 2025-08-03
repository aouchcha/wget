package main
import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func DownloadFileWithInfo(url string) error {
	start := time.Now()
	fmt.Println("start at", start.Format("2006-01-02 15:04:05"))

	fmt.Print("sending request, awaiting response... ")
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	fmt.Println("status", resp.Status)

	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status code: %s", resp.Status)
	}

	size := resp.ContentLength
	sizeMB := float64(size) / 1024.0 / 1024.0
	fmt.Printf("content size: %d [~%.2fMB]\n", size, sizeMB)

	fileName := getFileNameFromURL(url)
	fmt.Printf("saving file to: ./%s\n", fileName)

	out, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("%d bytes downloaded\n", written)
	fmt.Printf("Downloaded [%s]\n", url)
	fmt.Println("finished at", time.Now().Format("2006-01-02 15:04:05"))

	return nil
}

func getFileNameFromURL(url string) string {
	parts := strings.Split(url, "/")
	return parts[len(parts)-1]
}
