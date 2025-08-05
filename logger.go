package main

import (
    "fmt"
    "io"
    "os"
    "time"
)

var (
    Stdout io.Writer = os.Stdout
    Stderr io.Writer = os.Stderr
)

func logStart() {
    t := time.Now().Format("2006-01-02 15:04:05")
    fmt.Fprintf(Stdout, "start at %s\n", t)
}

func logRequest(status string) {
    fmt.Fprintf(Stdout, "sending request, awaiting response... status %s\n", status)
}

func logSize(size int64) {
    human := humanSize(float64(size))
    fmt.Fprintf(Stdout, "content size: %d [~%s]\n", size, human)
}

func logSaving(path string) {
    fmt.Fprintf(Stdout, "saving file to: %s\n", path)
}

func logProgress(downloaded, total int64, speed float64, elapsed time.Duration) {
    dl := humanBytes(float64(downloaded))
    tot := humanBytes(float64(total))
    percent := 0.0
    if total > 0 {
        percent = float64(downloaded) / float64(total) * 100
    }
    remaining := ""
    if speed > 0 && downloaded < total {
        secs := float64(total-downloaded) / speed
        remaining = fmt.Sprintf("%.0fs", secs)
    } else {
        remaining = "0s"
    }

    barLen := 60
    filled := int(float64(barLen) * percent / 100)
    bar := ""
    for i := 0; i < barLen; i++ {
        if i < filled {
            bar += "="
        } else {
            bar += " "
        }
    }

    speedStr := humanBytes(speed) + "/s"
    fmt.Fprintf(Stdout, "\r%s / %s [%s] %.2f%% %s %s",
        dl, tot, bar, percent, speedStr, remaining)
}

func logFinish(url string) {
    t := time.Now().Format("2006-01-02 15:04:05")
    fmt.Fprintf(Stdout, "\n\nDownloaded [%s]\n", url)
    fmt.Fprintf(Stdout, "finished at %s\n", t)
}

func logBackground() {
    fmt.Fprintf(Stdout, "Output will be written to \"wget-log\".\n")
}

func logError(msg string) {
    fmt.Fprintf(Stderr, "ERROR: %s\n", msg)
}

func openLogFile() (*os.File, error) {
    return os.OpenFile("wget-log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
}

// Utility functions
func humanSize(bytes float64) string {
    switch {
    case bytes >= 1<<30:
        return fmt.Sprintf("%.2fGB", bytes/(1<<30))
    case bytes >= 1<<20:
        return fmt.Sprintf("%.2fMB", bytes/(1<<20))
    case bytes >= 1<<10:
        return fmt.Sprintf("%.2fKB", bytes/(1<<10))
    default:
        return fmt.Sprintf("%.0fB", bytes)
    }
}

func humanBytes(bytes float64) string {
    switch {
    case bytes >= 1<<20:
        return fmt.Sprintf("%.2fMiB", bytes/(1<<20))
    case bytes >= 1<<10:
        return fmt.Sprintf("%.2fKiB", bytes/(1<<10))
    default:
        return fmt.Sprintf("%.0fB", bytes)
    }
}