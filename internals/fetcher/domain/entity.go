package domain

// Content represents a single object returned by the S3 listing API.
type Content struct {
	Key  string
	Size int64
}

// Job represents a single download task.
type Job struct {
	Key     string
	DestDir string
}

// Prefix holds an S3 prefix and its corresponding local destination directory.
type Prefix struct {
	S3Prefix string
	DestDir  string
}
