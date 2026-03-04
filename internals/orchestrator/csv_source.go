package orchestrator

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
)

// CSVDataSource implements DataSource for historical CSV files.
// It loads one file fully into memory at a time. When the current file
// is exhausted it loads the next one from the file list.
type CSVDataSource struct {
	topic    string
	dataType string
	priority int
	parse    ParseFunc
	files    []string // sorted chronological file paths

	fileIdx int      // index into files — next file to load
	rows    []RawRow // current file's parsed rows, fully in memory
	cursor  int      // next row to serve from rows
}

// NewCSVDataSource creates a CSVDataSource for the given topic.
// files must be sorted chronologically (oldest first).
func NewCSVDataSource(topic, dataType string, priority int, parse ParseFunc, files []string) *CSVDataSource {
	return &CSVDataSource{
		topic:    topic,
		dataType: dataType,
		priority: priority,
		parse:    parse,
		files:    files,
	}
}

// Topic implements DataSource.
func (s *CSVDataSource) Topic() string { return s.topic }

// DataType implements DataSource.
func (s *CSVDataSource) DataType() string { return s.dataType }

// Peek implements DataSource.
func (s *CSVDataSource) Peek() (int64, error) {
	if err := s.ensureLoaded(); err != nil {
		return 0, err
	}
	return s.rows[s.cursor].Timestamp, nil
}

// Drain implements DataSource.
func (s *CSVDataSource) Drain(ts int64) ([]RawRow, error) {
	if err := s.ensureLoaded(); err != nil {
		if err == ErrExhausted {
			return nil, nil
		}
		return nil, err
	}

	if s.rows[s.cursor].Timestamp != ts {
		return nil, nil
	}

	var out []RawRow
	for s.cursor < len(s.rows) && s.rows[s.cursor].Timestamp == ts {
		out = append(out, s.rows[s.cursor])
		s.cursor++
	}
	return out, nil
}

// ensureLoaded makes sure rows is populated and cursor is valid.
// If the current file is fully consumed it loads the next one.
func (s *CSVDataSource) ensureLoaded() error {
	for s.rows == nil || s.cursor >= len(s.rows) {
		if s.fileIdx >= len(s.files) {
			return ErrExhausted
		}
		if err := s.loadFile(s.files[s.fileIdx]); err != nil {
			// Log and skip unreadable files rather than aborting the whole source.
			s.fileIdx++
			return fmt.Errorf("csv_source: skip file %q: %w", s.files[s.fileIdx-1], err)
		}
		s.fileIdx++
	}
	return nil
}

// loadFile reads an entire CSV file into s.rows.
func (s *CSVDataSource) loadFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.ReuseRecord = false

	// Skip header row.
	if _, err := r.Read(); err != nil {
		return fmt.Errorf("read header: %w", err)
	}

	var rows []RawRow
	lineNum := 1 // 1-based, header was line 1
	for {
		lineNum++
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Skip malformed lines, don't abort the file.
			continue
		}

		ts, payload, err := s.parse(record)
		if err != nil {
			// Log skip — in production wire this to the session logger.
			_ = fmt.Sprintf("csv_source: %s line %d: %v (skipped)", path, lineNum, err)
			continue
		}

		rows = append(rows, RawRow{
			Timestamp: ts,
			DataType:  s.dataType,
			Priority:  s.priority,
			Topic:     s.topic,
			Payload:   payload,
		})
	}

	if len(rows) == 0 {
		return fmt.Errorf("no valid rows in %q", path)
	}

	s.rows = rows
	s.cursor = 0
	return nil
}
