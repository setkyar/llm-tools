package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/setkyar/llm-tools/mistral-ocr/pkg/mistral"
	"github.com/spf13/cobra"
)

var (
	jsonOutputFile     string
	includeImageBase64 bool

	processCmd = &cobra.Command{
		Use:   "process [file]",
		Short: "Process a document with OCR",
		Long: `Process a document file (PDF, image) using Mistral AI's OCR capabilities.
The file can be a local file or a URL.`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			filePath := args[0]

			// Determine if input is a URL or a local file
			if strings.HasPrefix(filePath, "http://") || strings.HasPrefix(filePath, "https://") {
				processURL(filePath)
			} else {
				// add debug logging of filePath
				processLocalFile(filePath)
			}
		},
	}
)

func init() {
	processCmd.Flags().StringVarP(&jsonOutputFile, "output-file", "o", "", "Output JSON file path (default is stdout)")
	processCmd.Flags().BoolVar(&includeImageBase64, "include-images", false, "Include base64 encoded images in the output")
}

func processURL(url string) {
	// Create Mistral client
	client := mistral.NewClient(getAPIKey())
	if client == nil {
		fmt.Println("Error: MISTRAL_API_KEY environment variable is not set and no --api-key flag was provided")
		os.Exit(1)
	}

	// Determine the document type based on URL
	docType := "document_url"
	if strings.HasSuffix(strings.ToLower(url), ".jpg") ||
		strings.HasSuffix(strings.ToLower(url), ".jpeg") ||
		strings.HasSuffix(strings.ToLower(url), ".png") ||
		strings.HasSuffix(strings.ToLower(url), ".webp") ||
		strings.HasSuffix(strings.ToLower(url), ".gif") {
		docType = "image_url"
	}

	// Process the document
	respData, err := client.ProcessOCR(docType, url, includeImageBase64)
	if err != nil {
		fmt.Printf("Error processing document: %v\n", err)
		os.Exit(1)
	}

	// Handle the output
	handleOutput(respData)
}

func processLocalFile(filePath string) {
	fmt.Printf("Processing local file: %s\n", filePath)
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("Error: file '%s' does not exist\n", filePath)
		os.Exit(1)
	}

	// Create Mistral client
	client := mistral.NewClient(getAPIKey())
	if client == nil {
		fmt.Println("Error: MISTRAL_API_KEY environment variable is not set and no --api-key flag was provided")
		os.Exit(1)
	}

	// Upload the file to Mistral API
	fileID, err := client.UploadFile(filePath)
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
	lowerFilePath := strings.ToLower(filePath)
	if strings.HasSuffix(lowerFilePath, ".jpg") ||
		strings.HasSuffix(lowerFilePath, ".jpeg") ||
		strings.HasSuffix(lowerFilePath, ".png") ||
		strings.HasSuffix(lowerFilePath, ".webp") ||
		strings.HasSuffix(lowerFilePath, ".gif") {
		docType = "image_url"
	}

	fmt.Printf("Processing with signed file URL (type: %s)\n", docType)

	// Process the uploaded file with the appropriate type
	respData, err := client.ProcessOCR(docType, fileURL, includeImageBase64)
	if err != nil {
		fmt.Printf("Error processing document: %v\n", err)
		os.Exit(1)
	}

	// Handle the output
	handleOutput(respData)
}

func handleOutput(data []byte) {
	// Pretty print the JSON response
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, data, "", "  "); err != nil {
		fmt.Printf("Error formatting JSON: %v\n", err)
		os.Exit(1)
	}

	// Write to output file or stdout
	if jsonOutputFile != "" {
		// Create directory if it doesn't exist
		dir := filepath.Dir(jsonOutputFile)
		if dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				fmt.Printf("Error creating output directory: %v\n", err)
				os.Exit(1)
			}
		}

		// Write the file
		if err := os.WriteFile(jsonOutputFile, prettyJSON.Bytes(), 0644); err != nil {
			fmt.Printf("Error writing output file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("OCR results saved to %s\n", jsonOutputFile)
	} else {
		// Write to stdout
		fmt.Println(prettyJSON.String())
	}
}
