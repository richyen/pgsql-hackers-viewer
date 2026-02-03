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
// If skipIfExists is true and the file already exists, it will return the path without downloading.
func DownloadMonth(dataDir, username, password string, year, month int, skipIfExists bool) (string, error) {
	name := fmt.Sprintf("pgsql-hackers.%04d%02d", year, month)
	url := ArchiveBaseURL + "/" + name
	destPath := filepath.Join(dataDir, name)

	// Check if file already exists and we should skip download
	if skipIfExists {
		if _, err := os.Stat(destPath); err == nil {
			log.Printf("Using cached mbox file: %s", destPath)
			return destPath, nil
		}
	}

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

// MonthDownload represents a month to download and its result
type MonthDownload struct {
	Year  int
	Month int
}

// MonthResult represents the result of downloading a month
type MonthResult struct {
	Year     int
	Month    int
	Path     string
	Error    error
	Duration time.Duration
}

// DownloadMonthsConcurrent downloads multiple months in parallel with a limited number of workers.
// Returns a slice of results (one per month) in the order they complete.
// If skipIfExists is true, existing files will not be re-downloaded.
func DownloadMonthsConcurrent(dataDir, username, password string, months []MonthDownload, workers int, skipIfExists bool) []MonthResult {
	if workers <= 0 {
		workers = 3 // default to 3 workers
	}

	jobs := make(chan MonthDownload, len(months))
	results := make(chan MonthResult, len(months))

	// Start worker pool
	for w := 0; w < workers; w++ {
		go downloadWorker(jobs, results, dataDir, username, password, skipIfExists)
	}

	// Send jobs to workers
	for _, month := range months {
		jobs <- month
	}
	close(jobs)

	// Collect results
	var out []MonthResult
	for i := 0; i < len(months); i++ {
		result := <-results
		out = append(out, result)
	}
	close(results)

	return out
}

// downloadWorker processes download jobs from the jobs channel
func downloadWorker(jobs <-chan MonthDownload, results chan<- MonthResult, dataDir, username, password string, skipIfExists bool) {
	for job := range jobs {
		start := time.Now()
		path, err := DownloadMonth(dataDir, username, password, job.Year, job.Month, skipIfExists)
		results <- MonthResult{
			Year:     job.Year,
			Month:    job.Month,
			Path:     path,
			Error:    err,
			Duration: time.Since(start),
		}
	}
}
