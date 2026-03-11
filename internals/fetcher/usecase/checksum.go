package usecase

import (
	"github.com/eichiarakaki/aegis/internals/config"
	"github.com/eichiarakaki/aegis/internals/fetcher/domain"
	"github.com/eichiarakaki/aegis/internals/logger"
)

// ChecksumUseCase orchestrates integrity verification across all downloaded files.
type ChecksumUseCase struct {
	verifier domain.ChecksumVerifier
}

// NewChecksumUseCase constructs a ChecksumUseCase with the given port.
func NewChecksumUseCase(verifier domain.ChecksumVerifier) *ChecksumUseCase {
	return &ChecksumUseCase{verifier: verifier}
}

// Run validates every .CHECKSUM sidecar found under dataPath.
// Configuration is read from the provided AegisConfig (loaded from YAML).
// Returns the number of failures.
func (uc *ChecksumUseCase) Run(dataPath string, cfg *config.AegisConfig) int {
	if cfg.Fetcher.SkipChecksumVerification {
		logger.Warn("Checksum verification is disabled via config - skipping integrity checks")
		return 0
	}

	failures := uc.verifier.VerifyAllChecksums(dataPath)

	if failures > 0 {
		logger.Infof("WARN %d checksum failure(s) detected - review errors above", failures)
	} else {
		logger.Info("All checksums passed")
	}

	return failures
}
