// mirror.go
package main

import (
	"bytes"
	"fmt"
	"io"
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
	routHost     string
	Convert      bool
	BaseDir      string
	Client       *http.Client
	MaxDepth     int
	visited      map[string]struct{}
	visitedMu    sync.RWMutex
}

var cssURLRegex = regexp.MustCompile(`url\(['"]?([^'")]+)['"]?\)`)

var cssURLPatterns = []*regexp.Regexp{
	regexp.MustCompile(`url\(['"]?([^'"()]+)['"]?\)`),           // url('path') or url("path") or url(path)
	regexp.MustCompile(`@import\s+['"]([^'"]+)['"]`),            // @import 'style.css'
	regexp.MustCompile(`@import\s+url\(['"]?([^'"()]+)['"]?\)`), // @import url('style.css')
}

func (m *FlagsComponents) NewMirrorConfig(rootURL string) error {
	u, err := url.Parse(rootURL)
	if err != nil {
		return err
	}

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
	m.MaxDepth = 1000
	m.Background = false
	m.OnlySameHost = true
	m.routHost = u.Host
	m.Client = client
	m.visited = make(map[string]struct{})
	m.visitedMu = sync.RWMutex{}

	return nil
}

func (m *FlagsComponents) crawl(absURL string, depth int) error {
	if depth >= m.MaxDepth {
		return nil
	}
	url, err := url.Parse(absURL)
	if err != nil || url.Host != m.routHost {
		return err
	}

	m.visitedMu.Lock()
	if _, seen := m.visited[absURL]; seen {
		m.visitedMu.Unlock()
		return nil
	}
	m.visited[absURL] = struct{}{}
	m.visitedMu.Unlock()

	for _, ext := range m.Reject {
		if strings.HasSuffix(strings.ToLower(url.Path), strings.ToLower(ext)) {
			fmt.Printf("[INFO] Skipping %s due to reject rule (%s)\n", absURL, ext)
			return nil
		}
	}

	for _, prefix := range m.Exclude {
		if strings.HasPrefix(url.Path, prefix) {
			fmt.Printf("[INFO] Skipping %s due to exclude path prefix (%s)\n", absURL, prefix)
			return nil
		}
	}

	req, err := http.NewRequest("GET", absURL, nil)
	if err != nil {
		return err
	}

	// Set a real User-Agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Wget/1.21)")
	resp, err := m.Client.Do(req)
	if err != nil {
		logError(fmt.Sprintf("Failed to fetch %s: %v", absURL, err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logError(fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status))
		return fmt.Errorf("failed: %s", resp.Status)
	}

	logRequest(resp.Status)

	contentType := resp.Header.Get("Content-Type")
	localPath, err := m.GetLocalPath(url, contentType)
	if err != nil {
		logError(fmt.Sprintf("Failed to determine path for %s: %v", absURL, err))
		return err
	}

	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		logError(fmt.Sprintf("Failed to create directory for %s: %v", localPath, err))
		return err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logError(fmt.Sprintf("Failed to read body from %s: %v", absURL, err))
		return err
	}

	// Log file size and saving path
	size := int64(len(body))
	logSize(size)
	logSaving(localPath)

	if err := os.WriteFile(localPath, body, 0o644); err != nil {
		logError(fmt.Sprintf("Failed to write file %s: %v", localPath, err))
		return err
	}

	if m.Convert && strings.Contains(contentType, "text/html") {
		convertedBody, err := m.convertLinks(body, url, localPath)
		if err != nil {
			logError(fmt.Sprintf("Failed to convert links in %s: %v", localPath, err))
		} else {
			err = os.WriteFile(localPath, convertedBody, 0o644)
			if err != nil {
				logError(fmt.Sprintf("Failed to write converted file %s: %v", localPath, err))
			}
		}
	}

	// Extract links if HTML
	if strings.Contains(contentType, "text/html") {
		doc, err := html.Parse(strings.NewReader(string(body)))
		if err != nil {
			logError(fmt.Sprintf("Failed to parse HTML from %s: %v", absURL, err))
			return err
		}
		seen := []string{}
		var extract func(*html.Node)
		extract = func(n *html.Node) {
			switch n.Type {
			case html.ElementNode:
				attrsToCheck := []string{"href", "src", "poster", "data-src", "action", "style"}
				for _, attr := range n.Attr {
					for _, key := range attrsToCheck {
						if attr.Key == key {
							switch key {
							case "style":
								seen = append(seen, m.extractCSSLinks([]byte(attr.Val), url)...)
							default:
								seen = append(seen, makeAbsoluteURL(attr.Val, url))
							}
						}
					}
				}

			case html.TextNode:
				if n.Data != "" {
					urls, _ := m.extractURLsFromCSS(n.Data, url)
					seen = append(seen, urls...)
				}
			}

			for c := n.FirstChild; c != nil; c = c.NextSibling {
				extract(c)
			}
		}
		extract(doc)

		// Download found links
		var wg sync.WaitGroup
		for _, link := range seen {
			wg.Add(1)
			go func(url string) {
				defer wg.Done()
				_ = m.crawl(url, depth+1)
			}(link)
		}
		wg.Wait()
	}

	// Parse CSS files
	if strings.Contains(contentType, "css") {
		links := m.extractCSSLinks(body, url)
		var wg sync.WaitGroup
		for _, link := range links {
			wg.Add(1)
			go func(url string) {
				defer wg.Done()
				_ = m.crawl(url, depth+1)
			}(link)
		}
		wg.Wait()
	}
	return nil
}

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
	path := u.Path
	if path == "" || strings.HasSuffix(path, "/") {
		path += "index.html"
	}

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
			path += ".txt"
		}
	}

	path = filepath.Clean("/" + path)
	return filepath.Join(m.BaseDir, u.Host, path), nil
}

