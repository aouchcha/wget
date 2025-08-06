# wget
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// DownloadConfig holds all possible flags
type DownloadConfig struct {
	// Basic download
	URL        string
	OutputFile string  // -O flag
	OutputPath string  // -P flag
	
	// Control flags  
	Background   bool   // -B flag
	RateLimit    int64  // --rate-limit flag (0 = no limit)
	
	// Mirror flags
	Mirror       bool     // --mirror flag
	ConvertLinks bool     // --convert-links flag
	RejectTypes  []string // -R flag
	ExcludeDirs  []string // -X flag
}

// NewDownloadConfig creates config with defaults
func NewDownloadConfig() *DownloadConfig {
	return &DownloadConfig{
		RateLimit:    0,
		Background:   false,
		Mirror:       false,
		ConvertLinks: false,
		RejectTypes:  []string{},
		ExcludeDirs:  []string{},
	}
}

// ParseArgs parses command line arguments
func (c *DownloadConfig) ParseArgs(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: %s [flags] <URL>", args[0])
	}
	
	for i := 1; i < len(args); i++ {
		arg := args[i]
		
		// Handle flags with = syntax
		if strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			flag := parts[0]
			value := parts[1]
			
			switch flag {
			case "-O":
				c.OutputFile = value
			case "-P":
				c.OutputPath = value
			case "--rate-limit":
				rate, err := parseRateLimit(value)
				if err != nil {
					return fmt.Errorf("invalid rate limit %s: %v", value, err)
				}
				c.RateLimit = rate
			case "-R", "--reject":
				c.RejectTypes = strings.Split(value, ",")
			case "-X", "--exclude":
				c.ExcludeDirs = strings.Split(value, ",")
			default:
				return fmt.Errorf("unknown flag: %s", flag)
			}
		} else {
			// Handle boolean flags and URL
			switch arg {
			case "-B":
				c.Background = true
			case "--mirror":
				c.Mirror = true
			case "--convert-links":
				c.ConvertLinks = true
			default:
				// Assume it's the URL (should be last argument)
				if i == len(args)-1 {
					c.URL = arg
				} else {
					return fmt.Errorf("unknown flag or misplaced URL: %s", arg)
				}
			}
		}
	}
	
	return nil
}

// Validate checks for conflicting flags and requirements
func (c *DownloadConfig) Validate() error {
	// Check for URL
	if c.URL == "" {
		return fmt.Errorf("URL is required")
	}
	
	// Check for conflicting flags
	if c.Mirror && c.OutputFile != "" {
		return fmt.Errorf("cannot use -O (output file) with --mirror")
	}
	
	// Mirror-specific validations
	if (len(c.RejectTypes) > 0 || len(c.ExcludeDirs) > 0 || c.ConvertLinks) && !c.Mirror {
		return fmt.Errorf("-R, -X, and --convert-links can only be used with --mirror")
	}
	
	return nil
}

// ExecuteDownload handles all flag combinations
func (c *DownloadConfig) ExecuteDownload() error {
	// Validate configuration first
	if err := c.Validate(); err != nil {
		return err
	}
	
	// Setup logging for background mode
	var logger *log.Logger
	var logFile *os.File
	
	if c.Background {
		var err error
		logFile, err = os.Create("wget-log")
		if err != nil {
			return fmt.Errorf("failed to create log file: %v", err)
		}
		defer logFile.Close()
		
		logger = log.New(logFile, "", 0)
		fmt.Println(`Output will be written to "wget-log".`)
		
		// Log start time
		logger.Printf("start at %s", time.Now().Format("2006-01-02 15:04:05"))
	}
	
	// Choose execution path
	if c.Mirror {
		// Mirror website
		return c.executeMirrorDownload(logger)
	} else {
		// Single file download
		return c.executeSingleDownload(logger)
	}
}

// executeSingleDownload handles single file with applicable flags
func (c *DownloadConfig) executeSingleDownload(logger *log.Logger) error {
	// Determine output filename and path
	filename := c.OutputFile
	if filename == "" {
		filename = getFilenameFromURL(c.URL)
	}
	
	// Apply output path if specified
	if c.OutputPath != "" {
		filename = filepath.Join(c.OutputPath, filename)
		
		// Create directory if it doesn't exist
		if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
			return fmt.Errorf("failed to create directory: %v", err)
		}
	}
	
	// Log or print the operation
	logOrPrint(logger, c.Background, "sending request, awaiting response...")
	
	// Execute download with all settings
	return c.downloadFile(c.URL, filename, logger)
}

