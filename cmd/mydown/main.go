package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/lcylcyll/mydown/pkg/cropper"
	"github.com/lcylcyll/mydown/pkg/downloader"
	"github.com/lcylcyll/mydown/pkg/wordgenerator"
)

func main() {
	// Define flags
	downloadListFile := flag.String("download-list", "", "Path to a text file with image URLs (one per line). (Required)")
	templateDir := flag.String("template-dir", "templates/", "Path to template images directory.")
	workDir := flag.String("work-dir", "work_temp/", "Path to working directory for intermediate files.")
	configFile := flag.String("config", "config.json", "Path to the JSON layout configuration file for Word documents.")
	outputDir := flag.String("output-dir", "output_docx/", "Final output directory for .docx files.")

	// Customize usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "Downloads images, optionally crops them, and generates Word documents.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s -download-list=urls.txt -template-dir=./my_templates -config=layout.json -output-dir=./final_docs -work-dir=./processing_files\n", filepath.Base(os.Args[0]))
	}

	// Parse flags
	flag.Parse()

	// --- Validate Inputs ---
	if *downloadListFile == "" {
		fmt.Fprintln(os.Stderr, "Error: -download-list flag is required.\n")
		flag.Usage()
		os.Exit(1)
	}

	// Check -download-list file existence
	if _, err := os.Stat(*downloadListFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Download list file not found: %s\n\n", *downloadListFile)
		flag.Usage()
		os.Exit(1)
	} else if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking download list file %s: %v\n\n", *downloadListFile, err)
		flag.Usage()
		os.Exit(1)
	}

	// Check -template-dir directory existence
	templateDirStat, err := os.Stat(*templateDir)
	if os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Template directory not found: %s\n\n", *templateDir)
		flag.Usage()
		os.Exit(1)
	} else if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking template directory %s: %v\n\n", *templateDir, err)
		flag.Usage()
		os.Exit(1)
	}
	if !templateDirStat.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: Template path %s is not a directory.\n\n", *templateDir)
		flag.Usage()
		os.Exit(1)
	}

	// Check -config file existence
	if _, err := os.Stat(*configFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Configuration file not found: %s\n\n", *configFile)
		flag.Usage()
		os.Exit(1)
	} else if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking configuration file %s: %v\n\n", *configFile, err)
		flag.Usage()
		os.Exit(1)
	}

	// Create work-dir and its subdirectories
	downloadedSubDir := filepath.Join(*workDir, "downloaded")
	croppedSubDir := filepath.Join(*workDir, "cropped")

	for _, dir := range []string{*workDir, downloadedSubDir, croppedSubDir, *outputDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Could not create directory %s: %v\n\n", dir, err)
			os.Exit(1)
		}
	}
	fmt.Printf("Working directory: %s\n", *workDir)
	fmt.Printf("Download directory: %s\n", downloadedSubDir)
	fmt.Printf("Cropped images directory: %s\n", croppedSubDir)
	fmt.Printf("Final output directory: %s\n", *outputDir)


	// Read image URLs from the download list file
	file, err := os.Open(*downloadListFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening download list file %s: %v\n", *downloadListFile, err)
		os.Exit(1)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var urls []string
	for scanner.Scan() {
		url := strings.TrimSpace(scanner.Text())
		if url != "" {
			urls = append(urls, url)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading download list file %s: %v\n", *downloadListFile, err)
		os.Exit(1)
	}

	if len(urls) == 0 {
		fmt.Println("No URLs found in the download list. Exiting.")
		os.Exit(0)
	}

	fmt.Printf("Found %d URLs to process.\n", len(urls))

	// Loop through each URL for downloading and cropping
	var imagesForWordProcessing []string // Stores basenames of images successfully processed

	for i, url := range urls {
		fmt.Printf("\nProcessing URL %d/%d: %s\n", i+1, len(urls), url)

		// Generate filename from URL basename
		baseName := filepath.Base(url)
		if baseName == "." || baseName == "/" {
			fmt.Fprintf(os.Stderr, "Error: Could not derive a valid filename from URL: %s. Skipping.\n", url)
			continue
		}
		// Sanitize basename further if necessary (e.g., remove query params if present in basename)
		if idx := strings.Index(baseName, "?"); idx != -1 {
			baseName = baseName[:idx]
		}


		downloadedImagePath := filepath.Join(downloadedSubDir, baseName)
		croppedImagePath := filepath.Join(croppedSubDir, baseName) // Cropped image has same name, in different dir

		// Download
		fmt.Printf("  Downloading to: %s\n", downloadedImagePath)
		err := downloader.DownloadImage(url, downloadedImagePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Error downloading %s: %v. Skipping this URL.\n", url, err)
			continue
		}
		fmt.Println("  Download successful.")

		// Crop
		fmt.Printf("  Attempting to crop: %s using templates from %s\n", downloadedImagePath, *templateDir)
		cropped, err := cropper.CropImageWithTemplate(downloadedImagePath, *templateDir, croppedImagePath)
		if err != nil {
			// An error here means something went wrong during the cropping attempt (e.g., image read error, template read error, GoCV error)
			// not just "template not found".
			fmt.Fprintf(os.Stderr, "  Error during cropping attempt for %s: %v. Using original downloaded image for Word.\n", baseName, err)
			// Copy original downloaded image to croppedSubDir for word generator
			if copyErr := copyFile(downloadedImagePath, croppedImagePath); copyErr != nil {
				fmt.Fprintf(os.Stderr, "    Failed to copy original image %s to %s: %v. Skipping for Word.\n", downloadedImagePath, croppedImagePath, copyErr)
				continue // Skip this image for word processing
			}
			fmt.Printf("    Copied original %s to %s for Word processing.\n", baseName, croppedSubDir)
		} else if cropped {
			fmt.Printf("  Successfully cropped and saved to: %s\n", croppedImagePath)
		} else {
			// No template found, err is nil.
			fmt.Printf("  No template found for %s. Using original downloaded image.\n", baseName)
			// Copy original downloaded image to croppedSubDir for word generator
			if copyErr := copyFile(downloadedImagePath, croppedImagePath); copyErr != nil {
				fmt.Fprintf(os.Stderr, "    Failed to copy original image %s to %s: %v. Skipping for Word.\n", downloadedImagePath, croppedImagePath, copyErr)
				continue // Skip this image for word processing
			}
			fmt.Printf("    Copied original %s to %s for Word processing.\n", baseName, croppedSubDir)
		}
		imagesForWordProcessing = append(imagesForWordProcessing, baseName) // Add basename
	}

	// Generate Word documents
	if len(imagesForWordProcessing) > 0 {
		// Although imagesForWordProcessing isn't directly used by GenerateWordDocuments,
		// its length indicates if any images were successfully processed to be available
		// in the croppedSubDir. GenerateWordDocuments will read config.json and pick files from croppedSubDir.
		fmt.Printf("\nStarting Word document generation using images from: %s\n", croppedSubDir)
		err = wordgenerator.GenerateWordDocuments(*configFile, croppedSubDir, *outputDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nError generating Word documents: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("\nSuccessfully generated Word documents.")
	} else {
		fmt.Println("\nNo images were successfully processed for Word document generation.")
	}
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)
	return err
}
