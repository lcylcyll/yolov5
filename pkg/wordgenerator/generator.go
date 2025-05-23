package wordgenerator

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color" // For background color if needed with imaging
	_ "image/gif" // Register GIF decoder
	_ "image/jpeg" // Register JPEG decoder
	_ "image/png"  // Register PNG decoder
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"baliance.com/gooxml/color" as docxColor
	"baliance.com/gooxml/document"
	"baliance.com/gooxml/measurement"
	"baliance.com/gooxml/schema/soo/wml"
	"github.com/disintegration/imaging"
	// For TIFF, a more specific library is needed. Standard library doesn't support it well.
	// Using golang.org/x/image/tiff for decoding.
	_ "golang.org/x/image/tiff"
)

const (
	// A4 paper dimensions in inches
	a4WidthInches  = 8.27
	a4HeightInches = 11.69
	// Default margin in inches (Word's default is often 1 inch)
	defaultMarginInches = 1.0
)

// GenerateWordDocuments processes the JSON configuration and generates DOCX files.
func GenerateWordDocuments(configFilePath string, imageBaseDir string, outputDir string) error {
	// Read and parse the JSON configuration
	configFile, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", configFilePath, err)
	}

	var config DocxConfig
	if err := json.Unmarshal(configFile, &config); err != nil {
		return fmt.Errorf("failed to unmarshal config JSON: %w", err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}

	for docxName, layoutDetail := range config {
		doc := document.New()

		// Process images and apply layout
		err := processAndLayoutImages(doc, &layoutDetail, imageBaseDir, outputDir)
		if err != nil {
			fmt.Printf("Error processing document %s: %v. Skipping.\n", docxName, err)
			// Decide if one error should stop all, or just skip the current doc.
			// For now, skipping.
			continue
		}

		// Save the document
		outputPath := filepath.Join(outputDir, docxName)
		if err := doc.SaveToFile(outputPath); err != nil {
			fmt.Printf("Error saving document %s: %v. Skipping.\n", outputPath, err)
			// Consider collecting errors and returning them at the end.
		}
	}
	return nil // Or return a collection of errors if any occurred
}

