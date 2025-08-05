// mirror.go
package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

type FlagsComponents struct {
	Links        []string
	InputFile    string
	OutputFile   string
	PathFile     string
	RateLimite   string
	Exclude      []string
	Reject       []string
	isMirror     bool
	Background   bool
	OnlySameHost bool
	RootHost     string
	Convert      bool
	BaseDir      string
	Client       *http.Client
	MaxDepth     int
	visited      map[string]struct{}
	visitedMu    sync.RWMutex
	// wg         sync.WaitGroup
}

var cssURLRegex = regexp.MustCompile(`url\(['"]?([^'")]+)['"]?\)`)

// NewMirrorConfig initializes config with root domain
func (m *FlagsComponents) NewMirrorConfig(rootURL string) error {
	u, err := url.Parse(rootURL)
	if err != nil {
		return err
	}
	host := u.Host

	// Create transport with idle connections
	transport := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: false,
	}

	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}

	m.BaseDir = "."
	m.MaxDepth = 3
	m.Background = false
	m.OnlySameHost = true
	m.RootHost = host
	m.Client = client
	m.visited = make(map[string]struct{})
	m.visitedMu = sync.RWMutex{}

	return nil
}

// ParseAndDownload downloads a page and its assets
func (m *FlagsComponents) ParseAndDownload(pageURL string) error {
	// Log background mode if enabled
	if m.Background {
		logBackground()
		// Optionally redirect output to log file
		if file, err := openLogFile(); err == nil {
			Stdout = file
			Stderr = file
		}
	}

	logStart(pageURL)

	u, err := url.Parse(pageURL)
	if err != nil {
		logError(fmt.Sprintf("Invalid URL: %v", err))
		return err
	}

	if err := m.crawl(u, 0); err != nil {
		return err
	}

	logFinish(pageURL)
	return nil
}

func (m *FlagsComponents) crawl(u *url.URL, depth int) error {
	absURL := u.String()
	// Skip if already visited
	m.visitedMu.Lock()
	if _, seen := m.visited[absURL]; seen {
		m.visitedMu.Unlock()
		return nil
	}
	m.visited[absURL] = struct{}{}
	m.visitedMu.Unlock()

	// Check reject list before downloading
	for _, ext := range m.Reject {
		if strings.HasSuffix(strings.ToLower(u.Path), strings.ToLower(ext)) {
			fmt.Printf("[INFO] Skipping %s due to reject rule (%s)\n", u.String(), ext)
			return nil
		}
	}

	// Skip URLs with excluded path prefixes
	for _, prefix := range m.Exclude {
		if strings.HasPrefix(u.Path, prefix) {
			fmt.Printf("[INFO] Skipping %s due to exclude path prefix (%s)\n", u.String(), prefix)
			return nil
		}
	}

	// Respect max depth
	if depth >= m.MaxDepth {
		return nil
	}

	// Skip external domains
	if m.OnlySameHost && u.Host != m.RootHost {
		return nil
	}

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return err
	}

	// Set a real User-Agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Wget/1.21)")
	resp, err := m.Client.Do(req)
	if err != nil {
		logError(fmt.Sprintf("Failed to fetch %s: %v", u.String(), err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logError(fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status))
		return fmt.Errorf("failed: %s", resp.Status)
	}

	// Log the request status
	logRequest(resp.Status)

	contentType := resp.Header.Get("Content-Type")
	localPath, err := m.GetLocalPath(u, contentType)
	if err != nil {
		logError(fmt.Sprintf("Failed to determine path for %s: %v", u.String(), err))
		return err
	}

	// Create dir and save file
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		logError(fmt.Sprintf("Failed to create directory for %s: %v", localPath, err))
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logError(fmt.Sprintf("Failed to read body from %s: %v", u.String(), err))
		return err
	}

	// Log file size and saving path
	size := int64(len(body))
	logSize(size)
	logSaving(localPath)

	if err := ioutil.WriteFile(localPath, body, 0o644); err != nil {
		logError(fmt.Sprintf("Failed to write file %s: %v", localPath, err))
		return err
	}

	// === NEW: convert links inside saved HTML if --convert-links is enabled ===
	if m.Convert && strings.Contains(contentType, "text/html") {
		convertedBody, err := m.convertLinks(body, u, localPath)
		if err != nil {
			logError(fmt.Sprintf("Failed to convert links in %s: %v", localPath, err))
		} else {
			err = ioutil.WriteFile(localPath, convertedBody, 0o644)
			if err != nil {
				logError(fmt.Sprintf("Failed to write converted file %s: %v", localPath, err))
			}
		}
	}

	// Extract links if HTML
	if strings.Contains(contentType, "text/html") {
		doc, err := html.Parse(strings.NewReader(string(body)))
		if err != nil {
			logError(fmt.Sprintf("Failed to parse HTML from %s: %v", u.String(), err))
			return err
		}

		var base *url.URL
		var walkBase func(*html.Node)
		walkBase = func(n *html.Node) {
			if n.Type == html.ElementNode && n.Data == "base" {
				for _, attr := range n.Attr {
					if attr.Key == "href" {
						if b, _ := u.Parse(attr.Val); b != nil {
							base = b
						}
					}
				}
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walkBase(c)
			}
		}
		walkBase(doc)

		baseURL := u
		if base != nil {
			baseURL = base
		}

		seen := make(map[string]struct{})
		var extract func(*html.Node)
		extract = func(n *html.Node) {
			if n.Type == html.ElementNode {
				attrs := []string{"href", "src", "srcset", "poster", "data-src", "data-srcset", "data-original", "action"}
				for _, key := range attrs {
					for _, attr := range n.Attr {
						if attr.Key == key {
							m.extractURLs(attr.Val, baseURL, seen, key == "srcset" || key == "data-srcset")
						}
					}
				}

				// Extract from style="background-image: url(...)"
				for _, attr := range n.Attr {
					if attr.Key == "style" {
						matches := cssURLRegex.FindAllStringSubmatch(attr.Val, -1)
						for _, match := range matches {
							m.addLink(match[1], baseURL, seen)
						}
					}
				}
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				extract(c)
			}
		}
		extract(doc)

		// Download found links
		var wg sync.WaitGroup
		for link := range seen {
			linkURL, err := u.Parse(link)
			if err != nil {
				continue
			}
			wg.Add(1)
			go func(url *url.URL) {
				defer wg.Done()
				_ = m.crawl(url, depth+1)
			}(linkURL)
		}
		wg.Wait()
	}

	// Parse CSS files
	if strings.Contains(contentType, "css") {
		links := m.extractCSSLinks(body, u)
		var wg sync.WaitGroup
		for _, link := range links {
			linkURL, err := u.Parse(link)
			if err != nil {
				continue
			}
			wg.Add(1)
			go func(url *url.URL) {
				defer wg.Done()
				_ = m.crawl(url, depth+1)
			}(linkURL)
		}
		wg.Wait()
	}
	return nil
}

