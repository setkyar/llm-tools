package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/setkyar/llm-tools/mistral-ocr/pkg/mistral"
	"github.com/spf13/cobra"
)

var (
	processMarkdownCmd = &cobra.Command{
		Use:   "markdown [file_or_url]",
		Short: "Process document and convert to markdown in one step",
		Long: `Process a document with OCR and convert the output directly to markdown.
This combines the 'process' and 'convert' commands in a single operation.`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fileOrURL := args[0]
			processAndConvertToMarkdown(fileOrURL)
		},
	}

	// Output file variables (moved into function init)
)

func init() {
	// Add flags from both process and convert commands
	processMarkdownCmd.Flags().StringVarP(&jsonOutputFile, "json-file", "j", "", "Save intermediate JSON to file (optional)")

	// Markdown conversion flags
	processMarkdownCmd.Flags().StringVarP(&markdownDir, "output-dir", "d", "markdown_output", "Directory to store markdown files")
	processMarkdownCmd.Flags().StringVarP(&markdownFile, "output-file", "o", "", "Path for output markdown file (implies --single-file)")
	processMarkdownCmd.Flags().BoolVar(&includeImages, "images", false, "Include extracted images in markdown (if available)")
	processMarkdownCmd.Flags().BoolVar(&includePageBreaks, "page-breaks", true, "Include page break indicators between pages")
	processMarkdownCmd.Flags().BoolVar(&titleFromFilename, "title-from-filename", true, "Use filename as document title")
	processMarkdownCmd.Flags().BoolVar(&singleFile, "single-file", false, "Create a single markdown file instead of one per page")

	// Ensure that if --images is true, includeImageBase64 is also true
	processMarkdownCmd.PreRun = func(cmd *cobra.Command, args []string) {
		if includeImages {
			includeImageBase64 = true
		}

		// If output file is specified, enable single file mode
		if markdownFile != "" {
			singleFile = true
		}
	}
}

func processAndConvertToMarkdown(fileOrURL string) {
	// Create temporary file for JSON output if not specified
	var jsonOutputPath string
	if jsonOutputFile == "" {
		tmpFile, err := os.CreateTemp("", "mistral-ocr-*.json")
		if err != nil {
			fmt.Printf("Error creating temporary file: %v\n", err)
			os.Exit(1)
		}
		defer os.Remove(tmpFile.Name()) // Clean up temporary file when done
		tmpFile.Close()
		jsonOutputPath = tmpFile.Name()
	} else {
		jsonOutputPath = jsonOutputFile
	}

	// Step 1: Process the document - reuse logic from processURL and processLocalFile
	var respData []byte
	var err error

	// Create Mistral client
	client := mistral.NewClient(getAPIKey())
	if client == nil {
		fmt.Println("Error: MISTRAL_API_KEY environment variable is not set and no --api-key flag was provided")
		os.Exit(1)
	}

	// Determine if input is URL or local file
	if strings.HasPrefix(fileOrURL, "http://") || strings.HasPrefix(fileOrURL, "https://") {
		// Process URL
		docType := "document_url"
		if strings.HasSuffix(strings.ToLower(fileOrURL), ".jpg") ||
			strings.HasSuffix(strings.ToLower(fileOrURL), ".jpeg") ||
			strings.HasSuffix(strings.ToLower(fileOrURL), ".png") ||
			strings.HasSuffix(strings.ToLower(fileOrURL), ".webp") ||
			strings.HasSuffix(strings.ToLower(fileOrURL), ".gif") {
			docType = "image_url"
		}

		fmt.Printf("Processing URL: %s\n", fileOrURL)
		respData, err = client.ProcessOCR(docType, fileOrURL, includeImageBase64)
	} else {
		// Process local file
		if _, err := os.Stat(fileOrURL); os.IsNotExist(err) {
			fmt.Printf("Error: file '%s' does not exist\n", fileOrURL)
			os.Exit(1)
		}

		fmt.Printf("Processing local file: %s\n", fileOrURL)
		fileID, err := client.UploadFile(fileOrURL)
		if err != nil {
			fmt.Printf("Error uploading file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("File uploaded successfully with ID: %s\n", fileID)

		// Get the signed file URL for processing
		fileURL, err := client.GetFileURL(fileID)
		if err != nil {
			fmt.Printf("Error getting signed file URL: %v\n", err)
			os.Exit(1)
		}

		// Determine the document type based on file extension
		docType := "document_url"
		lowerFilePath := strings.ToLower(fileOrURL)
		if strings.HasSuffix(lowerFilePath, ".jpg") ||
			strings.HasSuffix(lowerFilePath, ".jpeg") ||
			strings.HasSuffix(lowerFilePath, ".png") ||
			strings.HasSuffix(lowerFilePath, ".webp") ||
			strings.HasSuffix(lowerFilePath, ".gif") {
			docType = "image_url"
		}

		fmt.Printf("Processing with signed file URL (type: %s)\n", docType)
		fmt.Printf("File URL: %s\n", fileURL)
		fmt.Printf("Include Image Base64: %v\n", includeImageBase64)
		respData, err = client.ProcessOCR(docType, fileURL, includeImageBase64)

		if err != nil {
			fmt.Printf("Error processing document: %v\n", err)
			os.Exit(1)
		}
	}

	if err != nil {
		fmt.Printf("Error processing document: %v\n", err)
		os.Exit(1)
	}

	// Check if we received a valid response
	if len(respData) == 0 {
		fmt.Println("Error: Received empty response from Mistral API after all retries")

		// Create a fallback file with PDF information
		if err := os.MkdirAll(markdownDir, 0755); err != nil {
			fmt.Printf("Error creating output directory: %v\n", err)
			os.Exit(1)
		}

		os.Exit(1)
	}

	// Save the JSON response
	if err := os.WriteFile(jsonOutputPath, respData, 0644); err != nil {
		fmt.Printf("Error writing JSON file: %v\n", err)
		os.Exit(1)
	}

	if jsonOutputFile != "" {
		fmt.Printf("OCR results saved to %s\n", jsonOutputPath)
	}

	// Step 2: Convert the JSON to markdown
	fmt.Println("Converting JSON to Markdown...")

	// If we're using a custom output file path, set it for convertJSONToMarkdown
	// (already handled by PreRun function, which sets singleFile to true if markdownFile is set)

	// Convert JSON to markdown
	convertJSONToMarkdown(jsonOutputPath)
}