func processAndLayoutImages(doc *document.Document, layoutDetail *FileLayoutDetail, imageBaseDir string, outputDir string) error {
	if len(layoutDetail.Files) == 0 {
		return fmt.Errorf("no files specified for layout type %s", layoutDetail.Layout)
	}

	// Available width and height on page (A4 with 1-inch margins)
	// gooxml uses TWIPS (1/1440th of an inch) by default for many things, but measurement.Distance helps.
	pageWidth := measurement.Distance(a4WidthInches * measurement.Inch)
	pageHeight := measurement.Distance(a4HeightInches * measurement.Inch)
	margin := measurement.Distance(defaultMarginInches * measurement.Inch)

	// Usable width and height after margins
	usableWidth := pageWidth - (2 * margin)
	usableHeight := pageHeight - (2 * margin)
	
	// Set default page margins (optional, library might have defaults)
    // Note: Baliance gooxml sets page margins on a section property.
    // Ensure the document has a section property.
    bodySectPr := doc.BodySection()
    if bodySectPr == nil {
        // This is unlikely as New() should create one.
        // If sections are added manually, ensure they get margins.
        paraForSect := doc.AddParagraph() // A paragraph is needed to attach section properties
        bodySectPr = paraForSect.Properties().AddSection(wml.ST_SectionMarkNextPage) // or other type
    }
    bodySectPr.SetPageMargins(margin, margin, margin, margin, margin, margin, 0)


	var images []image.Image
	var imageRefs []document.ImageRef
	var originalFilePaths []string

	for i, fileName := range layoutDetail.Files {
		fullPath := filepath.Join(imageBaseDir, fileName)
		originalFilePaths = append(originalFilePaths, fullPath)

		imgData, err := os.Open(fullPath)
		if err != nil {
			return fmt.Errorf("failed to open image file %s: %w", fullPath, err)
		}
		defer imgData.Close()

		img, _, err := image.Decode(imgData) // Using image.Decode to handle various formats
		if err != nil {
			// Attempt with imaging.Open for potentially broader format support or better error messages
			img, err = imaging.Open(fullPath)
			if err != nil {
				return fmt.Errorf("failed to decode image %s: %w", fullPath, err)
			}
		}
		
		// Handle Rotation
		rotationAngle := layoutDetail.getRotationForIndex(i)
		if rotationAngle != nil {
			switch *rotationAngle {
			case 90:
				img = imaging.Rotate90(img)
			case 180:
				img = imaging.Rotate180(img)
			case 270:
				img = imaging.Rotate270(img)
			case -90: // Assuming -90 is 270 clockwise
				img = imaging.Rotate270(img)
			case -180:
				img = imaging.Rotate180(img)
			case -270: // Assuming -270 is 90 clockwise
				img = imaging.Rotate90(img)
			}
		}
		images = append(images, img)

		// Add image to document package (this step uses the original image data or re-encoded)
		// For simplicity, let's re-encode the potentially rotated image to PNG for docx insertion.
		// This avoids issues with library support for direct image.Image objects if they are modified.
		
		// Create a unique temp name for rotated image if needed for AddImageFromFile
		// Or, use AddImage which takes an image.Image
		// Baliance gooxml's common.Image has FromGoImage. Let's try that.
		docxImg, err := document.ImageFromGoImage(img)
		if err != nil {
			return fmt.Errorf("failed to convert Go image to docx image for %s: %w", fileName, err)
		}

		imgRef, err := doc.AddImage(docxImg)
		if err != nil {
			return fmt.Errorf("failed to add image %s to document: %w", fileName, err)
		}
		imageRefs = append(imageRefs, imgRef)
	}

	switch layoutDetail.Layout {
	case "full_page":
		if len(imageRefs) == 0 || len(images) == 0 { return fmt.Errorf("no image for full_page layout") }
		addFullPageImage(doc.AddParagraph(), imageRefs[0], images[0], usableWidth, usableHeight)
	case "vertical_2_split":
		if len(imageRefs) < 2 || len(images) < 2 { return fmt.Errorf("not enough images for vertical_2_split, got %d", len(imageRefs)) }
		addVerticalSplitLayout(doc, imageRefs[:2], images[:2], 2, usableWidth, usableHeight)
	case "vertical_3_split":
		if len(imageRefs) < 3 || len(images) < 3 { return fmt.Errorf("not enough images for vertical_3_split, got %d", len(imageRefs)) }
		addVerticalSplitLayout(doc, imageRefs[:3], images[:3], 3, usableWidth, usableHeight)
	case "custom_ratio":
		if layoutDetail.Orientation == "vertical" && layoutDetail.RatioOrientation == "vertical" {
			if len(imageRefs) < len(layoutDetail.Ratios) || len(images) < len(layoutDetail.Ratios) {
				return fmt.Errorf("not enough images for custom_ratio, expected %d, got %d", len(layoutDetail.Ratios), len(imageRefs))
			}
			addCustomRatioVerticalLayout(doc, imageRefs, images, layoutDetail.Ratios, usableWidth, usableHeight)
		} else {
			return fmt.Errorf("unsupported custom_ratio orientation: %s/%s", layoutDetail.Orientation, layoutDetail.RatioOrientation)
		}
	default:
		return fmt.Errorf("unknown layout type: %s", layoutDetail.Layout)
	}

	return nil
}

func addFullPageImage(p *document.Paragraph, imgRef document.ImageRef, goImg image.Image, usableWidth, usableHeight measurement.Distance) {
	// Calculate aspect ratio
	bounds := goImg.Bounds()
	aspectRatio := float64(bounds.Dx()) / float64(bounds.Dy())

	// Calculate image dimensions to fit within usable page dimensions while maintaining aspect ratio
	imgWidth := usableWidth
	imgHeight := measurement.Distance(float64(usableWidth) / aspectRatio)

	if imgHeight > usableHeight {
		imgHeight = usableHeight
		imgWidth = measurement.Distance(float64(usableHeight) * aspectRatio)
	}
	
	run := p.AddRun()
	inlineImg, err := run.AddDrawingInline(imgRef)
	if err != nil {
		fmt.Printf("Error adding inline drawing for full page: %v\n", err)
		return
	}
	inlineImg.SetSize(imgWidth, imgHeight)
	p.Properties().SetAlignment(wml.ST_JcCenter) // Center the paragraph/image
}

