package infra

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/eichiarakaki/aegis/internals/logger"
)

const cdnBaseURL = "https://data.binance.vision"

// CDNDownloader implements domain.FileDownloader using the Binance CDN.
type CDNDownloader struct{}

// NewCDNDownloader constructs a CDNDownloader.
func NewCDNDownloader() *CDNDownloader {
	return &CDNDownloader{}
}

// DownloadFile fetches key from the CDN and writes it to destDir.
// Skips the download if the file already exists on disk.
func (d *CDNDownloader) DownloadFile(key, destDir string, overwriteDownloadedFiles bool) error {
	fileURL := cdnBaseURL + "/" + key
	filename := filepath.Base(key)
	destPath := filepath.Join(destDir, filename)

	if _, err := os.Stat(destPath); err == nil {
		if !overwriteDownloadedFiles {
			logger.Infof("SKIP %s", filename)
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
