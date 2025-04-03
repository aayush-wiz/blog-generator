package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

// LlamaIndexRequest represents the data sent to the Python LlamaIndex script
type LlamaIndexRequest struct {
	Topic    string           `json:"topic"`
	Contents []ScrapedContent `json:"contents"`
}

// LlamaIndexResponse represents the response from the Python LlamaIndex script
type LlamaIndexResponse struct {
	Title         string        `json:"title"`
	Content       []BlogContent `json:"content"`
	FeaturedImage string        `json:"featuredImage"`
	Tags          []string      `json:"tags"`
	Summary       string        `json:"summary"`
}

// GenerateBlogWithLlamaIndex calls the Python script that implements LlamaIndex to generate a blog
func GenerateBlogWithLlamaIndex(topic string, contents []ScrapedContent) (LlamaIndexResponse, error) {
	var response LlamaIndexResponse

	// Create the request
	request := LlamaIndexRequest{
		Topic:    topic,
		Contents: contents,
	}

	// Convert request to JSON
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return response, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Prepare the Python script command
	cmd := exec.Command("python3", "llamaindex_service.py")

	// Set up stdin/stdout pipes
	cmd.Stdin = bytes.NewBuffer(requestJSON)
	var out bytes.Buffer
	cmd.Stdout = &out
	var errOut bytes.Buffer
	cmd.Stderr = &errOut

	// Run the command
	err = cmd.Run()
	if err != nil {
		return response, fmt.Errorf("failed to run Python script: %v\nStderr: %s", err, errOut.String())
	}

	// Parse the response
	err = json.Unmarshal(out.Bytes(), &response)
	if err != nil {
		return response, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return response, nil
}