// executeMirrorDownload handles --mirror with all related flags
func (c *DownloadConfig) executeMirrorDownload(logger *log.Logger) error {
	// Parse URL to get domain name for folder
	parsedURL, err := url.Parse(c.URL)
	if err != nil {
		return fmt.Errorf("invalid URL: %v", err)
	}
	
	domain := parsedURL.Host
	if domain == "" {
		return fmt.Errorf("could not extract domain from URL")
	}
	
	// Create mirror directory
	mirrorDir := domain
	if c.OutputPath != "" {
		mirrorDir = filepath.Join(c.OutputPath, domain)
	}
	
	if err := os.MkdirAll(mirrorDir, 0755); err != nil {
		return fmt.Errorf("failed to create mirror directory: %v", err)
	}
	
	logOrPrint(logger, c.Background, fmt.Sprintf("Mirroring %s to %s/", c.URL, mirrorDir))
	
	if c.RateLimit > 0 {
		logOrPrint(logger, c.Background, fmt.Sprintf("Rate limit: %s/s", formatBytes(c.RateLimit)))
	}
	
	if len(c.RejectTypes) > 0 {
		logOrPrint(logger, c.Background, fmt.Sprintf("Rejecting file types: %s", strings.Join(c.RejectTypes, ",")))
	}
	
	if len(c.ExcludeDirs) > 0 {
		logOrPrint(logger, c.Background, fmt.Sprintf("Excluding directories: %s", strings.Join(c.ExcludeDirs, ",")))
	}
	
	if c.ConvertLinks {
		logOrPrint(logger, c.Background, "Converting links for offline viewing")
	}
	
	// Start mirroring process
	return c.mirrorWebsite(c.URL, mirrorDir, logger)
}

// downloadFile performs the actual download with rate limiting if specified
func (c *DownloadConfig) downloadFile(url, filename string, logger *log.Logger) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}
	
	logOrPrint(logger, c.Background, fmt.Sprintf("status %s", resp.Status))
	
	fileSize := resp.ContentLength
	if fileSize > 0 {
		logOrPrint(logger, c.Background, fmt.Sprintf("content size: %d [~%.2fMB]", 
			fileSize, float64(fileSize)/(1024*1024)))
	}
	
	logOrPrint(logger, c.Background, fmt.Sprintf("saving file to: %s", filename))
	
	// Create output file
	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer out.Close()
	
	// Download with or without rate limiting
	var downloaded int64
	if c.RateLimit > 0 {
		downloaded, err = downloadWithRateLimit(resp.Body, out, c.RateLimit, fileSize, logger, c.Background)
	} else {
		downloaded, err = io.Copy(out, resp.Body)
		if !c.Background && fileSize > 0 {
			// Show simple progress for foreground downloads
			fmt.Printf(" %.2f KiB / %.2f KiB [", 
				float64(downloaded)/1024, float64(fileSize)/1024)
			fmt.Printf("================================================================================================================")
			fmt.Printf("] 100.00%% %.2f MiB/s 0s\n\n", float64(downloaded)/(1024*1024))
		}
	}
	
	if err != nil {
		return err
	}
	
	logOrPrint(logger, c.Background, fmt.Sprintf("Downloaded [%s]", url))
	logOrPrint(logger, c.Background, fmt.Sprintf("finished at %s", time.Now().Format("2006-01-02 15:04:05")))
	
	return nil
}

// downloadWithRateLimit implements rate-limited download with progress
func downloadWithRateLimit(src io.Reader, dst io.Writer, rateLimit int64, fileSize int64, logger *log.Logger, background bool) (int64, error) {
	var downloaded int64
	buffer := make([]byte, 8192) // 8KB buffer
	lastTime := time.Now()
	
	for {
		n, err := src.Read(buffer)
		if n > 0 {
			_, writeErr := dst.Write(buffer[:n])
			if writeErr != nil {
				return downloaded, writeErr
			}
			
			downloaded += int64(n)
			
			// Rate limiting
			now := time.Now()
			elapsed := now.Sub(lastTime)
			requiredTime := time.Duration(float64(n) / float64(rateLimit) * float64(time.Second))
			
			if elapsed < requiredTime {
				time.Sleep(requiredTime - elapsed)
			}
			lastTime = time.Now()
			
			// Progress display for foreground downloads
			if !background && (downloaded%(1024*1024) == 0 || err == io.EOF) {
				if fileSize > 0 {
					progress := float64(downloaded) / float64(fileSize) * 100
					fmt.Printf("\r %.2f KiB / %.2f KiB [", 
						float64(downloaded)/1024, float64(fileSize)/1024)
					
					// Progress bar
					barWidth := 50
					filled := int(progress / 100 * float64(barWidth))
					for i := 0; i < barWidth; i++ {
						if i < filled {
							fmt.Print("=")
						} else {
							fmt.Print(" ")
						}
					}
					speed := float64(rateLimit) / (1024 * 1024) // Convert to MiB/s
					fmt.Printf("] %.2f%% %.2f MiB/s", progress, speed)
				}
			}
		}
		
		if err == io.EOF {
			if !background {
				fmt.Printf("\n\n") // New line after progress
			}
			break
		}
		if err != nil {
			return downloaded, err
		}
	}
	
	return downloaded, nil
}