func (m *FlagsComponents) convertLinks(htmlContent []byte, pageURL *url.URL, localPath string) ([]byte, error) {
	doc, err := html.Parse(bytes.NewReader(htmlContent))
	if err != nil {
		return nil, err
	}

	// Helper to recursively walk nodes and rewrite URLs
	var rewrite func(*html.Node)

	rewrite = func(n *html.Node) {
		switch n.Type {

		case html.ElementNode:
			// Attributes that can contain URLs
			attrsToCheck := []string{"href", "src", "poster", "data-src", "action", "style"}

			for i, attr := range n.Attr {
				for _, key := range attrsToCheck {
					if attr.Key == key {
						switch key {
						case "style":
							_, n.Attr[i].Val = m.convertURLsFromCSS(n.Attr[i].Val, pageURL)
						default:
							n.Attr[i].Val = makeAbsoluteURL(attr.Val, pageURL)
						}
					}
				}
			}

		case html.TextNode:
			if n.Data != "" {
				_, n.Data = m.convertURLsFromCSS(n.Data, pageURL)
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

func (m *FlagsComponents) extractURLsFromCSS(cssContent string, pageURL *url.URL) ([]string, string) {
	var urls []string
	for _, pattern := range cssURLPatterns {
		matches := pattern.FindAllStringSubmatch(cssContent, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				old := strings.TrimSpace(match[1])
				url := makeAbsoluteURL(old, pageURL)
				// url := old
				cssContent = strings.Replace(cssContent, old, url, 1)
				if url != "" {
					urls = append(urls, url)
				}
			}
		}
	}

	return urls, cssContent
}

func (m *FlagsComponents) convertURLsFromCSS(cssContent string, pageURL *url.URL) ([]string, string) {
	var urls []string
	for _, pattern := range cssURLPatterns {
		matches := pattern.FindAllStringSubmatch(cssContent, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				url := strings.TrimPrefix(strings.TrimSpace(match[1]), "/")
				// url := makeAbsoluteURL(old, pageURL)
				// url := old
				cssContent = strings.Replace(cssContent, strings.TrimSpace(match[1]), url, 1)
				if url != "" {
					urls = append(urls, url)
				}
			}
		}
	}

	return urls, cssContent
}

func makeAbsoluteURL(linkURL string, base_url *url.URL) string {
	link, err := url.Parse(linkURL)
	if err != nil {
		return ""
	}

	absolute := base_url.ResolveReference(link)
	return absolute.String()
}