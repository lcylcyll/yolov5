package cropper

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCropImageWithTemplate_TemplateResolution focuses on the logic for finding template files.
// It does not test the actual cropping (which requires GoCV).
func TestCropImageWithTemplate_TemplateResolution(t *testing.T) {
	baseImageName := "test_image"

	// Create a dummy downloaded image file. It must exist for os.Stat in the function to pass
	// before template search begins. Content doesn't matter as gocv.IMRead should fail for it.
	dummyDownloadedImageDir := t.TempDir()
	dummyDownloadedImagePath := filepath.Join(dummyDownloadedImageDir, baseImageName+".jpg")
	f, err := os.Create(dummyDownloadedImagePath)
	if err != nil {
		t.Fatalf("Failed to create dummy downloaded image: %v", err)
	}
	f.Close() // Empty file is fine, gocv.IMRead will fail it.

	dummyOutputPath := filepath.Join(t.TempDir(), "output_cropped.jpg")

	tests := []struct {
		name                 string
		imageNameToTest      string // e.g., "test_image.jpg", "archive.tar.gz", "no_ext_image"
		templatesToCreate    []string // e.g., ["test_image_template.png", "test_image_template.jpg"] relative to templateDir
		expectedTemplateUsed string   // Expected suffix, e.g., "_template.png" or empty if none
		expectCropped        bool     // This will be false if IMRead fails, but we check if a template was found
		expectErrorContains  string   // Substring to check in error, or empty if no error expected *from template logic*
		                               // If a template is found, an error from gocv.IMRead(downloadedImagePath,...) is expected.
	}{
		{
			name:            "no templates exist",
			imageNameToTest: baseImageName + ".jpg",
			// expectCropped will be false, expectErrorContains will be empty (nil error)
		},
		{
			name:                 "only png template exists",
			imageNameToTest:      baseImageName + ".jpg",
			templatesToCreate:    []string{baseImageName + "_template.png"},
			expectedTemplateUsed: "_template.png",
			expectErrorContains:  fmt.Sprintf("failed to read downloaded image: %s", dummyDownloadedImagePath), // error from gocv.IMRead
		},
		{
			name:                 "only jpg template exists",
			imageNameToTest:      baseImageName + ".jpg",
			templatesToCreate:    []string{baseImageName + "_template.jpg"},
			expectedTemplateUsed: "_template.jpg",
			expectErrorContains:  fmt.Sprintf("failed to read downloaded image: %s", dummyDownloadedImagePath),
		},
		{
			name:                 "only jpeg template exists",
			imageNameToTest:      baseImageName + ".jpg",
			templatesToCreate:    []string{baseImageName + "_template.jpeg"},
			expectedTemplateUsed: "_template.jpeg",
			expectErrorContains:  fmt.Sprintf("failed to read downloaded image: %s", dummyDownloadedImagePath),
		},
		{
			name:                 "png takes precedence over jpg",
			imageNameToTest:      baseImageName + ".jpg",
			templatesToCreate:    []string{baseImageName + "_template.jpg", baseImageName + "_template.png"},
			expectedTemplateUsed: "_template.png",
			expectErrorContains:  fmt.Sprintf("failed to read downloaded image: %s", dummyDownloadedImagePath),
		},
		{
			name:                 "jpg takes precedence over jpeg (if png not present)",
			imageNameToTest:      baseImageName + ".jpg",
			templatesToCreate:    []string{baseImageName + "_template.jpeg", baseImageName + "_template.jpg"},
			expectedTemplateUsed: "_template.jpg",
			expectErrorContains:  fmt.Sprintf("failed to read downloaded image: %s", dummyDownloadedImagePath),
		},
		{
			name:            "image with multiple dots", // e.g. my.archive.tar.gz -> my.archive.tar_template.png
			imageNameToTest: "my.archive.tar.gz",
			templatesToCreate: []string{"my.archive.tar_template.png"},
			expectedTemplateUsed: "_template.png", // The function will look for "my.archive.tar_template.png"
			// Need to adjust dummyDownloadedImagePath for this test case
			expectErrorContains: "failed to read downloaded image:", // Path will change
		},
		{
			name:            "image with no extension",
			imageNameToTest: "imageNoExt",
			templatesToCreate: []string{"imageNoExt_template.jpg"},
			expectedTemplateUsed: "_template.jpg",
			// Need to adjust dummyDownloadedImagePath for this test case
			expectErrorContains: "failed to read downloaded image:", // Path will change
		},
		{
			name:                 "downloaded image does not exist",
			imageNameToTest:      "nonexistent_download_image.jpg", // This will be the path used
			templatesToCreate:    []string{"nonexistent_download_image_template.png"}, // Template exists
			expectedTemplateUsed: "_template.png",
			expectErrorContains:  "failed to read downloaded image:", // Error from gocv.IMRead
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			templateDir := t.TempDir()
			for _, tmplName := range tt.templatesToCreate {
				f, err := os.Create(filepath.Join(templateDir, tmplName))
				if err != nil {
					t.Fatalf("Failed to create dummy template %s: %v", tmplName, err)
				}
				f.Close() // Empty file, as gocv.IMRead on templateMat will be the next failure point
			}
			
			currentDownloadedImagePath := dummyDownloadedImagePath
			if strings.Contains(tt.name, "multiple dots") || strings.Contains(tt.name, "no extension") || strings.Contains(tt.name, "downloaded image does not exist") {
				// Adjust the dummy downloaded image path for these specific test cases
				// For "downloaded image does not exist", we use a path that won't be the pre-created dummy
				if strings.Contains(tt.name, "downloaded image does not exist") {
					currentDownloadedImagePath = filepath.Join(dummyDownloadedImageDir, tt.imageNameToTest)
                    // We don't create this file, so IMRead should fail.
				} else {
					// For multi-dot and no-ext, we still need a file to exist for the initial gocv.IMRead to be attempted
					currentDownloadedImagePath = filepath.Join(dummyDownloadedImageDir, tt.imageNameToTest)
					imgFile, err := os.Create(currentDownloadedImagePath)
					if err != nil {
						t.Fatalf("Failed to create dummy downloaded image for %s: %v", tt.imageNameToTest, err)
					}
					imgFile.Close()
					defer os.Remove(currentDownloadedImagePath) // Clean up this specific dummy file
				}
			}
            // If testing "downloaded image does not exist", ensure the error reflects that for the *downloaded* image.
            // The current `expectErrorContains` might be too generic for this case.
            if strings.Contains(tt.name, "downloaded image does not exist") {
                 tt.expectErrorContains = fmt.Sprintf("failed to read downloaded image: %s", currentDownloadedImagePath)
            } else if tt.expectedTemplateUsed != "" && !strings.Contains(tt.name, "downloaded image does not exist") {
				// If a template is expected to be found, and the downloaded image *should* exist (even if dummy)
				tt.expectErrorContains = fmt.Sprintf("failed to read downloaded image: %s", currentDownloadedImagePath)
			}


			cropped, err := CropImageWithTemplate(currentDownloadedImagePath, templateDir, dummyOutputPath)

			if tt.expectErrorContains == "" { // Case: No template found, or successful crop (not tested here)
				if err != nil {
					t.Errorf("CropImageWithTemplate() error = %v, want nil (or error from GoCV if template found)", err)
				}
				if cropped != tt.expectCropped { // expectCropped should be false if no template found
					t.Errorf("CropImageWithTemplate() cropped = %v, want %v", cropped, tt.expectCropped)
				}
			} else { // Case: Error expected (either from template logic or early GoCV fail)
				if err == nil {
					t.Errorf("CropImageWithTemplate() expected error containing %q, got nil", tt.expectErrorContains)
				} else if !strings.Contains(err.Error(), tt.expectErrorContains) {
					t.Errorf("CropImageWithTemplate() error = %q, want error containing %q", err.Error(), tt.expectErrorContains)
				}
				// If an error occurs (e.g. gocv.IMRead fails), cropped should be false.
				if cropped {
					t.Errorf("CropImageWithTemplate() cropped = true, want false when an error occurs like '%s'", tt.expectErrorContains)
				}
			}

			// Further check for template path selection if a template was expected to be "used"
			// This is indirectly tested by whether an error from gocv.IMRead occurs or not.
			// If no template was found, err is nil. If a template was found, IMRead(downloadedImg) fails.
			// A more direct test would require refactoring CropImageWithTemplate or checking logs.
			// For now, the error pattern (nil vs non-nil for IMRead failure) indicates template presence.
		})
	}
}
