package mistral

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	BaseURL = "https://api.mistral.ai/v1"
	// Maximum file size allowed by Mistral API (52.4 MB)
	MaxFileSize = 52 * 1024 * 1024
)

// Client represents a Mistral API client
type Client struct {
	APIKey string
	client *resty.Client
}

// NewClient creates a new Mistral API client
func NewClient(apiKey string) *Client {
	if apiKey == "" {
		apiKey = os.Getenv("MISTRAL_API_KEY")
		if apiKey == "" {
			return nil
		}
	}

	return &Client{
		APIKey: apiKey,
		client: resty.New().
			SetBaseURL(BaseURL).
			SetTimeout(120 * time.Second), // Add a 2-minute timeout for OCR operations
	}
}

// GetFileURL returns the signed URL for an uploaded file
func (c *Client) GetFileURL(fileID string) (string, error) {
	// Request a signed URL with 24 hour expiry
	resp, err := c.client.R().
		SetHeader("Authorization", "Bearer "+c.APIKey).
		SetHeader("Accept", "application/json").
		Get(fmt.Sprintf("/files/%s/url?expiry=24", fileID))

	if err != nil {
		return "", fmt.Errorf("error fetching file URL: %v", err)
	}

	if resp.StatusCode() != 200 {
		return "", fmt.Errorf("API returned error status: %d - %s", resp.StatusCode(), resp.String())
	}

	// Parse the response to get the signed URL
	var urlResponse struct {
		URL string `json:"url"`
	}

	if err := json.Unmarshal(resp.Body(), &urlResponse); err != nil {
		return "", fmt.Errorf("error parsing URL response: %v", err)
	}

	if urlResponse.URL == "" {
		return "", fmt.Errorf("API response did not contain a URL")
	}

	return urlResponse.URL, nil
}

// UploadFile uploads a file to Mistral API for OCR processing
func (c *Client) UploadFile(filePath string) (string, error) {
	// Check file size before uploading
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return "", fmt.Errorf("error checking file size: %v", err)
	}

	if fileInfo.Size() > MaxFileSize {
		return "", fmt.Errorf("file is too large (%.2f MB). Maximum allowed size is %.2f MB",
			float64(fileInfo.Size())/1024/1024, float64(MaxFileSize)/1024/1024)
	}

	// Add retry logic
	maxRetries := 3
	retryDelay := 3 * time.Second

	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		resp, err := c.client.R().
			SetHeader("Authorization", "Bearer "+c.APIKey).
			SetFile("file", filePath).
			SetFormData(map[string]string{
				"purpose": "ocr",
			}).
			Post("/files")

		if err != nil {
			lastErr = fmt.Errorf("error making upload request: %v", err)
			time.Sleep(retryDelay)
			continue
		}

		if resp.StatusCode() != 200 {
			errMsg := resp.String()
			lastErr = fmt.Errorf("API returned error status: %d - %s", resp.StatusCode(), errMsg)

			// Check if we should retry based on status code
			if resp.StatusCode() >= 500 || resp.StatusCode() == 429 {
				time.Sleep(retryDelay)
				continue
			}

			return "", lastErr
		}

		// Check for empty response
		if len(resp.Body()) == 0 {
			lastErr = fmt.Errorf("received empty response from API")
			time.Sleep(retryDelay)
			continue
		}

		// Parse the response to get the file ID
		var fileResponse struct {
			ID string `json:"id"`
		}

		if err := json.Unmarshal(resp.Body(), &fileResponse); err != nil {
			lastErr = fmt.Errorf("error parsing response: %v", err)
			time.Sleep(retryDelay)
			continue
		}

		if fileResponse.ID == "" {
			lastErr = fmt.Errorf("received response without file ID")
			time.Sleep(retryDelay)
			continue
		}

		// Success
		return fileResponse.ID, nil
	}

	return "", fmt.Errorf("failed to upload file after %d attempts: %v", maxRetries, lastErr)
}

// ProcessOCR processes a document with OCR
func (c *Client) ProcessOCR(docType, docSource string, includeImageBase64 bool) ([]byte, error) {
	// Create document map based on the document type
	documentMap := map[string]interface{}{
		"type": docType,
	}

	// Add the appropriate field based on document type
	switch docType {
	case "document_url":
		documentMap["document_url"] = docSource
	case "image_url":
		documentMap["image_url"] = docSource
	default:
		return nil, fmt.Errorf("unsupported document type: %s", docType)
	}

	requestBody := map[string]interface{}{
		"model":                "mistral-ocr-latest",
		"document":             documentMap,
		"include_image_base64": includeImageBase64,
	}

	// Add retry logic for empty responses
	maxRetries := 5
	retryDelay := 10 * time.Second

	var lastErr error
	var resp *resty.Response

	for attempt := 1; attempt <= maxRetries; attempt++ {
		resp, lastErr = c.client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Authorization", "Bearer "+c.APIKey).
			SetHeader("Accept", "application/json").
			SetBody(requestBody).
			Post("/ocr")

		// Check for API error status codes
		if lastErr != nil {
			time.Sleep(retryDelay)
			continue
		}

		// Check for non-200 status codes
		if resp.StatusCode() != 200 {
			var errMsg string
			if len(resp.Body()) > 0 {
				errMsg = string(resp.Body())
			} else {
				errMsg = resp.Status()
			}

			// Check for specific error codes that might indicate we should retry
			if resp.StatusCode() >= 500 || resp.StatusCode() == 429 {
				lastErr = fmt.Errorf("API returned error status: %d - %s", resp.StatusCode(), errMsg)
				time.Sleep(retryDelay)
				continue
			}

			// For other errors, don't retry
			return nil, fmt.Errorf("API returned error status: %d - %s", resp.StatusCode(), errMsg)
		}

		// Check for empty response
		if len(resp.Body()) == 0 {
			lastErr = fmt.Errorf("received empty response from API")

			// For empty responses, try with a longer delay
			adjustedDelay := retryDelay * time.Duration(attempt)
			time.Sleep(adjustedDelay)
			continue
		}

		// Check if response appears to be valid JSON
		if !json.Valid(resp.Body()) {
			lastErr = fmt.Errorf("received invalid JSON response from API")
			time.Sleep(retryDelay)
			continue
		}

		// If we got here, we have a valid response
		return resp.Body(), nil
	}

	// If we've exhausted all retries, provide a detailed error
	return nil, fmt.Errorf("failed after %d attempts. Last error: %v", maxRetries, lastErr)
}
