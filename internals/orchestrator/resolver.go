package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
}

// NewFileResolver creates a FileResolver with the given data root directory.
func NewFileResolver(dataRoot string) *FileResolver {
	return &FileResolver{dataRoot: expandHome(dataRoot)}
}

// Resolve returns the sorted list of CSV files for the given TopicParts.
//
// klines:     <root>/<symbol>/klines/<timeframe>/<symbol>-<timeframe>-*.csv
// flat types: <root>/<symbol>/<data_type>/<symbol>-<data_type>-*.csv
func (r *FileResolver) Resolve(tp TopicParts) ([]string, error) {
	var pattern string
	if tp.Timeframe != "" {
		// e.g. /data/BTCUSDT/klines/1m/BTCUSDT-1m-*.csv
		pattern = filepath.Join(
			r.dataRoot,
			tp.Symbol,
			tp.DataType,
			tp.Timeframe,
			fmt.Sprintf("%s-%s-*.csv", tp.Symbol, tp.Timeframe),
		)
	} else {
		// e.g. /data/BTCUSDT/aggTrades/BTCUSDT-aggTrades-*.csv
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

	// Lexicographic sort = chronological order since filenames end in YYYY-MM-DD.csv
	sort.Strings(matches)
	return matches, nil
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
