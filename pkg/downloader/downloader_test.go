package downloader

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDownloadImage_Success tests a successful image download.
func TestDownloadImage_Success(t *testing.T) {
	expectedContent := "dummy image data"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/test.jpg" {
			w.Header().Set("Content-Type", "image/jpeg")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(expectedContent))
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	tempDir, err := ioutil.TempDir("", "downloader_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	targetFilename := "downloaded_image.jpg"
	targetFilepath := filepath.Join(tempDir, "subdir", targetFilename) // Test subdir creation

	err = DownloadImage(server.URL+"/test.jpg", targetFilepath)
	if err != nil {
		t.Errorf("DownloadImage(%q, %q) failed unexpectedly: %v", server.URL+"/test.jpg", targetFilepath, err)
	}

	if _, err := os.Stat(targetFilepath); os.IsNotExist(err) {
		t.Errorf("DownloadImage did not create the target file %q", targetFilepath)
	} else {
		content, readErr := ioutil.ReadFile(targetFilepath)
		if readErr != nil {
			t.Errorf("Failed to read downloaded file %s: %v", targetFilepath, readErr)
		}
		if string(content) != expectedContent {
			t.Errorf("Downloaded file content mismatch. Got %q, want %q", string(content), expectedContent)
		}
	}
}

// TestDownloadImage_InvalidURL tests behavior with a clearly malformed URL.
func TestDownloadImage_InvalidURL(t *testing.T) {
	invalidURL := "this-is-not-a-valid-url"
	tempFile, err := ioutil.TempFile("", "test_invalid_url_*.jpg")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	filePath := tempFile.Name()
	tempFile.Close() // Close because DownloadImage will try to create it.
	defer os.Remove(filePath)

	err = DownloadImage(invalidURL, filePath)
	if err == nil {
		t.Errorf("Expected an error for invalid URL %q, but got nil", invalidURL)
	}
}

// TestDownloadImage_BadStatusCode tests non-200 status codes.
func TestDownloadImage_BadStatusCode(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
		path       string
	}{
		{"NotFound", http.StatusNotFound, "/notfound.jpg"},
		{"ServerError", http.StatusInternalServerError, "/servererror.jpg"},
		{"Unauthorized", http.StatusUnauthorized, "/unauthorized.jpg"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == tc.path {
					w.WriteHeader(tc.statusCode)
				} else {
					http.NotFound(w, r) // Should not happen if test URL is correct
				}
			}))
			defer server.Close()

			tempFile, err := ioutil.TempFile("", "test_status_*.jpg")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			filePath := tempFile.Name()
			tempFile.Close()
			defer os.Remove(filePath)

			err = DownloadImage(server.URL+tc.path, filePath)
			if err == nil {
				t.Errorf("Expected an error for URL %q (status %d), but got nil", server.URL+tc.path, tc.statusCode)
			} else {
				// Check if the error message contains the status code
				if !strings.Contains(err.Error(), fmt.Sprintf("%d", tc.statusCode)) {
					t.Errorf("Error message %q does not contain expected status code %d", err.Error(), tc.statusCode)
				}
				t.Logf("Received expected error for status %d: %v", tc.statusCode, err)
			}

			// Ensure file was not created (or is empty if created before error)
			if _, statErr := os.Stat(filePath); statErr == nil {
				content, _ := ioutil.ReadFile(filePath)
				if len(content) > 0 {
					t.Errorf("File %s was created and contains data for a non-200 response", filePath)
				}
			}
		})
	}
}

// TestDownloadImage_NetworkError attempts to simulate a network error.
// This uses a non-routable IP address. Behavior might vary across systems/networks.
func TestDownloadImage_NetworkError(t *testing.T) {
	// Using a "TEST-NET-1" address, which should not be routable on the public internet.
	// See RFC 5737 for details.
	nonRoutableURL := "http://192.0.2.1/dummy.jpg"

	tempFile, err := ioutil.TempFile("", "test_network_error_*.jpg")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	filePath := tempFile.Name()
	tempFile.Close()
	defer os.Remove(filePath)

	err = DownloadImage(nonRoutableURL, filePath)
	if err == nil {
		// This might not be an error on all systems (e.g., if DNS resolution times out quickly
		// or if system routes TEST-NET-1 locally). The goal is for http.Get to fail.
		t.Logf("DownloadImage(%q) did not return an error as expected for a non-routable URL. This test might be environment-dependent.", nonRoutableURL)
	} else {
		t.Logf("DownloadImage(%q) returned an error as expected: %v", nonRoutableURL, err)
	}
}

// TestDownloadImage_FileSystemError_DirCreation tests when the destination directory cannot be created.
func TestDownloadImage_FileSystemError_DirCreation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("content"))
	}))
	defer server.Close()

	// Create a file where a directory is supposed to be created by DownloadImage
	tempFileAsDirBlocker, err := ioutil.TempFile("", "dirblocker")
	if err != nil {
		t.Fatalf("Failed to create temp file to block directory creation: %v", err)
	}
	tempFileAsDirBlocker.Close() // Close it so it's just a file path
	defer os.Remove(tempFileAsDirBlocker.Name())

	// Target path where DownloadImage will try to create a subdir, but its parent is a file.
	targetFilepath := filepath.Join(tempFileAsDirBlocker.Name(), "sub", "image.jpg")

	err = DownloadImage(server.URL+"/image.jpg", targetFilepath)
	if err == nil {
		t.Errorf("Expected a file system error when creating directory %s, but got nil", filepath.Dir(targetFilepath))
	} else {
		t.Logf("Received expected file system error: %v", err)
		// More specific error check could be added here, e.g., "mkdir", "not a directory"
	}
}

// TestDownloadImage_FileSystemError_FileCreation tests when the destination file cannot be created.
func TestDownloadImage_FileSystemError_FileCreation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("content"))
	}))
	defer server.Close()

	// Create a directory where the file is supposed to be.
	// This will cause os.Create(targetFilepath) to fail if targetFilepath is a directory.
	tempDirAsFileBlocker, err := ioutil.TempDir("", "fileblocker_dir_")
	if err != nil {
		t.Fatalf("Failed to create temp dir to block file creation: %v", err)
	}
	defer os.RemoveAll(tempDirAsFileBlocker)

	targetFilepath := tempDirAsFileBlocker // Target is the directory itself

	err = DownloadImage(server.URL+"/image.jpg", targetFilepath)
	if err == nil {
		t.Errorf("Expected a file system error when creating file %s (it's a dir), but got nil", targetFilepath)
	} else {
		t.Logf("Received expected file system error for file creation: %v", err)
		// Error message should ideally indicate "is a directory" or similar on POSIX
	}
}