func addVerticalSplitLayout(doc *document.Document, imgRefs []document.ImageRef, goImgs []image.Image, numSplits int, usableWidth, usableHeight measurement.Distance) {
	if len(imgRefs) != numSplits || len(goImgs) != numSplits {
		fmt.Printf("Warning: Mismatch between expected splits (%d) and provided images (%d)\n", numSplits, len(imgRefs))
		return
	}
	
	table := doc.AddTable()
	table.Properties().SetWidthPercent(100 * 100 / float64(usableWidth.Twips()) * float64(usableWidth.Twips()) / 100) // 100% of page width effectively
    table.Properties().SetLayout(wml.ST_TblLayoutTypeFixed) // Fixed layout

	cellWidth := usableWidth // Each cell takes full width

	for i := 0; i < numSplits; i++ {
		row := table.AddRow()
		cell := row.AddCell()
		cell.Properties().SetWidth(cellWidth) // Set cell width

		// Calculate image height for this split, maintaining aspect ratio
		img := goImgs[i]
		bounds := img.Bounds()
		aspectRatio := float64(bounds.Dx()) / float64(bounds.Dy())

		// Each image gets 1/numSplits of the usable height
		// This is a simplification; actual cell height needs to be set if possible, or image scaled to fit.
		// Baliance gooxml table row height setting is `row.Properties().SetHeight(h, rule)`.
		// For now, scale image to fit proportional height and full usable width.

		var scaledWidth, scaledHeight measurement.Distance
		targetCellHeight := measurement.Distance(float64(usableHeight) / float64(numSplits))

		// Scale to fit width first
		scaledWidth = cellWidth
		scaledHeight = measurement.Distance(float64(scaledWidth) / aspectRatio)

		// If scaled height is too much for its allocated slot, rescale based on height
		if scaledHeight > targetCellHeight {
			scaledHeight = targetCellHeight
			scaledWidth = measurement.Distance(float64(scaledHeight) * aspectRatio)
		}
		
		// Ensure cell content is vertically centered if possible (or top)
        cell.Properties().SetVerticalAlignment(wml.ST_VerticalJcCenter)


		p := cell.AddParagraph()
		p.Properties().SetAlignment(wml.ST_JcCenter) // Center image in cell
		run := p.AddRun()
		inlineImg, err := run.AddDrawingInline(imgRefs[i])
		if err != nil {
			fmt.Printf("Error adding inline drawing for split layout: %v\n", err)
			continue
		}
		inlineImg.SetSize(scaledWidth, scaledHeight)
	}
}

func addCustomRatioVerticalLayout(doc *document.Document, imgRefs []document.ImageRef, goImgs []image.Image, ratios []float64, usableWidth, usableHeight measurement.Distance) {
	if len(imgRefs) != len(ratios) || len(goImgs) != len(ratios) {
		fmt.Printf("Warning: Mismatch between ratios count (%d) and images count (%d)\n", len(ratios), len(imgRefs))
		return
	}

	table := doc.AddTable()
	table.Properties().SetWidthPercent(100) // 100% of page width effectively
    table.Properties().SetLayout(wml.ST_TblLayoutTypeFixed)

	cellWidth := usableWidth

	for i, ratio := range ratios {
		row := table.AddRow()
		cell := row.AddCell()
		cell.Properties().SetWidth(cellWidth)
        cell.Properties().SetVerticalAlignment(wml.ST_VerticalJcCenter)


		// Calculate target height for this image based on ratio
		targetImageHeight := measurement.Distance(float64(usableHeight) * ratio)
		
		// Set row height (this is an approximation, actual rendering depends on Word)
        // The ST_HeightRuleExact or ST_HeightRuleAtLeast might be useful here.
        // Using ST_HeightRuleAtLeast to ensure content fits.
        row.Properties().SetHeight(targetImageHeight, wml.ST_HeightRuleAtLeast)


		img := goImgs[i]
		bounds := img.Bounds()
		aspectRatio := float64(bounds.Dx()) / float64(bounds.Dy())

		// Scale image to fit targetImageHeight while maintaining aspect ratio
		scaledHeight := targetImageHeight
		scaledWidth := measurement.Distance(float64(scaledHeight) * aspectRatio)

		// If scaled width exceeds usable width, rescale based on width
		if scaledWidth > usableWidth {
			scaledWidth = usableWidth
			scaledHeight = measurement.Distance(float64(scaledWidth) / aspectRatio)
		}
		
		p := cell.AddParagraph()
		p.Properties().SetAlignment(wml.ST_JcCenter)
		run := p.AddRun()
		inlineImg, err := run.AddDrawingInline(imgRefs[i])
		if err != nil {
			fmt.Printf("Error adding inline drawing for custom ratio layout: %v\n", err)
			continue
		}
		inlineImg.SetSize(scaledWidth, scaledHeight)
	}
}

// Helper to convert measurement.Distance to a string with unit for debugging
func formatDistance(d measurement.Distance, unit measurement.Distance) string {
	return fmt.Sprintf("%.2f inches", d.Inches())
}
