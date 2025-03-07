# Mistral OCR CLI

A command-line tool for processing documents with Mistral AI's OCR capabilities.

## Features

- Process PDF documents and images using Mistral AI's OCR
- Extract text and structured content from documents
- Process local files or files from URLs
- Output results to stdout or to a file
- Convert OCR results to Markdown format
- Maintain document structure and formatting in the output

## Installation

### Requirements

- Go 1.18 or later

### Building from source

```bash
git clone https://github.com/setkyar/llm-tools
cd llm-tools/mistral-ocr
go build -o mistral-ocr
```

## Usage

### Setting up your API key

You can provide your Mistral API key in two ways:

1. Environment variable:
```bash
export MISTRAL_API_KEY=your-api-key
```

2. Command line flag:
```bash
mistral-ocr --api-key=your-api-key [command]
```

### Commands

#### Process a document

Process a document file or URL:

```bash
# Process a local PDF file
mistral-ocr process path/to/document.pdf

# Process a document from a URL
mistral-ocr process https://example.com/document.pdf

# Process an image from a URL
mistral-ocr process https://example.com/image.jpg

# Save output to a file
mistral-ocr process path/to/document.pdf --output-file results.json

# Include base64 encoded images in the output
mistral-ocr process path/to/document.pdf --include-images
```

#### Convert OCR JSON to Markdown

Convert previously processed OCR JSON results to Markdown:

```bash
# Convert OCR JSON to Markdown
mistral-ocr convert results.json

# Specify output directory
mistral-ocr convert results.json --output-dir output_folder

# Create a single markdown file instead of one per page
mistral-ocr convert results.json --single-file

# Specify output filename for single file mode
mistral-ocr convert results.json --output-file document.md

# Include images in markdown (if available in JSON)
mistral-ocr convert results.json --images
```

#### Process and Convert in One Step

Process a document and convert to Markdown in a single command:

```bash
# Process document and generate markdown files
mistral-ocr markdown path/to/document.pdf

# Generate a single markdown file instead of separate files per page
mistral-ocr markdown path/to/document.pdf --single-file

# Specify output directory for markdown files
mistral-ocr markdown https://example.com/document.pdf --output-dir docs

# Specify a specific output file path (implies single file)
mistral-ocr markdown path/to/document.pdf --output-file docs/result.md

# Save intermediate JSON and generate markdown files
mistral-ocr markdown path/to/document.pdf --json-file results.json --output-dir docs
```

This command combines the `process` and `convert` steps, creating markdown files directly from the document.

#### Version information

```bash
mistral-ocr version
```

### Examples

### Process a local PDF and save the output

```bash
mistral-ocr process ~/Documents/sample.pdf --output-file results.json
```

### Process a document from a URL

```bash
mistral-ocr process https://arxiv.org/pdf/2201.04234 > output.json
```

### Convert OCR JSON to Markdown files

```bash
# Create separate files (one per page)
mistral-ocr convert output.json --output-dir markdown_docs

# Create a single file with all pages
mistral-ocr convert output.json --single-file --output-dir markdown_docs

# Create a single file with a specific filename
mistral-ocr convert output.json --output-file docs/paper.md
```

### Process a document and generate markdown files in one step

```bash
# Generate separate files (one per page)
mistral-ocr markdown ~/Documents/research-paper.pdf --output-dir research_docs

# Generate a single markdown file
mistral-ocr markdown ~/Documents/research-paper.pdf --single-file --output-dir research_docs

# Generate a single markdown file with specific filename
mistral-ocr markdown ~/Documents/research-paper.pdf --output-file research_docs/paper.md
```

## License

MIT