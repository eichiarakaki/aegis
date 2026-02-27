package infra

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/eichiarakaki/aegis/internals/fetcher/domain"
	"github.com/eichiarakaki/aegis/internals/logger"
)

const cdnBaseURL = "https://data.binance.vision"

// CDNDownloader implements domain.FileDownloader using the Binance CDN.
type CDNDownloader struct{}

// NewCDNDownloader constructs a CDNDownloader.
func NewCDNDownloader() *CDNDownloader {
	return &CDNDownloader{}
}

// datePattern matches the YYYY-MM-DD fragment embedded in Binance filenames.
// e.g. "BTCUSDT-1m-2024-01-21.zip" → "2024-01-21"
var datePattern = regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)

// extractDate parses the first YYYY-MM-DD occurrence found in filename.
func extractDate(filename string) (time.Time, error) {
	match := datePattern.FindString(filename)
	if match == "" {
		return time.Time{}, fmt.Errorf("no date in filename %q", filename)
	}
	return time.Parse("2006-01-02", match)
}

// withinRange reports whether t falls within the inclusive [start, end] range.
func withinRange(t time.Time, dr domain.DateRange) bool {
	return !t.Before(dr.Start) && !t.After(dr.End)
}

// DownloadFile fetches key from the CDN and writes it to destDir.
// It skips the file if its embedded date falls outside dateRange.
// It also skips already-existing files unless overwrite is true.
func (d *CDNDownloader) DownloadFile(key, destDir string, overwrite bool, dateRange domain.DateRange) error {
	filename := filepath.Base(key)

	// Extract and validate the file date before doing any I/O
	fileDate, err := extractDate(filename)
	if err != nil {
		// If we cannot parse a date (e.g. CHECKSUM files with no date), let it through
		logger.Infof("WARN no date found in %s — downloading anyway", filename)
	} else if !withinRange(fileDate, dateRange) {
		logger.Infof("SKIP (out of range) %s", filename)
		return nil
	}

	fileURL := cdnBaseURL + "/" + key
	destPath := filepath.Join(destDir, filename)

	if _, err := os.Stat(destPath); err == nil {
		if !overwrite {
			logger.Infof("SKIP (exists) %s", filename)
			return nil
		}
		logger.Infof("OVERWRITE %s", filename)
		if err := os.Remove(destPath); err != nil {
			return fmt.Errorf("remove existing file %s: %w", destPath, err)
		}
	}

	body, statusCode, err := doGetWithRetry(fileURL)
	if err != nil {
		return fmt.Errorf("download %s: %w", key, err)
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("download %s: HTTP %d", key, statusCode)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	n, err := f.Write(body)
	if err != nil {
		return err
	}

	logger.Infof("OK %s (%.1f KB)", filename, float64(n)/1024)
	return nil
}
