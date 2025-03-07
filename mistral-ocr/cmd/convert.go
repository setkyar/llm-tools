package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

var (
	markdownDir       string
	markdownFile      string
	includeImages     bool
	includePageBreaks bool
	titleFromFilename bool
	singleFile        bool

	convertCmd = &cobra.Command{
		Use:   "convert [json_file]",
		Short: "Convert OCR JSON output to Markdown",
		Long: `Convert OCR JSON output from Mistral AI to Markdown format.
The tool will extract text and structure from the JSON output and create Markdown files.`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			jsonFile := args[0]
			convertJSONToMarkdown(jsonFile)
		},
	}
)

func init() {
	convertCmd.Flags().StringVarP(&markdownDir, "output-dir", "d", "markdown_output", "Directory to store markdown files")
	convertCmd.Flags().StringVarP(&markdownFile, "output-file", "o", "", "Output filename for single file mode (default: document.md)")
	convertCmd.Flags().BoolVar(&includeImages, "images", false, "Include images in markdown (if available)")
	convertCmd.Flags().BoolVar(&includePageBreaks, "page-breaks", true, "Include page break indicators between pages")
	convertCmd.Flags().BoolVar(&titleFromFilename, "title-from-filename", true, "Use filename as document title")
	convertCmd.Flags().BoolVar(&singleFile, "single-file", false, "Create a single markdown file instead of one per page")

	// If output file is specified, enable single file mode
	convertCmd.PreRun = func(cmd *cobra.Command, args []string) {
		if markdownFile != "" {
			singleFile = true
		}
	}
}

// OCRResponse represents the structure of Mistral OCR API response
type OCRResponse struct {
	Pages []struct {
		Index    int    `json:"index"`
		Markdown string `json:"markdown"`
		Image    string `json:"image,omitempty"`
		Images   []struct {
			ID           string `json:"id"`
			TopLeftX     int    `json:"top_left_x"`
			TopLeftY     int    `json:"top_left_y"`
			BottomRightX int    `json:"bottom_right_x"`
			BottomRightY int    `json:"bottom_right_y"`
			ImageBase64  string `json:"image_base64"`
		} `json:"images,omitempty"`
		Dimensions struct {
			DPI    int `json:"dpi"`
			Height int `json:"height"`
			Width  int `json:"width"`
		} `json:"dimensions,omitempty"`
	} `json:"pages"`
	Metadata struct {
		Title        string `json:"title,omitempty"`
		Author       string `json:"author,omitempty"`
		CreationDate string `json:"creation_date,omitempty"`
		PageCount    int    `json:"page_count,omitempty"`
	} `json:"metadata,omitempty"`
}

// replaceImageReferences replaces image references in markdown content with base64 data
// Format: ![img-id.ext](img-id.ext) becomes ![img-id.ext](data:image/jpeg;base64,DATA)
func replaceImageReferences(content string, images []OCRResponse_Image) string {
	if !includeImages || len(images) == 0 {
		return content
	}

	// Create a map of image IDs to their base64 data
	imageMap := make(map[string]string)
	for _, img := range images {
		if img.ImageBase64 != "" {
			imgData := img.ImageBase64
			if !strings.HasPrefix(imgData, "data:") {
				imgData = "data:image/jpeg;base64," + imgData
			}
			imageMap[img.ID] = imgData
		}
	}

	// Replace all image references with base64 data
	for id, base64Data := range imageMap {
		// Escape special characters in the ID for regex
		escapedID := regexp.QuoteMeta(id)
		pattern := fmt.Sprintf(`!\[%s\]\(%s\)`, escapedID, escapedID)
		replacement := fmt.Sprintf(`![%s](%s)`, id, base64Data)

		re := regexp.MustCompile(pattern)
		content = re.ReplaceAllString(content, replacement)
	}

	return content
}

// OCRResponse_Image is a helper type for the replaceImageReferences function
type OCRResponse_Image struct {
	ID          string
	ImageBase64 string
}

