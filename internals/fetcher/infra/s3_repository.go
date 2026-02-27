package infra

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"time"
)

const s3BaseURL = "https://s3-ap-northeast-1.amazonaws.com/data.binance.vision"

// s3ListBucketResult represents the top-level XML response from the S3 listing API.
type s3ListBucketResult struct {
	XMLName     xml.Name    `xml:"ListBucketResult"`
	IsTruncated bool        `xml:"IsTruncated"`
	NextMarker  string      `xml:"NextMarker"`
	Contents    []s3Content `xml:"Contents"`
}

// s3Content represents a single object entry in the S3 XML response.
type s3Content struct {
	Key  string `xml:"Key"`
	Size int64  `xml:"Size"`
}

// S3Repository implements domain.ObjectLister against the Binance S3 bucket.
type S3Repository struct{}

// NewS3Repository constructs an S3Repository.
func NewS3Repository() *S3Repository {
	return &S3Repository{}
}

// ListObjects calls the S3 XML API to list all keys under prefix,
// handling pagination automatically via the marker parameter.
func (r *S3Repository) ListObjects(prefix string) ([]string, error) {
	var keys []string
	marker := ""

	for {
		reqURL := fmt.Sprintf("%s?delimiter=/&prefix=%s", s3BaseURL, prefix)
		if marker != "" {
			reqURL += "&marker=" + marker
		}

		body, statusCode, err := doGetWithRetry(reqURL)
		if err != nil {
			return keys, fmt.Errorf("listing %s: %w", prefix, err)
		}

		if statusCode != http.StatusOK {
			preview := string(body)
			if len(preview) > 300 {
				preview = preview[:300]
			}
			return keys, fmt.Errorf("listing %s: HTTP %d: %s", prefix, statusCode, preview)
		}

		var result s3ListBucketResult
		if err := xml.Unmarshal(body, &result); err != nil {
			preview := string(body)
			if len(preview) > 300 {
				preview = preview[:300]
			}
			return keys, fmt.Errorf("XML parse for %s (HTTP %d, body: %s): %w",
				prefix, statusCode, preview, err)
		}

		for _, c := range result.Contents {
			keys = append(keys, c.Key)
		}

		if !result.IsTruncated {
			break
		}
		marker = result.NextMarker
		time.Sleep(200 * time.Millisecond)
	}

	return keys, nil
}
