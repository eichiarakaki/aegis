/*
The main function of the Aegis Fetcher program: This program is responsible for fetching data from various sources and processing it for use in the Aegis system. It will include functionality for connecting to APIs, databases, or other data sources, as well as handling any necessary data transformation or cleaning before passing the data on to other components of the Aegis system.
*/

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/eichiarakaki/aegis/internals/config"
	"github.com/eichiarakaki/aegis/internals/fetcher/infra"
	"github.com/eichiarakaki/aegis/internals/fetcher/usecase"
	"github.com/eichiarakaki/aegis/internals/logger"
)

// main wires the infrastructure adapters
// to the use cases and runs the three pipeline phases sequentially:
//  1. Download  — fetch all listed objects from Binance S3/CDN.
//  2. Verify    — validate SHA-256 checksums for every downloaded file.
//  3. Extract   — decompress every .zip archive in the output directory.
func main() {
	cfg, err := config.LoadAegis()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		return
	}
	dataPath := cfg.DataPath

	logger.Infof("Output dir: %s", dataPath)
	logger.Info(strings.Repeat("=", 60))

	// Wire infrastructure adapters
	s3Repo := infra.NewS3Repository()
	downloader := infra.NewCDNDownloader()
	verifier := infra.NewSHA256Verifier()
	extractor := infra.NewZipExtractor()

	// ── Phase 1: Download ────────────────────────────────────────────────────
	logger.Info("PHASE 1/3 — Downloading files")
	fetchUC := usecase.NewFetchUseCase(s3Repo, downloader)
	total := fetchUC.Run(dataPath)
	logger.Infof("Download complete — queued %d files", total)
	logger.Info(strings.Repeat("=", 60))

	// ── Phase 2: Checksum verification ───────────────────────────────────────
	logger.Info("PHASE 2/3 — Verifying checksums")
	checksumUC := usecase.NewChecksumUseCase(verifier)
	checksumUC.Run(dataPath)
	logger.Info(strings.Repeat("=", 60))

	// ── Phase 3: Extraction ──────────────────────────────────────────────────
	logger.Info("PHASE 3/3 — Extracting zip archives")
	extractUC := usecase.NewExtractUseCase(extractor)
	extractUC.Run(dataPath)
	logger.Info(strings.Repeat("=", 60))

	logger.Infof("Done! Output directory: %s", dataPath)
}