func convertJSONToMarkdown(jsonFile string) {
	// Read JSON file
	data, err := os.ReadFile(jsonFile)
	if err != nil {
		fmt.Printf("Error reading JSON file: %v\n", err)
		os.Exit(1)
	}

	// Parse JSON
	var ocrResponse OCRResponse
	if err := json.Unmarshal(data, &ocrResponse); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)

		// Try parsing as raw map to debug structure
		var rawJSON map[string]interface{}
		if jsonErr := json.Unmarshal(data, &rawJSON); jsonErr == nil {
			fmt.Println("JSON top-level keys:")
			for k := range rawJSON {
				fmt.Printf("- %s\n", k)
			}
		}

		os.Exit(1)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(markdownDir, 0755); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	if singleFile {
		// Process all pages into a single markdown file
		var combined strings.Builder
		title := "Document"

		// Use metadata title if available
		if ocrResponse.Metadata.Title != "" {
			title = ocrResponse.Metadata.Title
		} else if titleFromFilename {
			// Use filename without extension
			base := filepath.Base(jsonFile)
			title = strings.TrimSuffix(base, filepath.Ext(base))
		}

		combined.WriteString(fmt.Sprintf("# %s\n\n", title))

		// Add metadata if available
		if ocrResponse.Metadata.Author != "" || ocrResponse.Metadata.CreationDate != "" {
			combined.WriteString("## Document Metadata\n\n")
			if ocrResponse.Metadata.Author != "" {
				combined.WriteString(fmt.Sprintf("**Author:** %s\n\n", ocrResponse.Metadata.Author))
			}
			if ocrResponse.Metadata.CreationDate != "" {
				combined.WriteString(fmt.Sprintf("**Creation Date:** %s\n\n", ocrResponse.Metadata.CreationDate))
			}
			if ocrResponse.Metadata.PageCount > 0 {
				combined.WriteString(fmt.Sprintf("**Page Count:** %d\n\n", ocrResponse.Metadata.PageCount))
			}
		}

		// Process each page
		for i, page := range ocrResponse.Pages {
			// Add page header
			combined.WriteString(fmt.Sprintf("## Page %d\n\n", page.Index+1))

			// Convert page images to OCRResponse_Image format
			var pageImages []OCRResponse_Image
			for _, img := range page.Images {
				pageImages = append(pageImages, OCRResponse_Image{
					ID:          img.ID,
					ImageBase64: img.ImageBase64,
				})
			}

			// Replace image references in markdown content if includeImages is true
			pageContent := page.Markdown
			if includeImages {
				pageContent = replaceImageReferences(pageContent, pageImages)
			}

			// Add page content
			combined.WriteString(pageContent)
			combined.WriteString("\n\n")

			// Add page separator if not the last page
			if includePageBreaks && i < len(ocrResponse.Pages)-1 {
				combined.WriteString("\n\n---\n\n")
			}
		}

		// Write combined markdown file
		// Use custom filename if provided, otherwise use default
		filename := "document.md"
		if markdownFile != "" {
			// If markdownFile contains directory components, ensure they exist
			dir := filepath.Dir(markdownFile)
			if dir != "." {
				if err := os.MkdirAll(filepath.Join(markdownDir, dir), 0755); err != nil {
					fmt.Printf("Error creating output subdirectory: %v\n", err)
					os.Exit(1)
				}
			}
			filename = markdownFile
		}
		outputFilePath := filepath.Join(markdownDir, filename)

		if err := os.WriteFile(outputFilePath, []byte(combined.String()), 0644); err != nil {
			fmt.Printf("Error writing markdown file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Created single markdown file: %s\n", outputFilePath)
	} else {
		// Process each page into a separate file
		for _, page := range ocrResponse.Pages {
			// Use page index as the filename
			filename := fmt.Sprintf("%d.md", page.Index)
			outputFilePath := filepath.Join(markdownDir, filename)

			// Convert page images to OCRResponse_Image format
			var pageImages []OCRResponse_Image
			for _, img := range page.Images {
				pageImages = append(pageImages, OCRResponse_Image{
					ID:          img.ID,
					ImageBase64: img.ImageBase64,
				})
			}

			// Get page content with image references replaced if needed
			markdownContent := page.Markdown
			if includeImages {
				markdownContent = replaceImageReferences(markdownContent, pageImages)
			}

			if err := os.WriteFile(outputFilePath, []byte(markdownContent), 0644); err != nil {
				fmt.Printf("Error writing markdown file %s: %v\n", outputFilePath, err)
				os.Exit(1)
			}

			fmt.Printf("Created markdown file: %s\n", outputFilePath)
		}
	}

	fmt.Printf("Successfully converted %s to markdown files in %s/\n", jsonFile, markdownDir)
	fmt.Printf("Total pages: %d\n", len(ocrResponse.Pages))
}
