# MyDown - Document Generation Utility

## Description

MyDown is a command-line utility written in Go that automates the process of creating Microsoft Word (.docx) documents from a collection of images. It performs the following main operations:

1.  **Downloads Images**: Fetches images from a list of URLs provided in a text file.
2.  **Conditional Cropping**: Optionally crops these downloaded images based on template images. If a corresponding template for an image is found in the specified template directory, the image is cropped using template matching (powered by GoCV and OpenCV). If no template is found, the original downloaded image is used.
3.  **Word Document Generation**: Arranges the processed (cropped or original) images into .docx files according to layouts defined in a JSON configuration file. This includes support for various layouts like full-page images, vertical splits, and custom ratio distributions.

## Prerequisites

*   **Go**: Version 1.18 or newer is recommended. (The project uses Go 1.22.2).
*   **OpenCV**: A system-level installation of OpenCV (version 3.x or 4.x) is required for the image cropping functionality, as it relies on GoCV (`gocv.io/x/gocv`).
    *   Installation guides for OpenCV can be found at [https://opencv.org/get-started/](https://opencv.org/get-started/). Ensure that the development headers and libraries are installed (e.g., `libopencv-dev` on Debian/Ubuntu).
*   **C Compiler**: A C compiler (like GCC) is needed for GoCV to build.
*   **pkg-config**: This tool is needed for GoCV to find OpenCV.

**Important Licensing Note:**
The `pkg/wordgenerator` component of this project uses the `baliance.com/gooxml` library for creating .docx files. This library is licensed under the **AGPL-3.0 license**. Please be aware of its terms if you intend to use or distribute this software.

## Building

1.  **Ensure Prerequisites are Met**: Install Go and OpenCV (including development files) on your system.
2.  **Clone the Repository** (if you haven't already):
    ```bash
    git clone <repository-url>
    cd mydown # Navigate to the project's root directory
    ```
3.  **Build the Executable**:
    From the project root directory, run:
    ```bash
    go build -o mydown_app ./cmd/mydown
    ```
    This will create an executable named `mydown_app` (or `mydown_app.exe` on Windows) in the project root.

    **Note**: The build process *must* be performed on a machine where OpenCV is correctly installed and discoverable by GoCV (usually via `pkg-config`). If OpenCV is not detected, the build will fail, typically with errors related to GoCV or Cgo.

## Running

Execute the compiled application from your terminal, providing the necessary flags.

**CLI Flags:**

*   `-download-list <filepath>`: (Required) Path to a text file containing image URLs, one URL per line.
*   `-template-dir <dirpath>`: Path to the directory containing template images for cropping. (Default: `templates/`)
*   `-config <filepath>`: Path to the JSON layout configuration file for Word document generation. (Default: `config.json`)
*   `-work-dir <dirpath>`: Path to a working directory for storing intermediate files (downloaded images, cropped images). (Default: `work_temp/`)
*   `-output-dir <dirpath>`: Path to the final output directory where generated .docx files will be saved. (Default: `output_docx/`)

**Example Command:**

```bash
./mydown_app -download-list urls.txt -template-dir ./image_templates -config layout_config.json -output-dir ./generated_documents -work-dir ./temp_processing
```

## Project Structure

*   `cmd/mydown/`: Contains the main application entry point (`main.go`) for the CLI.
*   `pkg/downloader/`: Go package responsible for downloading images from URLs.
*   `pkg/cropper/`: Go package responsible for conditional image cropping using GoCV and template matching.
*   `pkg/wordgenerator/`: Go package responsible for generating .docx files from images based on JSON layout configurations, using the `baliance.com/gooxml` library.
*   `templates/` (example): Directory to store template images for cropping. Create this if using the default.
*   `config.json` (example): JSON file defining how images should be laid out in the generated Word documents. Create this if using the default.

## Configuration Files

### 1. Download List File (`-download-list`)

A plain text file where each line contains a single URL pointing to an image to be downloaded.

Example (`urls.txt`):
```
http://example.com/image1.jpg
https://example.org/another_image.png
http://server.net/path/to/image.tif
```

The basename of the URL (e.g., `image1.jpg`, `another_image.png`, `image.tif` after removing any query parameters) will be used as the filename for downloaded and processed images. These filenames should correspond to the image names used in the layout configuration JSON.

### 2. Template Directory (`-template-dir`)

This directory should contain template images used for the cropping process. The naming convention for templates is crucial:

*   For a downloaded image named `filename.ext` (e.g., `image1.jpg`), the cropper will search for templates in this directory in the following order:
    1.  `filename_template.png`
    2.  `filename_template.jpg`
    3.  `filename_template.jpeg`
*   The first template found will be used for cropping. If no matching template is found, the original downloaded image will be used for document generation.

Example: If `image1.jpg` is downloaded, you might have `image1_template.png` in your template directory.

### 3. Layout Config JSON (`-config`)

This JSON file defines how one or more .docx files should be generated, which images they should include, and what layout to use. The keys in the JSON object are the desired output .docx filenames.

*   `files`: An array of image filenames (these should match the basenames of the downloaded URLs, after processing/cropping, found in the `work-dir/cropped/` directory).
*   `layout`: A string specifying the layout type. Supported types include:
    *   `"full_page"`: A single image fills the page (respecting margins).
    *   `"vertical_2_split"`: Two images arranged vertically, each taking part of the page.
    *   `"vertical_3_split"`: Three images arranged vertically.
    *   `"custom_ratio"`: Images arranged based on specified ratios (currently supports `orientation: "vertical"` and `ratio_orientation: "vertical"`).
*   `rotate` (optional): An integer (e.g., `90`, `180`, `270`) to rotate all images in the `files` array, or an array of integers/nulls corresponding to each file (e.g., `[90, null, 180]`). `null` or omitted means no rotation for that specific image. Rotation is clockwise.
*   `ratios` (optional, required for `custom_ratio`): An array of floating-point numbers (e.g., `[0.7, 0.3]`) that sum to 1.0, defining the proportion of page space each image should take in a custom ratio layout.
*   `orientation` (optional, for `custom_ratio`): e.g., `"vertical"`.
*   `ratio_orientation` (optional, for `custom_ratio`): e.g., `"vertical"`.

**Example (`config.json`):**
```json
{
  "MyDocument_Report1.docx": {
    "files": ["image1.jpg", "image2.png"],
    "layout": "vertical_2_split",
    "rotate": [90, 0]
  },
  "Appendix_Images.docx": {
    "files": ["figure_a.tif"],
    "layout": "full_page",
    "rotate": 180
  },
  "CustomLayoutDemo.docx": {
    "files": ["photo_scene.jpg", "detail_shot.png"],
    "layout": "custom_ratio",
    "orientation": "vertical",
    "ratio_orientation": "vertical",
    "ratios": [0.65, 0.35]
  }
}
```
