package main

import (
	"fmt"
	"io"
	"time"
)

func copyWithRateLimit(src io.Reader, out io.Writer, rateLimit int64, total int64, filename string, Link string) (int64, error) {
	var written int64

	bufferSize := 32 * 1024 // Default 32KB
	if rateLimit > 0 {
		if rateLimit < int64(bufferSize) {
			bufferSize = int(rateLimit / 10)
			if bufferSize < 1024 {
				bufferSize = 1024
			}
		}
	}

	buf := make([]byte, bufferSize)
	startTime := time.Now()
	lastUpdate := time.Now()

	for {
		readStart := time.Now()

		number_of_bytes_readed, err := src.Read(buf)
		if number_of_bytes_readed > 0 {
			written_bytes, writeErr := out.Write(buf[:number_of_bytes_readed])
			if writeErr != nil {
				return written, writeErr
			}
			if written_bytes != number_of_bytes_readed {
				return written, io.ErrShortWrite
			}
			written += int64(number_of_bytes_readed)

			// if rateLimit > 0 {
				expectedDuration := time.Duration(float64(number_of_bytes_readed) / float64(rateLimit) * float64(time.Second))

				// How long did it actually take?
				actualDuration := time.Since(readStart)

				// If we read too fast, sleep for the difference
				if actualDuration < expectedDuration {
					sleepTime := expectedDuration - actualDuration
					time.Sleep(sleepTime)
				}
			// }

			// Update progress
			now := time.Now()
			if now.Sub(lastUpdate) > 500*time.Millisecond || err == io.EOF {
				showProgress(written, total, filename, time.Since(startTime))
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
	showProgress(written, total, filename, time.Since(startTime))
	fmt.Println()
	return written, nil
}
