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

// UnzipAll walks dataPath and extracts every .zip archive found.
// Returns the number of failures encountered.
func (e *ZipExtractor) UnzipAll(dataPath string, removeAfterExtraction bool, overrideExtractedFiles bool, extractFiles bool) int {
	failures := 0

	err := filepath.Walk(dataPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".zip") {
			if unzipErr := e.unzipFile(path, removeAfterExtraction, overrideExtractedFiles, extractFiles); unzipErr != nil {
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
// command, then removes the archive if removeAfterExtraction is true.
//
// Removal is always attempted after a successful extraction - or when the
// archive was skipped because the .csv already exists - so that stale .zip
// files are cleaned up regardless of whether extraction was needed.
func (e *ZipExtractor) unzipFile(zipPath string, removeAfterExtraction bool, overrideExtractedFiles bool, extractFiles bool) error {
	// removeIfRequested is a helper that deletes the archive when the config asks
	// for it. It is called on every non-error return path so that .zip files are
	// cleaned up even when extraction is skipped.
	removeIfRequested := func() {
		if !removeAfterExtraction {
			return
		}
		if err := os.Remove(zipPath); err != nil {
			logger.Warnf("could not remove archive after extraction: %s", zipPath)
		}
	}

	// If extraction is disabled globally, skip but still honour removal.
	if !extractFiles {
		logger.Infof("SKIP extraction of %s (disabled in config)", filepath.Base(zipPath))
		removeIfRequested()
		return nil
	}

	// If overwrite is off and the expected .csv already exists, skip extraction
	// but still honour removal so stale .zip files are cleaned up.
	if !overrideExtractedFiles {
		expectedCSV := strings.TrimSuffix(zipPath, ".zip") + ".csv"
		if _, err := os.Stat(expectedCSV); err == nil {
			logger.Infof("SKIP %s (already extracted)", filepath.Base(zipPath))
			removeIfRequested()
			return nil
		}
	}

	destDir := filepath.Dir(zipPath)

	cmd := exec.Command("unzip", "-o", zipPath, "-d", destDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// Do not remove the archive if extraction failed - keep it for diagnosis.
		return fmt.Errorf("unzip %s: %w", zipPath, err)
	}

	logger.Infof("UNZIP OK %s", filepath.Base(zipPath))
	removeIfRequested()
	return nil
}
