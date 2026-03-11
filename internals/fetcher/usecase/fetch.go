package usecase

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/eichiarakaki/aegis/internals/config"
	"github.com/eichiarakaki/aegis/internals/fetcher/domain"
	"github.com/eichiarakaki/aegis/internals/logger"
)

const basePrefix = "data/futures/um/daily/"

// FetchUseCase orchestrates the discovery and download of all remote files.
type FetchUseCase struct {
	lister     domain.ObjectLister
	downloader domain.FileDownloader
}

// NewFetchUseCase constructs a FetchUseCase with the given ports.
func NewFetchUseCase(lister domain.ObjectLister, downloader domain.FileDownloader) *FetchUseCase {
	return &FetchUseCase{lister: lister, downloader: downloader}
}

// Run lists all objects for every symbol/dataType/interval combination derived
// from the CLI arguments, then downloads them concurrently using a worker pool.
// Only files whose embedded date falls within dateRange are downloaded.
// intervals is only used when dataTypes contains "klines"; it is ignored otherwise.
// Returns the total number of files queued.
func (uc *FetchUseCase) Run(
	dataPath string,
	symbols []string,
	dataTypes []string,
	intervals []string,
	dateRange domain.DateRange,
	cfg *config.AegisConfig,
) int {
	prefixes := buildPrefixes(dataPath, symbols, dataTypes, intervals)
	jobs := make(chan domain.Job, 1000)

	var wg sync.WaitGroup
	for i := 0; i < cfg.Fetcher.Download.MaxConcurrentDownloads; i++ {
		wg.Add(1)
		go uc.worker(i+1, jobs, &wg, cfg.Fetcher.Download.OverwriteDownloadedFiles, dateRange)
	}

	totalFiles := 0

	for _, p := range prefixes {
		logger.Infof("Listing: %s", p.S3Prefix)

		keys, err := uc.lister.ListObjects(p.S3Prefix)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERR] %v\n", err)
			continue
		}

		filtered := filterKeys(keys)
		logger.Infof("Found %d files", len(filtered))
		totalFiles += len(filtered)

		for _, k := range filtered {
			jobs <- domain.Job{Key: k, DestDir: p.DestDir}
		}

		time.Sleep(300 * time.Millisecond)
	}

	close(jobs)
	wg.Wait()

	return totalFiles
}

// worker consumes jobs from the channel, calling the downloader for each one.
func (uc *FetchUseCase) worker(
	id int,
	jobs <-chan domain.Job,
	wg *sync.WaitGroup,
	overwriteDownloadedFiles bool,
	dateRange domain.DateRange,
) {
	defer wg.Done()
	for j := range jobs {
		if err := uc.downloader.DownloadFile(j.Key, j.DestDir, overwriteDownloadedFiles, dateRange); err != nil {
			fmt.Fprintf(os.Stderr, "[ERR] worker %d: %v\n", id, err)
		}
	}
}

// filterKeys retains only .zip, .csv, and .CHECKSUM files.
func filterKeys(keys []string) []string {
	var out []string
	for _, k := range keys {
		if strings.HasSuffix(k, ".zip") ||
			strings.HasSuffix(k, ".csv") ||
			strings.HasSuffix(k, ".CHECKSUM") {
			out = append(out, k)
		}
	}
	return out
}

// buildPrefixes constructs all (S3 prefix, local destination) pairs for every
// combination of symbol, data type, and interval (where applicable).
// All values come from the CLI — no config defaults are applied here.
func buildPrefixes(
	dataPath string,
	symbols []string,
	dataTypes []string,
	intervals []string,
) []domain.Prefix {
	var prefixes []domain.Prefix

	for _, sym := range symbols {
		symUpper := strings.ToUpper(sym)
		for _, dt := range dataTypes {
			switch strings.ToLower(dt) {
			case "klines":
				// intervals is guaranteed non-empty by CLI validation when klines is requested.
				for _, interval := range intervals {
					prefixes = append(prefixes, domain.Prefix{
						S3Prefix: fmt.Sprintf("%s%s/%s/%s/", basePrefix, dt, symUpper, interval),
						DestDir:  fmt.Sprintf("%s/%s/%s/%s", dataPath, symUpper, dt, interval),
					})
				}
			default:
				prefixes = append(prefixes, domain.Prefix{
					S3Prefix: fmt.Sprintf("%s%s/%s/", basePrefix, dt, symUpper),
					DestDir:  fmt.Sprintf("%s/%s/%s", dataPath, symUpper, dt),
				})
			}
		}
	}

	return prefixes
}