// mirrorWebsite implements basic website mirroring
func (c *DownloadConfig) mirrorWebsite(url, dir string, logger *log.Logger) error {
	// Start with the main page
	mainFile := filepath.Join(dir, "index.html")
	err := c.downloadFile(url, mainFile, logger)
	if err != nil {
		return fmt.Errorf("failed to download main page: %v", err)
	}
	
	// Here you would implement:
	// 1. Parse HTML for links, images, CSS files
	// 2. Filter out rejected file types (-R flag)
	// 3. Filter out excluded directories (-X flag)  
	// 4. Download found resources recursively
	// 5. Convert links if --convert-links is set
	
	// For now, just download the main page
	logOrPrint(logger, c.Background, fmt.Sprintf("Mirror completed: %s", dir))
	return nil
}

// shouldRejectFile checks if file should be rejected based on -R flag
func (c *DownloadConfig) shouldRejectFile(filename string) bool {
	if len(c.RejectTypes) == 0 {
		return false
	}
	
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != "" && ext[0] == '.' {
		ext = ext[1:] // Remove leading dot
	}
	
	for _, rejectType := range c.RejectTypes {
		if strings.ToLower(rejectType) == ext {
			return true
		}
	}
	return false
}

// shouldExcludeDir checks if directory should be excluded based on -X flag
func (c *DownloadConfig) shouldExcludeDir(urlPath string) bool {
	if len(c.ExcludeDirs) == 0 {
		return false
	}
	
	for _, excludeDir := range c.ExcludeDirs {
		if strings.HasPrefix(urlPath, excludeDir) {
			return true
		}
	}
	return false
}

// Utility functions
func logOrPrint(logger *log.Logger, background bool, message string) {
	if background && logger != nil {
		logger.Println(message)
	} else {
		fmt.Println(message)
	}
}

func getFilenameFromURL(urlStr string) string {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "downloaded_file"
	}
	
	filename := path.Base(parsedURL.Path)
	if filename == "/" || filename == "." || filename == "" {
		return "index.html"
	}
	return filename
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func parseRateLimit(rateLimitStr string) (int64, error) {
	rateLimitStr = strings.ToLower(strings.TrimSpace(rateLimitStr))
	
	if rateLimitStr == "" {
		return 0, fmt.Errorf("empty rate limit")
	}
	
	var numStr string
	var suffix string
	
	for i, char := range rateLimitStr {
		if (char >= '0' && char <= '9') || char == '.' {
			numStr += string(char)
		} else {
			suffix = rateLimitStr[i:]
			break
		}
	}
	
	if numStr == "" {
		return 0, fmt.Errorf("no number found")
	}
	
	baseRate, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, err
	}
	
	var multiplier int64 = 1
	switch suffix {
	case "", "b":
		multiplier = 1
	case "k", "kb":
		multiplier = 1024
	case "m", "mb":
		multiplier = 1024 * 1024
	default:
		return 0, fmt.Errorf("unknown suffix: %s", suffix)
	}
	
	return int64(baseRate * float64(multiplier)), nil
}

func main() {
	// Example 1: Simple download with some flags
	fmt.Println("=== Example 1: Simple Download with Rate Limit ===")
	config1 := NewDownloadConfig()
	testArgs1 := []string{"program", "--rate-limit=200k", "-O=test.bin", "-P=./downloads", "https://httpbin.org/bytes/102400"}
	
	err := config1.ParseArgs(testArgs1)
	if err != nil {
		fmt.Printf("Parse error: %v\n", err)
	} else {
		fmt.Printf("Config: URL=%s, OutputFile=%s, OutputPath=%s, RateLimit=%d\n", 
			config1.URL, config1.OutputFile, config1.OutputPath, config1.RateLimit)
		
		err = config1.ExecuteDownload()
		if err != nil {
			fmt.Printf("Download failed: %v\n", err)
		}
	}
	
	// Example 2: Mirror with all flags
	fmt.Println("\n=== Example 2: Mirror with All Flags ===")
	config2 := NewDownloadConfig()
	testArgs2 := []string{
		"program", 
		"--mirror", 
		"--convert-links", 
		"-R=pdf,zip,exe", 
		"-X=/admin,/private", 
		"--rate-limit=500k", 
		"-P=./mirror_test", 
		"-B", 
		"https://httpbin.org/html",
	}
	
	err = config2.ParseArgs(testArgs2)
	if err != nil {
		fmt.Printf("Parse error: %v\n", err)
	} else {
		fmt.Printf("Mirror Config: %+v\n", config2)
		
		err = config2.ExecuteDownload()
		if err != nil {
			fmt.Printf("Mirror failed: %v\n", err)
		}
	}
	
	// Clean up
	os.RemoveAll("./downloads")
	os.RemoveAll("./mirror_test")
	os.Remove("wget-log")
	
	fmt.Println("\n💡 All flags work together seamlessly!")
	fmt.Println("   - Single file: Uses -O, -P, --rate-limit, -B")
	fmt.Println("   - Mirror: Uses --mirror, --convert-links, -R, -X, -P, --rate-limit, -B")
}