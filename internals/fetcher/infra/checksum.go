package infra

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/eichiarakaki/aegis/internals/logger"
)

// SHA256Verifier implements domain.ChecksumVerifier using SHA-256.
type SHA256Verifier struct{}

// NewSHA256Verifier constructs a SHA256Verifier.
func NewSHA256Verifier() *SHA256Verifier {
	return &SHA256Verifier{}
}

// VerifyAllChecksums walks dataPath and validates every .CHECKSUM sidecar file.
// Returns the number of failures encountered.
func (v *SHA256Verifier) VerifyAllChecksums(dataPath string) int {
	failures := 0

	err := filepath.Walk(dataPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToUpper(path), ".CHECKSUM") {
			if verifyErr := v.verifyChecksum(path); verifyErr != nil {
				fmt.Fprintf(os.Stderr, "[ERR] %v\n", verifyErr)
				failures++
			}
		}
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERR] walking for checksums: %v\n", err)
	}

	return failures
}

// verifyChecksum reads the .CHECKSUM sidecar and validates the SHA-256 digest
// of the corresponding data file that lives in the same directory.
func (v *SHA256Verifier) verifyChecksum(checksumPath string) error {
	f, err := os.Open(checksumPath)
	if err != nil {
		return fmt.Errorf("open checksum file: %w", err)
	}
	defer f.Close()

	// Binance checksum files contain a single line: "<hex-digest>  <filename>"
	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return fmt.Errorf("checksum file is empty: %s", checksumPath)
	}
	line := strings.TrimSpace(scanner.Text())

	parts := strings.Fields(line)
	if len(parts) < 2 {
		return fmt.Errorf("unexpected checksum format in %s: %q", checksumPath, line)
	}
	expectedHash := strings.ToLower(parts[0])
	dataFilename := parts[1]

	dataPath := filepath.Join(filepath.Dir(checksumPath), filepath.Base(dataFilename))

	df, err := os.Open(dataPath)
	if err != nil {
		return fmt.Errorf("open data file %s: %w", dataPath, err)
	}
	defer df.Close()

	h := sha256.New()
	if _, err := io.Copy(h, df); err != nil {
		return fmt.Errorf("hashing %s: %w", dataPath, err)
	}
	actualHash := hex.EncodeToString(h.Sum(nil))

	if actualHash != expectedHash {
		return fmt.Errorf("checksum MISMATCH for %s: expected %s, got %s",
			dataFilename, expectedHash, actualHash)
	}

	logger.Infof("CHECKSUM OK %s", dataFilename)
	return nil
}
