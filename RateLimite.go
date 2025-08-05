package main

import (
	"io"
	"log"
	"time"
)

func copyWithRateLimit(src io.Reader, dst io.Writer, rateLimit int64, total int64, filename string, logger *log.Logger, background bool) (int64, error) {
	var written int64

	// Choose buffer size
	bufferSize := 32 * 1024 // Default 32KB
	if rateLimit > 0 {
		// Adjust buffer size based on rate limit
		if rateLimit < int64(bufferSize) {
			bufferSize = int(rateLimit / 10) // Read 1/10th of rate limit per chunk
			if bufferSize < 1024 {
				bufferSize = 1024 // Minimum 1KB
			}
		}
	}

	buf := make([]byte, bufferSize)
	startTime := time.Now()
	lastUpdate := time.Now()

	for {
		// Record time before read
		readStart := time.Now()

		// Read data
		number_of_bytes_readed, err := src.Read(buf)
		if number_of_bytes_readed > 0 {
			// Write data
			written_bytes, writeErr := dst.Write(buf[:number_of_bytes_readed])
			if writeErr != nil {
				return written, writeErr
			}
			if written_bytes != number_of_bytes_readed {
				return written, io.ErrShortWrite
			}
			written += int64(number_of_bytes_readed)

			// Rate limiting: calculate how long to sleep
			if rateLimit > 0 {
				// How long should this read have taken at the target rate?
				expectedDuration := time.Duration(float64(number_of_bytes_readed) / float64(rateLimit) * float64(time.Second))

				// How long did it actually take?
				actualDuration := time.Since(readStart)

				// If we read too fast, sleep for the difference
				if actualDuration < expectedDuration {
					sleepTime := expectedDuration - actualDuration
					time.Sleep(sleepTime)
				}
			}

			// Update progress
			now := time.Now()
			if now.Sub(lastUpdate) > 500*time.Millisecond || err == io.EOF {
				showProgress(written, total, filename, time.Since(startTime), logger, background)
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
	showProgress(written, total, filename, time.Since(startTime), logger, background)
	return written, nil
}
