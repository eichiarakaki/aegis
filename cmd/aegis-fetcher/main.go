package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/eichiarakaki/aegis/internals/config"
	"github.com/eichiarakaki/aegis/internals/fetcher/domain"
	"github.com/eichiarakaki/aegis/internals/fetcher/infra"
	"github.com/eichiarakaki/aegis/internals/fetcher/usecase"
	"github.com/eichiarakaki/aegis/internals/logger"
)

const dateLayout = "2006-01-02"

// cliArgs holds the parsed command-line arguments.
type cliArgs struct {
	Symbols   []string
	DataTypes []string
	Intervals []string
	SavePath  string
	DateRange domain.DateRange
}

func main() {
	args, err := parseCLI()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		flag.Usage()
		os.Exit(1)
	}

	aegisCfg, err := config.LoadAegis()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger.Infof("Output dir:  %s", args.SavePath)
	logger.Infof("Symbols:     %s", strings.Join(args.Symbols, ", "))
	logger.Infof("Data types:  %s", strings.Join(args.DataTypes, ", "))
	if len(args.Intervals) > 0 {
		logger.Infof("Intervals:   %s", strings.Join(args.Intervals, ", "))
	}
	logger.Infof("Date range:  %s → %s",
		args.DateRange.Start.Format(dateLayout),
		args.DateRange.End.Format(dateLayout),
	)
	logger.Info(strings.Repeat("=", 60))

	// Wire infrastructure adapters.
	s3Repo := infra.NewS3Repository()
	downloader := infra.NewCDNDownloader()
	verifier := infra.NewSHA256Verifier()
	extractor := infra.NewZipExtractor()

	// ── Phase 1: Download ────────────────────────────────────────────────────
	logger.Info("PHASE 1/3 - Downloading files")
	fetchUC := usecase.NewFetchUseCase(s3Repo, downloader)
	total := fetchUC.Run(args.SavePath, args.Symbols, args.DataTypes, args.Intervals, args.DateRange, aegisCfg)
	logger.Infof("Download complete - queued %d files", total)
	logger.Info(strings.Repeat("=", 60))

	// ── Phase 2: Checksum verification ───────────────────────────────────────
	logger.Info("PHASE 2/3 - Verifying checksums")
	checksumUC := usecase.NewChecksumUseCase(verifier)
	checksumUC.Run(args.SavePath, aegisCfg)
	logger.Info(strings.Repeat("=", 60))

	// ── Phase 3: Extraction ──────────────────────────────────────────────────
	logger.Info("PHASE 3/3 - Extracting zip archives")
	extractUC := usecase.NewExtractUseCase(extractor)
	extractUC.Run(args.SavePath, aegisCfg)
	logger.Info(strings.Repeat("=", 60))

	logger.Infof("Done! Output directory: %s", args.SavePath)
}

// parseCLI parses and validates all command-line flags.
func parseCLI() (*cliArgs, error) {
	symbolsFlag := flag.String("symbol", "", "Comma-separated list of symbols (required)\n\te.g. BTCUSDT,ETHUSDT")
	dataTypesFlag := flag.String("datatype", "", "Comma-separated list of data types (required)\n\te.g. klines,aggTrades,bookTicker,trades")
	intervalsFlag := flag.String("interval", "", "Comma-separated kline intervals (required when datatype includes klines)\n\te.g. 1m,5m,1h,1d")
	fromFlag := flag.String("from", "", "Start date, inclusive, YYYY-MM-DD (required)\n\te.g. 2024-01-01")
	toFlag := flag.String("to", "", "End date, inclusive, YYYY-MM-DD (required)\n\te.g. 2024-03-31")
	saveFlag := flag.String("save", "", "Directory where downloaded files will be saved (required)\n\te.g. /mnt/data/binance")

	flag.Usage = printUsage
	flag.Parse()

	// Validate required flags.
	if *symbolsFlag == "" {
		return nil, fmt.Errorf("--symbol is required")
	}
	if *dataTypesFlag == "" {
		return nil, fmt.Errorf("--datatype is required")
	}
	if *fromFlag == "" {
		return nil, fmt.Errorf("--from is required")
	}
	if *toFlag == "" {
		return nil, fmt.Errorf("--to is required")
	}
	if *saveFlag == "" {
		return nil, fmt.Errorf("--save is required")
	}

	dataTypes := splitAndTrim(*dataTypesFlag)
	intervals := splitAndTrim(*intervalsFlag)

	// --interval is required only when klines is one of the requested data types.
	if containsKlines(dataTypes) && len(intervals) == 0 {
		return nil, fmt.Errorf("--interval is required when --datatype includes \"klines\"\n  e.g. --interval 1m,5m,1h")
	}

	startDate, err := parseExactDate(*fromFlag, "from")
	if err != nil {
		return nil, err
	}

	endDate, err := parseExactDate(*toFlag, "to")
	if err != nil {
		return nil, err
	}

	if endDate.Before(startDate) {
		return nil, fmt.Errorf("--to (%s) must not be before --from (%s)", *toFlag, *fromFlag)
	}

	savePath := strings.TrimRight(*saveFlag, "/")
	if err := os.MkdirAll(savePath, 0755); err != nil {
		return nil, fmt.Errorf("cannot create save directory %q: %w", savePath, err)
	}

	return &cliArgs{
		Symbols:   splitAndTrim(*symbolsFlag),
		DataTypes: dataTypes,
		Intervals: intervals,
		SavePath:  savePath,
		DateRange: domain.DateRange{
			Start: startDate,
			End:   endDate,
		},
	}, nil
}

