package cropper

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"

	"gocv.io/x/gocv"
)

// CropImageWithTemplate attempts to crop a downloaded image based on a template.
// downloadedImagePath: Path to the downloaded image.
// templateDir: Directory containing template files.
// outputPath: Path to save the cropped image.
//
// Template Naming Logic:
// For a downloaded image `filename.ext` (e.g., `example.jpg`),
// the function searches for templates in `templateDir` in the following order:
// 1. `filename_template.png`
// 2. `filename_template.jpg`
// 3. `filename_template.jpeg`
//
// Returns:
// cropped (bool): True if cropping was performed, false otherwise.
// err (error): An error if any issue occurred during the process. If no template is found,
//              it returns (false, nil) indicating no error and no cropping.
func CropImageWithTemplate(downloadedImagePath string, templateDir string, outputPath string) (cropped bool, err error) {
	// a. Extract the base filename
	baseNameWithExt := filepath.Base(downloadedImagePath)
	extension := filepath.Ext(baseNameWithExt)
	baseName := strings.TrimSuffix(baseNameWithExt, extension)

	// b. Try to find a corresponding template file
	templateExtensions := []string{".png", ".jpg", ".jpeg"}
	var templatePath string

	for _, tmplExt := range templateExtensions {
		potentialTemplate := filepath.Join(templateDir, baseName+"_template"+tmplExt)
		if _, statErr := os.Stat(potentialTemplate); statErr == nil {
			templatePath = potentialTemplate
			break
		}
	}

	// c. If no template is found, return cropped = false, err = nil
	if templatePath == "" {
		// It's not an error condition, just means no cropping needed for this image.
		// A log message here might be useful in a real application.
		// fmt.Printf("No template found for image %s in directory %s\n", baseNameWithExt, templateDir)
		return false, nil
	}

	// d. Load the downloaded image and the template image
	downloadedMat := gocv.IMRead(downloadedImagePath, gocv.IMReadColor)
	if downloadedMat.Empty() {
		return false, fmt.Errorf("failed to read downloaded image: %s", downloadedImagePath)
	}
	defer downloadedMat.Close()

	templateMat := gocv.IMRead(templatePath, gocv.IMReadColor)
	if templateMat.Empty() {
		return false, fmt.Errorf("failed to read template image: %s", templatePath)
	}
	defer templateMat.Close()

	// Ensure the template is not larger than the image
	if downloadedMat.Cols() < templateMat.Cols() || downloadedMat.Rows() < templateMat.Rows() {
		return false, fmt.Errorf("template dimensions (%dx%d) exceed image dimensions (%dx%d)",
			templateMat.Cols(), templateMat.Rows(), downloadedMat.Cols(), downloadedMat.Rows())
	}
	
	// e. Perform template matching
	// gocv.TmCcoeffNormed is a good general-purpose matching method.
	result := gocv.MatchTemplate(downloadedMat, templateMat, gocv.TmCcoeffNormed, gocv.NewMat())
	if result.Empty() {
		return false, fmt.Errorf("template matching result was empty")
	}
	defer result.Close()

	// f. Find the location of the best match
	_, _, _, maxLoc := gocv.MinMaxLoc(result)

	// g. Define the cropping rectangle
	// Top-left is maxLoc. Width and height are from templateMat.
	cropRect := image.Rect(maxLoc.X, maxLoc.Y, maxLoc.X+templateMat.Cols(), maxLoc.Y+templateMat.Rows())

	// h. Create the region (crop)
	// Ensure the rectangle is within the bounds of the downloadedMat
	// This check is crucial as maxLoc + templateMat dimensions might exceed downloadedMat bounds slightly
	// depending on the matching algorithm and image borders.
	if cropRect.Max.X > downloadedMat.Cols() || cropRect.Max.Y > downloadedMat.Rows() {
		// Adjust the rectangle if it exceeds bounds. This can happen with some matching results.
		// A more robust solution might involve checking why this happened or padding.
		// For now, we'll cap it to the image dimensions.
		cropRect.Max.X = min(cropRect.Max.X, downloadedMat.Cols())
		cropRect.Max.Y = min(cropRect.Max.Y, downloadedMat.Rows())
		// Ensure Min is still less than Max after adjustment
		if cropRect.Min.X >= cropRect.Max.X || cropRect.Min.Y >= cropRect.Max.Y {
			return false, fmt.Errorf("adjusted crop rectangle is invalid for image %s with template %s. Original maxLoc: %v", downloadedImagePath, templatePath, maxLoc)
		}
	}
	
	region := downloadedMat.Region(cropRect)
	if region.Empty() {
		return false, fmt.Errorf("failed to create cropped region for image %s with template %s", downloadedImagePath, templatePath)
	}
	defer region.Close() // region is a Mat, so it needs to be closed

	// Before saving, ensure the output directory exists
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return false, fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}

	// i. Save the cropped region
	if ok := gocv.IMWrite(outputPath, region); !ok {
		return false, fmt.Errorf("failed to write cropped image to: %s", outputPath)
	}

	return true, nil
}

// min helper function if not using Go 1.21+ math.Min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
