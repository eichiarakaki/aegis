package infra

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/eichiarakaki/aegis/internals/logger"
)

// ZipExtractor implements domain.Extractor using the system unzip binary.
type ZipExtractor struct{}

// NewZipExtractor constructs a ZipExtractor.
func NewZipExtractor() *ZipExtractor {
	return &ZipExtractor{}
}

// UnzipAll walks dataPath and extracts every .zip archive found,
// removing the archive after a successful extraction.
// Returns the number of failures encountered.
func (e *ZipExtractor) UnzipAll(dataPath string) int {
	failures := 0

	err := filepath.Walk(dataPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".zip") {
			if unzipErr := e.unzipFile(path); unzipErr != nil {
				fmt.Fprintf(os.Stderr, "[ERR] %v\n", unzipErr)
				failures++
			}
		}
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERR] walking for zips: %v\n", err)
	}

	return failures
}

// unzipFile extracts a .zip archive to its own directory via the system unzip
// command, then removes the archive on success.
func (e *ZipExtractor) unzipFile(zipPath string) error {
	destDir := filepath.Dir(zipPath)

	cmd := exec.Command("unzip", "-o", zipPath, "-d", destDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("unzip %s: %w", zipPath, err)
	}

	if err := os.Remove(zipPath); err != nil {
		logger.Infof("WARN could not remove archive after extraction: %s", zipPath)
	}

	logger.Infof("UNZIP OK %s", filepath.Base(zipPath))
	return nil
}
