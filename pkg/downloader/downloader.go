package downloader

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// DownloadImage downloads an image from a given URL and saves it to the specified filepath.
// It creates the destination directory if it doesn't exist.
func DownloadImage(url string, targetFilepath string) error {
	// Make GET request
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to make GET request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-200 status code (%d) from %s", resp.StatusCode, url)
	}

	// Ensure destination directory exists
	dir := filepath.Dir(targetFilepath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Create output file
	out, err := os.Create(targetFilepath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", targetFilepath, err)
	}
	defer out.Close()

	// Copy response body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to copy response body to file %s: %w", targetFilepath, err)
	}

	return nil
}
