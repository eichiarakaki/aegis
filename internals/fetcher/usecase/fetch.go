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

// Run lists all objects for every symbol/dataType/interval combination,
// then downloads them concurrently using a worker pool.
// Only files whose embedded date falls within [StartDate, EndDate] are downloaded.
// Returns the total number of files queued.
func (uc *FetchUseCase) Run(dataPath string) int {
	cfg := config.LoadAegisFetcher()

	if !cfg.Download.Enable {
		logger.Info("Download disabled in config — skipping download phase")
		return 0
	}

	// Parse date range from config strings ("2024-01-21" format)
	dateRange, err := parseDateRange(cfg.Download.StartDate, cfg.Download.EndDate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERR] invalid date range in config: %v\n", err)
		return 0
	}

	logger.Infof("Date range: %s → %s",
		dateRange.Start.Format("2006-01-02"),
		dateRange.End.Format("2006-01-02"),
	)

	prefixes := buildPrefixes(dataPath, cfg)
	jobs := make(chan domain.Job, 1000)

	var wg sync.WaitGroup
	for i := 0; i < cfg.Download.MaxConcurrentDownloads; i++ {
		wg.Add(1)
		go uc.worker(i+1, jobs, &wg, cfg.Download.OverwriteDownloadedFiles, dateRange)
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

// parseDateRange parses two "2006-01-02" strings into a domain.DateRange.
func parseDateRange(start, end string) (domain.DateRange, error) {
	const layout = "2006-01-02"

	s, err := time.Parse(layout, start)
	if err != nil {
		return domain.DateRange{}, fmt.Errorf("parsing StartDate %q: %w", start, err)
	}

	e, err := time.Parse(layout, end)
	if err != nil {
		return domain.DateRange{}, fmt.Errorf("parsing EndDate %q: %w", end, err)
	}

	if e.Before(s) {
		return domain.DateRange{}, fmt.Errorf("EndDate %q is before StartDate %q", end, start)
	}

	return domain.DateRange{Start: s, End: e}, nil
}

// worker consumes jobs from the channel, calling the downloader for each one.
func (uc *FetchUseCase) worker(id int, jobs <-chan domain.Job, wg *sync.WaitGroup, overwriteDownloadedFiles bool, dateRange domain.DateRange) {
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
// combination of symbol, data type, and kline interval (where applicable).
func buildPrefixes(dataPath string, cfg *config.AegisFetcherConfig) []domain.Prefix {
	var prefixes []domain.Prefix

	for _, sym := range cfg.Cryptocurrencies {
		for _, dt := range sym.DataTypes {
			switch dt {
			case "klines":
				for _, interval := range sym.Intervals {
					prefixes = append(prefixes, domain.Prefix{
						S3Prefix: fmt.Sprintf("%s%s/%s/%s/", basePrefix, dt, sym.Symbol, interval),
						DestDir:  fmt.Sprintf("%s/%s/%s/%s", dataPath, sym.Symbol, dt, interval),
					})
				}
			default:
				prefixes = append(prefixes, domain.Prefix{
					S3Prefix: fmt.Sprintf("%s%s/%s/", basePrefix, dt, sym.Symbol),
					DestDir:  fmt.Sprintf("%s/%s/%s", dataPath, sym.Symbol, dt),
				})
			}
		}
	}

	return prefixes
}
