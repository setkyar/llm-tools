package main

import (
	"fmt"
	"os"

	"github.com/setkyar/llm-tools/mistral-ocr/cmd"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
