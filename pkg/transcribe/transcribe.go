package transcribe

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/mousybusiness/mouse_whisperer/pkg/mic"
	log "github.com/sirupsen/logrus"
)

var ExampleCommands = []string{
	"cut that",
	"paste that",
	"enter that",
	"move up 2",
	"down 4",
	"go to start",
	"scroll up",
	"select right word",
	"center mouse",
	"prose apples",
	"type a,b,c",
	"click 0xfa (hex)",
	"move 0x240 (hex)",
	"insert semi-colon",
	"format",
	"line 123",
}

var CommandPrompt = fmt.Sprintf("Computer commands for text editing, code creation, navigation, e.g.: %v. selected: ", strings.Join(ExampleCommands, ","))

// ProsePrompt biases whisper toward natural prose for dictation finals
// and replays, unlike the command-vocabulary prompt used for ticks
const ProsePrompt = "A dictated sentence or paragraph of natural English prose."

type Response struct {
	Transcription string `json:"text"`
}

type Result struct {
	Session   mic.RecordingSession
	text      string
	isCommand bool
}

func (r Result) Filename() string {
	return r.Session.FileName
}

func (r Result) Text() string {
	return r.text
}

func (r Result) Copy(newTranscription string) Result {
	r.text = newTranscription
	return r
}

func Do(session mic.RecordingSession, model mic.ModelType) (*Result, error) {
	isCommand := session.RecordingRequest == mic.RecordingRequest{}
	prompt := session.RecordingRequest.Prompt()
	if prompt == "" {
		prompt = CommandPrompt
	}

	result, err := sendToServer(defaultClient, session.FileName, prompt, model)
	if err != nil {
		if err.Error() != "noise" {
			log.Errorf("Error sending request: %v", err)
		}
		return nil, err
	}

	return &Result{
		text:      result.Transcription,
		isCommand: isCommand,
		Session:   session,
	}, nil
}

// Whisper server endpoints; vars so tests can point at a stub server.
var (
	FastURL     = "http://127.0.0.1:51010/inference"
	AccurateURL = "http://127.0.0.1:51011/inference"
)

// defaultClient preserves the historical no-timeout behavior for batch
// transcription (the accurate model can be slow on long input).
var defaultClient = &http.Client{}

// streamClient bounds snapshot transcriptions: the streaming worker is the
// serialization point, and a wedged call would silently stop all partials.
var streamClient = &http.Client{Timeout: 6 * time.Second}

// File transcribes an arbitrary WAV file (used for streaming snapshots).
// Returns the "noise" error for empty/bracket-only results, same as Do.
// The multipart request carries the file path, not its bytes — the whisper
// server reads the local file, so it must exist until this returns.
func File(filename, prompt string, model mic.ModelType) (string, error) {
	result, err := sendToServer(streamClient, filename, prompt, model)
	if err != nil {
		return "", err
	}
	return result.Transcription, nil
}

// longClient bounds re-reads of joined dictation recordings, which can run
// to minutes of audio: far past streamClient's budget, but still bounded
// because the caller blocks the dictation worker.
var longClient = &http.Client{Timeout: 60 * time.Second}

// FileLong is File for long recordings (joined dictation replays).
func FileLong(filename, prompt string, model mic.ModelType) (string, error) {
	result, err := sendToServer(longClient, filename, prompt, model)
	if err != nil {
		return "", err
	}
	return result.Transcription, nil
}

func sendToServer(client *http.Client, filename string, prompt string, model mic.ModelType) (*Response, error) {
	start := time.Now()
	defer func() {
		log.Debugf("Transcription took %v\n", time.Since(start))
	}()

	log.Debugf("File: %v\n", filename)
	// Prepare the file
	file, err := os.Open(filename)
	if err != nil {
		log.Error("Error opening file:", err)
		return nil, err
	}
	defer file.Close()

	// Create a new multipart writer
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add other form fields
	_ = writer.WriteField("file", file.Name())
	_ = writer.WriteField("prompt", prompt)
	_ = writer.WriteField("temperature", "0.0")
	_ = writer.WriteField("temperature_inc", "0.2")
	_ = writer.WriteField("response_format", "json")

	// Close the multipart writer
	err = writer.Close()
	if err != nil {
		log.Error("Error closing writer:", err)
		return nil, err
	}

	// Create the HTTP request - select URL based on model type
	url := FastURL
	if model == mic.ModelAccurate {
		url = AccurateURL
	}
	log.Debugf("Sending to %s model at %s", model, url)
	req, err := http.NewRequest("POST", url, body) // This works with premade wav
	if err != nil {
		log.Error("Error creating request:", err)
		return nil, err
	}

	// Set the Content-Type header
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Make the HTTP request
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read body, err: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Server returned non-OK status: %d, body: %v", resp.StatusCode, string(b))
	}

	var result Response
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal response, err: %v", err)
	}

	result.Transcription = strings.TrimSpace(result.Transcription)

	re := regexp.MustCompile("\\[.*\\]")
	s := strings.TrimSpace(re.ReplaceAllString(result.Transcription, ""))
	if s == "" {
		log.Debugf("NOISE Transcription: %v", result.Transcription)
		return nil, errors.New("noise")
	}

	result.Transcription = s

	log.Debugf("Transcription: %v", s)

	return &result, nil
}
