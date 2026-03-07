package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// TopicParts holds the parsed components of a topic string.
// Topic format: <data_type>.<symbol>[.<timeframe>]
// Examples:
//
//	klines.BTCUSDT.1m   → DataType=klines, Symbol=BTCUSDT, Timeframe=1m
//	trades.BTCUSDT      → DataType=trades, Symbol=BTCUSDT, Timeframe=""
type TopicParts struct {
	DataType  string
	Symbol    string
	Timeframe string // empty for flat data types
}

// ParseTopic parses a component topic string into its parts.
func ParseTopic(topic string) (TopicParts, error) {
	parts := strings.SplitN(topic, ".", 3)
	if len(parts) < 2 {
		return TopicParts{}, fmt.Errorf("resolver: invalid topic %q: expected at least <data_type>.<symbol>", topic)
	}

	tp := TopicParts{
		DataType: parts[0],
		Symbol:   parts[1],
	}
	if len(parts) == 3 {
		tp.Timeframe = parts[2]
	}
	return tp, nil
}

// NATSTopic builds the full NATS topic from a session ID and TopicParts.
// aegis.<session_id>.<data_type>.<symbol>[.<timeframe>]
func NATSTopic(sessionID string, tp TopicParts) string {
	if tp.Timeframe != "" {
		return fmt.Sprintf("aegis.%s.%s.%s.%s", sessionID, tp.DataType, tp.Symbol, tp.Timeframe)
	}
	return fmt.Sprintf("aegis.%s.%s.%s", sessionID, tp.DataType, tp.Symbol)
}

// FileResolver resolves a TopicParts to an ordered list of CSV file paths on disk.
type FileResolver struct {
	dataRoot string
	fromTS   int64 // unix ms, 0 = no lower bound
	toTS     int64 // unix ms, 0 = no upper bound
}

// NewFileResolver creates a FileResolver with the given data root directory.
func NewFileResolver(dataRoot string) *FileResolver {
	return &FileResolver{dataRoot: expandHome(dataRoot)}
}

// NewFileResolverWithRange creates a FileResolver that restricts results to
// files whose date falls within [fromTS, toTS] (unix ms, inclusive).
// Zero values mean no bound.
func NewFileResolverWithRange(dataRoot string, fromTS, toTS int64) *FileResolver {
	return &FileResolver{
		dataRoot: expandHome(dataRoot),
		fromTS:   fromTS,
		toTS:     toTS,
	}
}

// Resolve returns the sorted list of CSV files for the given TopicParts,
// filtered to only include files that could contain rows within [fromTS, toTS].
//
// File names end in YYYY-MM-DD.csv. A file for date D is included if:
//   - D >= from_date  (or fromTS == 0)
//   - D <= to_date    (or toTS == 0)
//
// The day boundary is conservative: a file for 2024-01-15 could contain
// timestamps from 00:00 to 23:59 UTC on that day. We include boundary files
// rather than risk excluding valid rows.
func (r *FileResolver) Resolve(tp TopicParts) ([]string, error) {
	var pattern string
	if tp.Timeframe != "" {
		pattern = filepath.Join(
			r.dataRoot,
			tp.Symbol,
			tp.DataType,
			tp.Timeframe,
			fmt.Sprintf("%s-%s-*.csv", tp.Symbol, tp.Timeframe),
		)
	} else {
		pattern = filepath.Join(
			r.dataRoot,
			tp.Symbol,
			tp.DataType,
			fmt.Sprintf("%s-%s-*.csv", tp.Symbol, tp.DataType),
		)
	}

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("resolver: glob %q: %w", pattern, err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("resolver: no files found for pattern %q", pattern)
	}

	sort.Strings(matches)

	// No range filter — return all files.
	if r.fromTS == 0 && r.toTS == 0 {
		return matches, nil
	}

	fromDay := msToDay(r.fromTS) // zero time if fromTS==0
	toDay := msToDay(r.toTS)     // zero time if toTS==0

	var filtered []string
	for _, path := range matches {
		fileDay, ok := extractFileDay(path)
		if !ok {
			// Can't parse the date — include conservatively.
			filtered = append(filtered, path)
			continue
		}

		// Exclude files strictly before the from date.
		if !fromDay.IsZero() && fileDay.Before(fromDay) {
			continue
		}
		// Exclude files strictly after the to date.
		if !toDay.IsZero() && fileDay.After(toDay) {
			continue
		}

		filtered = append(filtered, path)
	}

	if len(filtered) == 0 {
		return nil, fmt.Errorf("resolver: no files found in range for pattern %q", pattern)
	}

	return filtered, nil
}

// extractFileDay parses the YYYY-MM-DD suffix from a CSV filename.
// Returns (day, true) on success, (zero, false) on failure.
func extractFileDay(path string) (time.Time, bool) {
	base := filepath.Base(path)
	// Strip .csv extension.
	name := strings.TrimSuffix(base, ".csv")
	// Date is always the last 10 characters: YYYY-MM-DD
	if len(name) < 10 {
		return time.Time{}, false
	}
	datePart := name[len(name)-10:]
	t, err := time.Parse("2006-01-02", datePart)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// msToDay converts a unix millisecond timestamp to a UTC calendar day (time.Time at midnight UTC).
// Returns zero time if ms == 0.
func msToDay(ms int64) time.Time {
	if ms == 0 {
		return time.Time{}
	}
	t := time.UnixMilli(ms).UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// expandHome replaces a leading ~ with the actual home directory.
func expandHome(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[1:])
}
