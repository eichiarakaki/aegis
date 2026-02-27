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
// Returns the number of failures.
func (uc *ExtractUseCase) Run(dataPath string) int {
	cfg := config.LoadAegisFetcher()

	failures := uc.extractor.UnzipAll(dataPath, cfg.Extraction.RemoveAfterExtraction, cfg.Extraction.OverrideExtractedFiles, cfg.Extraction.Enable)

	if failures > 0 {
		logger.Infof("WARN %d extraction failure(s) — review errors above", failures)
	} else {
		logger.Info("All archives extracted successfully")
	}

	return failures
}
