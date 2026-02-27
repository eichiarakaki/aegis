package domain

// ObjectLister is the port for listing remote objects under a given prefix.
type ObjectLister interface {
	ListObjects(prefix string) ([]string, error)
}

// FileDownloader is the port for downloading a remote object to disk.
type FileDownloader interface {
	DownloadFile(key, destDir string) error
}

// ChecksumVerifier is the port for validating file integrity.
type ChecksumVerifier interface {
	VerifyAllChecksums(dataPath string) (failures int)
}

// Extractor is the port for decompressing downloaded archives.
type Extractor interface {
	UnzipAll(dataPath string) (failures int)
}
