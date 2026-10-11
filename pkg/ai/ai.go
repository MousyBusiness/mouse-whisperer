package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type GenerateRequest struct {
	Model  string `json:"model"`
	Stream bool   `json:"stream"`
	Raw    bool   `json:"raw"`
	Prompt string `json:"prompt"`
}

type GenerateResponse struct {
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	Response  string    `json:"response"`
	Done      bool      `json:"done"`
	Error     string    `json:"error"` // ollama sends errors (e.g. model not found) as JSON
}

type ChatRequest struct {
	Model    string    `json:"model"`
	Stream   bool      `json:"stream"`
	Raw      bool      `json:"raw"`
	Messages []Message `json:"messages"`
}

type ChatResponse struct {
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	Message   Message   `json:"message"`
	Done      bool      `json:"done"`
}

// DefaultModel is the default Ollama model to use
const DefaultModel = "codellama:7b-instruct"

// client bounds generation time: callers (e.g. dictation tidy) block a
// worker while waiting, so a hung ollama must not hang them forever
var client = &http.Client{Timeout: 120 * time.Second}

// https://ollama.com/library/codellama:instruct
// https://ollama.com/blog/how-to-prompt-code-llama
func SendToOllama(model, prompt string) (string, error) {
	m := GenerateRequest{
		Model:  model,
		Stream: false,
		Raw:    false,
		Prompt: prompt,
	}

	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}

	resp, err := client.Post("http://localhost:11434/api/generate", "application/json", bytes.NewBuffer(b))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var response GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}
	if response.Error != "" {
		return "", fmt.Errorf("ollama: %s", response.Error)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama: unexpected status %d", resp.StatusCode)
	}

	return response.Response, nil
}

// func chat(prompt string) (string, error) {
// 	m := ChatRequest{
// 		Model:  "llama3.1",
// 		Stream: false,
// 		Raw:    false,
// 		Messages: []Message{
// 			{
// 				Role:    "server",
// 				Content: "RESPOND ONLY WITH REQUESTED CODE",
// 			},
// 			{
// 				Role:    "user",
// 				Content: prompt,
// 			},
// 		},
// 	}

// 	b, err := json.Marshal(m)
// 	if err != nil {
// 		log.Error(err)
// 		return "", err
// 	}

// 	resp, err := http.Post("http://localhost:11434/api/chat", "application/json", bytes.NewBuffer(b))
// 	if err != nil {
// 		return "", err
// 	}
// 	defer resp.Body.Close()

// 	var response ChatResponse
// 	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
// 		return "", err
// 	}

// 	return response.Message.Content, nil
// }
