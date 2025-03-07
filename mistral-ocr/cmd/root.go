package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// API key flag
	apiKey string

	// Root command
	RootCmd = &cobra.Command{
		Use:   "mistral-ocr",
		Short: "OCR tool using Mistral AI",
		Long: `A CLI tool for performing OCR on documents using Mistral AI.
It can process PDF documents and extract text maintaining document structure.`,
	}
)

func init() {
	// Initialize API key from environment variable if not provided as a flag
	RootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "Mistral API key (defaults to MISTRAL_API_KEY env variable)")

	// Add commands
	RootCmd.AddCommand(processCmd)
	RootCmd.AddCommand(convertCmd)
	RootCmd.AddCommand(processMarkdownCmd)
	RootCmd.AddCommand(versionCmd)
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// getAPIKey returns the API key from flag or environment variable
func getAPIKey() string {
	if apiKey != "" {
		return apiKey
	}

	apiKey = os.Getenv("MISTRAL_API_KEY")
	if apiKey == "" {
		fmt.Println("Error: MISTRAL_API_KEY environment variable is not set and no --api-key flag was provided")
		os.Exit(1)
	}

	return apiKey
}