// extractURLs handles normal or srcset URLs
func (m *FlagsComponents) extractURLs(val string, base *url.URL, seen map[string]struct{}, isSrcSet bool) {
	if isSrcSet {
		parts := strings.Split(val, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			urlPart := strings.Fields(part)[0]
			m.addLink(urlPart, base, seen)
		}
	} else {
		m.addLink(val, base, seen)
	}
}

// addLink resolves and deduplicates
func (m *FlagsComponents) addLink(rawURL string, base *url.URL, seen map[string]struct{}) {
	u, err := base.Parse(rawURL)
	if err != nil {
		return
	}
	abs := u.String()
	seen[abs] = struct{}{}
}

// extractCSSLinks finds url(...) in CSS
func (m *FlagsComponents) extractCSSLinks(css []byte, baseURL *url.URL) []string {
	matches := cssURLRegex.FindAllStringSubmatch(string(css), -1)
	var urls []string
	for _, m := range matches {
		u, err := baseURL.Parse(m[1])
		if err == nil {
			urls = append(urls, u.String())
		}
	}
	return urls
}

// GetLocalPath: save all domains under BaseDir as subfolders
func (m *FlagsComponents) GetLocalPath(u *url.URL, contentType string) (string, error) {
	cleanURL := *u
	cleanURL.Fragment = ""
	cleanURL.RawQuery = ""

	path := cleanURL.Path
	if path == "" || strings.HasSuffix(path, "/") {
		path += "index.html"
	}

	// Guess extension from Content-Type
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		switch {
		case strings.Contains(contentType, "javascript"):
			path += ".js"
		case strings.Contains(contentType, "css"):
			path += ".css"
		case strings.Contains(contentType, "image/png"):
			path += ".png"
		case strings.Contains(contentType, "image/jpeg"), strings.Contains(contentType, "image/jpg"):
			path += ".jpg"
		case strings.Contains(contentType, "image/gif"):
			path += ".gif"
		default:
			if !strings.HasSuffix(path, ".html") {
				path += ".html"
			}
		}
	}

	path = filepath.Clean("/" + path)
	return filepath.Join(m.BaseDir, u.Host, path), nil
}