// containsKlines reports whether the slice contains the "klines" data type.
func containsKlines(dataTypes []string) bool {
	for _, dt := range dataTypes {
		if strings.EqualFold(dt, "klines") {
			return true
		}
	}
	return false
}

// parseExactDate parses a date string and returns a descriptive error with a
// nearest-date suggestion when the value is not a valid YYYY-MM-DD date.
func parseExactDate(raw, flagName string) (time.Time, error) {
	t, err := time.Parse(dateLayout, raw)
	if err == nil {
		return t, nil
	}

	// Try common alternative formats to produce a helpful suggestion.
	alternatives := []string{
		"2006/01/02",
		"01/02/2006",
		"02-01-2006",
		"January 2, 2006",
		"Jan 2, 2006",
		"2006-1-2",
		"2006-01",
		"2006",
	}

	var suggestion string
	for _, layout := range alternatives {
		if parsed, altErr := time.Parse(layout, raw); altErr == nil {
			suggestion = closestDayBoundary(parsed, flagName)
			break
		}
	}

	msg := fmt.Sprintf(
		"--%s %q is not a valid date - expected format is YYYY-MM-DD (e.g. %s)",
		flagName, raw, time.Now().Format(dateLayout),
	)
	if suggestion != "" {
		msg += fmt.Sprintf("\n  Suggestion: --%s %s", flagName, suggestion)
	}

	return time.Time{}, fmt.Errorf("%s", msg)
}

// closestDayBoundary returns the nearest whole-day YYYY-MM-DD string.
// For --from it keeps the same calendar day; for --to it rounds up to the
// next day when the sub-day component exceeds 12 hours.
func closestDayBoundary(t time.Time, flagName string) string {
	day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	diff := t.UTC().Sub(day)
	if flagName == "to" && math.Abs(diff.Hours()) > 12 {
		day = day.AddDate(0, 0, 1)
	}
	return day.Format(dateLayout)
}

// splitAndTrim splits a comma-separated string and trims whitespace from each element.
func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// printUsage prints a friendly help message.
func printUsage() {
	fmt.Fprintf(os.Stderr, `aegis-fetcher - download market data from Binance Vision

Usage:
  aegis-fetcher --symbol <symbols> --datatype <types> [--interval <intervals>] \
                --from <date> --to <date> --save <path>

Required flags:
  --symbol    Comma-separated trading pairs
                e.g. BTCUSDT,ETHUSDT

  --datatype  Comma-separated data types
                e.g. klines,aggTrades,bookTicker,trades

  --from      Start date, inclusive, in YYYY-MM-DD format
                e.g. 2024-01-01

  --to        End date, inclusive, in YYYY-MM-DD format
                e.g. 2024-03-31

  --save      Directory where downloaded files will be saved
                e.g. /mnt/data/binance

Conditional flags:
  --interval  Comma-separated kline intervals - required when --datatype
              includes "klines", ignored otherwise
                e.g. 1m,5m,15m,1h,4h,1d

Notes:
  Dates must be exact YYYY-MM-DD values. Binance Vision stores one file per
  calendar day, so partial dates or timestamps are not accepted. If you pass
  a recognisable but incorrectly formatted date, a corrected suggestion will
  be shown.

  Checksum verification, extraction settings, concurrency, and overwrite
  behaviour are all configured via the aegis YAML config file.

Examples:
  aegis-fetcher \
    --symbol BTCUSDT \
    --datatype klines \
    --interval 1m,1h \
    --from 2024-01-01 \
    --to  2024-01-31 \
    --save ./data

  aegis-fetcher \
    --symbol BTCUSDT,ETHUSDT \
    --datatype klines,aggTrades \
    --interval 5m,1d \
    --from 2024-01-01 \
    --to  2024-12-31 \
    --save /mnt/binance

  aegis-fetcher \
    --symbol BTCUSDT \
    --datatype aggTrades \
    --from 2024-06-01 \
    --to  2024-06-30 \
    --save /mnt/binance

`)
}
