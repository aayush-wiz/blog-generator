package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

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
