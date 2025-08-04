package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

type Link struct {
	Href string
	Src  string
	Tag  string
}

type MirrorConfig struct {
	Reject     []string // file extensions to reject
	Exclude    []string // path prefixes to exclude
	Convert    bool
	BaseURL    *url.URL
	BaseDir    string
	Client     *http.Client
	Downloader *Downloader
}

func NewMirrorConfig(urlStr string) (*MirrorConfig, error) {
    u, err := url.Parse(urlStr)
    if err != nil {
        return nil, err
    }
    return &MirrorConfig{
        BaseURL: u,
        BaseDir: ".", // Mirror root is current directory
        Client:  &http.Client{Timeout: 10 * time.Second},
    }, nil
}

func (m *MirrorConfig) ShouldDownload(link string) bool {
	u, err := url.Parse(link)
	if err != nil {
		return false
	}

	// Only same host
	if u.Host != m.BaseURL.Host {
		return false
	}

	// Exclude paths
	for _, prefix := range m.Exclude {
		if strings.HasPrefix(u.Path, prefix) {
			return false
		}
	}

	// Reject extensions
	ext := strings.ToLower(filepath.Ext(u.Path))
	for _, rej := range m.Reject {
		if ext == "."+strings.ToLower(rej) {
			return false
		}
	}

	return true
}

// GetLocalPath converts a URL to a local file path, preserving full directory structure
func (m *MirrorConfig) GetLocalPath(link string) (string, error) {
    u, err := url.Parse(link)
    if err != nil {
        return "", err
    }

    path := u.Path
    if path == "" || path[len(path)-1] == '/' {
        path += "index.html"
    }

    path = filepath.Clean("/" + path)
    return filepath.Join(m.BaseDir, u.Host, path), nil
}

func (m *MirrorConfig) ConvertLink(link string) string {
    u, err := url.Parse(link)
    if err != nil || u.Host != m.BaseURL.Host {
        return link
    }
    path := u.Path
    if path == "" || path[len(path)-1] == '/' {
        path += "index.html"
    }
    fullPath := filepath.Join(m.BaseURL.Host, path)
    return "./" + filepath.ToSlash(fullPath)
}

func (m *MirrorConfig) ParseAndDownload(u string) error {
    resp, err := m.Client.Get(u)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("failed to fetch %s: %s", u, resp.Status)
    }

    // Use full path from URL to determine where to save
    localPath, err := m.GetLocalPath(u)
    if err != nil {
        return err
    }

    // Create full directory structure
    if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
        return err
    }

    outFile, err := os.Create(localPath)
    if err != nil {
        return err
    }
    defer outFile.Close()

    contentType := resp.Header.Get("Content-Type")
    isHTML := strings.Contains(contentType, "text/html")

    var links []string

    if isHTML {
        var buf strings.Builder
        tokenizer := html.NewTokenizer(resp.Body)

        for {
            tt := tokenizer.Next()
            if tt == html.ErrorToken {
                break
            }

            token := tokenizer.Token()
            tag := token.Data

            // Extract href and src
            for _, attr := range token.Attr {
                var link string
                if (tag == "a" || tag == "link") && attr.Key == "href" {
                    link = attr.Val
                } else if (tag == "img" || tag == "script" || tag == "link") && attr.Key == "src" {
                    link = attr.Val
                }

                if link != "" {
                    absLink := m.resolveURL(u, link)
                    if m.ShouldDownload(absLink) {
                        links = append(links, absLink)
                    }
                }
            }

            buf.WriteString(token.String())
        }

        htmlContent := buf.String()

        // Convert links for offline viewing if enabled
        if m.Convert {
            for _, link := range links {
                localLink := "./" + strings.TrimPrefix(
                    strings.TrimPrefix(link, "https://"),
                    "http://",
                )
                localLink = strings.ReplaceAll(localLink, "/", string(filepath.Separator))
                localLink = ".\\" + localLink // for Windows? Better: use filepath.Join logic
                // Instead, use relative path from target
                rel, _ := filepath.Rel(filepath.Dir(localPath), m.BaseDir)
                _ = rel // TODO: implement real relative conversion
                // For now, use simple conversion
                converted := m.ConvertLink(link)
                htmlContent = strings.ReplaceAll(htmlContent, link, converted)
            }
        }

        outFile.WriteString(htmlContent)
    } else {
        // Non-HTML: just copy the bytes
        io.Copy(outFile, resp.Body)
    }

    // Download all discovered assets concurrently
    var wg sync.WaitGroup
    for _, link := range links {
        wg.Add(1)
        go func(url string) {
            defer wg.Done()
            dl := NewDownloader(url)
            dl.DestDir = m.BaseDir // let buildOutputPath handle full path
            if err := dl.Download(); err != nil {
                fmt.Fprintf(Stderr, "Failed to download %s: %v\n", url, err)
            }
        }(link)
    }
    wg.Wait()

    return nil
}

func (m *MirrorConfig) resolveURL(base, rel string) string {
	u, err := url.Parse(rel)
	if err != nil {
		return rel
	}
	baseURL, _ := url.Parse(base)
	return baseURL.ResolveReference(u).String()
}

func updateAttr(attrs []html.Attribute, key, newVal string) []html.Attribute {
	for i := range attrs {
		if attrs[i].Key == key {
			attrs[i].Val = newVal
		}
	}
	return attrs
}
