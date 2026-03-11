package usecase

import (
	"github.com/eichiarakaki/aegis/internals/config"
	"github.com/eichiarakaki/aegis/internals/fetcher/domain"
	"github.com/eichiarakaki/aegis/internals/logger"
)

// ExtractUseCase orchestrates the extraction of all downloaded archives.
type ExtractUseCase struct {
	extractor domain.Extractor
}

// NewExtractUseCase constructs an ExtractUseCase with the given port.
func NewExtractUseCase(extractor domain.Extractor) *ExtractUseCase {
	return &ExtractUseCase{extractor: extractor}
}

// Run extracts every .zip archive found under dataPath.
// Configuration is read from the provided AegisConfig (loaded from YAML).
// Returns the number of failures.
func (uc *ExtractUseCase) Run(dataPath string, cfg *config.AegisConfig) int {
	failures := uc.extractor.UnzipAll(
		dataPath,
		cfg.Fetcher.Extraction.RemoveAfterExtraction,
		cfg.Fetcher.Extraction.OverwriteExtractedFiles,
		cfg.Fetcher.Extraction.Enable,
	)

	if failures > 0 {
		logger.Infof("WARN %d extraction failure(s) - review errors above", failures)
	} else {
		logger.Info("All archives extracted successfully")
	}

	return failures
}