// convertLinks rewrites all URLs in the HTML content to point to local relative paths
func (m *FlagsComponents) convertLinks(htmlContent []byte, pageURL *url.URL, localPath string) ([]byte, error) {
	doc, err := html.Parse(bytes.NewReader(htmlContent))
	if err != nil {
		return nil, err
	}

	// Helper to recursively walk nodes and rewrite URLs
	var rewrite func(*html.Node)
	rewrite = func(n *html.Node) {
		if n.Type == html.ElementNode {
			// Attributes that can contain URLs
			attrsToCheck := []string{"href", "src", "srcset", "poster", "data-src", "data-srcset", "data-original", "action", "style"}

			for i, attr := range n.Attr {
				for _, key := range attrsToCheck {
					if attr.Key == key {
						if key == "srcset" || key == "data-srcset" {
							n.Attr[i].Val = m.convertSrcset(attr.Val, pageURL, localPath)
						} else if key == "style" {
							n.Attr[i].Val = m.convertCSSURLs(attr.Val, pageURL, localPath)
						} else {
							n.Attr[i].Val = m.convertSingleURL(attr.Val, pageURL, localPath)
						}
					}
				}
			}
		}

		// Recursively process children
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			rewrite(c)
		}
	}

	rewrite(doc)

	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// convertSingleURL converts a single URL to relative local path if possible
func (m *FlagsComponents) convertSingleURL(rawurl string, pageURL *url.URL, localPath string) string {
	parsedURL, err := pageURL.Parse(rawurl)
	if err != nil {
		// If invalid URL, return original
		return rawurl
	}

	// Skip if URL is a fragment or empty
	if parsedURL.Scheme == "" && (parsedURL.Host == "" || parsedURL.Host == pageURL.Host) {
		// It's relative or same host URL

		localFilePath, err := m.GetLocalPath(parsedURL, "") // contentType not needed here
		if err != nil {
			return rawurl
		}

		rel, err := filepath.Rel(filepath.Dir(localPath), localFilePath)
		if err != nil {
			return rawurl
		}

		// Convert Windows paths to slashes for URLs
		rel = filepath.ToSlash(rel)
		return rel
	}

	// External URLs or others left unchanged
	return rawurl
}

// convertSrcset handles rewriting of multiple URLs in srcset attributes
func (m *FlagsComponents) convertSrcset(val string, pageURL *url.URL, localPath string) string {
	parts := strings.Split(val, ",")
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		fields := strings.Fields(part)
		if len(fields) == 0 {
			continue
		}
		urlPart := fields[0]
		converted := m.convertSingleURL(urlPart, pageURL, localPath)
		fields[0] = converted
		parts[i] = strings.Join(fields, " ")
	}
	return strings.Join(parts, ", ")
}

// convertCSSURLs rewrites url(...) inside CSS style attribute strings
func (m *FlagsComponents) convertCSSURLs(styleVal string, pageURL *url.URL, localPath string) string {
	const prefix = "url("
	const suffix = ")"

	// Replace all url(...) with converted paths
	replacer := func(match string) string {
		trimmed := strings.TrimPrefix(match, prefix)
		trimmed = strings.TrimSuffix(trimmed, suffix)
		trimmed = strings.Trim(trimmed, `"'`)

		converted := m.convertSingleURL(trimmed, pageURL, localPath)
		return prefix + `"` + converted + `"` + suffix
	}

	// This is a simple way, you could do regex or manual search
	// Here we replace all occurrences of url(...) in the style string
	var result strings.Builder
	remaining := styleVal
	for {
		idx := strings.Index(remaining, prefix)
		if idx == -1 {
			result.WriteString(remaining)
			break
		}
		result.WriteString(remaining[:idx])
		remaining = remaining[idx:]
		endIdx := strings.Index(remaining, suffix)
		if endIdx == -1 {
			// malformed, just append the rest
			result.WriteString(remaining)
			break
		}

		urlFunc := remaining[:endIdx+len(suffix)]
		converted := replacer(urlFunc)
		result.WriteString(converted)
		remaining = remaining[endIdx+len(suffix):]
	}

	return result.String()
}
