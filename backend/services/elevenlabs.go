package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type ElevenLabsService struct {
	apiKey string
	client *http.Client
}

type ElevenLabsRequest struct {
	Text          string        `json:"text"`
	ModelID       string        `json:"model_id"`
	VoiceID       string        `json:"voice_id"`
	VoiceSettings VoiceSettings `json:"voice_settings"`
}

type VoiceSettings struct {
	Stability       float64 `json:"stability"`
	SimilarityBoost float64 `json:"similarity_boost"`
}

func NewElevenLabsService(apiKey string) *ElevenLabsService {
	return &ElevenLabsService{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (e *ElevenLabsService) TextToSpeech(ctx context.Context, text string) (io.ReadCloser, error) {
	request := ElevenLabsRequest{
		Text:    text,
		ModelID: "eleven_turbo_v2",      // Fast model for real-time conversation
		VoiceID: "pNInz6obpgDQGcFmaJgB", // Default voice (Adam)
		VoiceSettings: VoiceSettings{
			Stability:       0.5,
			SimilarityBoost: 0.5,
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := "https://api.elevenlabs.io/v1/text-to-speech/pNInz6obpgDQGcFmaJgB"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("xi-api-key", e.apiKey)

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("elevenlabs API error: %d - %s", resp.StatusCode, string(body))
	}

	slog.Info("Generated audio from ElevenLabs", "text_length", len(text))
	return resp.Body, nil
}
