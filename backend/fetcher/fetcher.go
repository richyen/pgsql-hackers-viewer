package fetcher

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	// ArchiveBaseURL is the base URL for pgsql-hackers monthly mbox archives.
	ArchiveBaseURL = "https://www.postgresql.org/list/pgsql-hackers/mbox"
	// UserAgent identifies the client to the archive server.
	UserAgent = "pgsql-hackers-viewer/1.0"
)

// DownloadMonth downloads the monthly mbox file for the given year and month
// from the PostgreSQL mailing list archive and saves it to dataDir.
// username/password are used for HTTP Basic Auth (required by postgresql.org for raw mbox).
// Filename format: pgsql-hackers.YYYYMM (e.g. pgsql-hackers.202512).
// Returns the local file path, or error if download fails.
func DownloadMonth(dataDir, username, password string, year, month int) (string, error) {
	name := fmt.Sprintf("pgsql-hackers.%04d%02d", year, month)
	url := ArchiveBaseURL + "/" + name
	destPath := filepath.Join(dataDir, name)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", UserAgent)
	if username != "" && password != "" {
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)))
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download %s: status %s", url, resp.Status)
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return "", fmt.Errorf("create data dir: %w", err)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("create file %s: %w", destPath, err)
	}
	defer f.Close()

	n, err := io.Copy(f, resp.Body)
	if err != nil {
		os.Remove(destPath)
		return "", fmt.Errorf("write %s: %w", destPath, err)
	}

	log.Printf("Downloaded %s (%d bytes) to %s", name, n, destPath)
	return destPath, nil
}
