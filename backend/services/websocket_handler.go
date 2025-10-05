package services

import (
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"time"

	ws "github.com/krshsl/praxis/backend/websocket"
)

// safeSend tries to send a message to the client channel, recovers if closed
func safeSend(ch chan<- []byte, msg []byte) {
	defer func() {
		if r := recover(); r != nil {
			// Channel is closed, ignore
		}
	}()
	select {
	case ch <- msg:
		// sent
	default:
		// channel full or closed
	}
}

type WebSocketHandler struct {
	aiMessageProcessor *AIMessageProcessor
	timeoutService     *SessionTimeoutService
}

func NewWebSocketHandler(aiMessageProcessor *AIMessageProcessor, timeoutService *SessionTimeoutService) *WebSocketHandler {
	return &WebSocketHandler{
		aiMessageProcessor: aiMessageProcessor,
		timeoutService:     timeoutService,
	}
}

// HandleWebSocketConnection handles the initial WebSocket connection and auto-starts the interview
func (h *WebSocketHandler) HandleWebSocketConnection(client *ws.Client) {
	slog.Info("WebSocket connection handled", "user_id", client.UserID, "session_id", client.SessionID)

	// Auto-start the interview
	if h.aiMessageProcessor != nil {
		h.aiMessageProcessor.AutoStartInterview(client)
	} else {
		slog.Warn("AI message processor not available for auto-start", "session_id", client.SessionID)
	}
}

// HandleWebSocketMessage processes incoming WebSocket messages and routes them to AI processing
func (h *WebSocketHandler) HandleWebSocketMessage(client *ws.Client, messageBytes []byte) {
	var msg ws.Message
	if err := json.Unmarshal(messageBytes, &msg); err != nil {
		slog.Error("Failed to unmarshal WebSocket message", "error", err)
		return
	}

	slog.Info("WebSocket message received", "type", msg.Type, "user_id", client.UserID, "session_id", client.SessionID)

	// Route message to appropriate AI processor
	switch msg.Type {
	case "text":
		if h.aiMessageProcessor != nil {
			h.aiMessageProcessor.ProcessTextMessage(client, msg.Content)
		} else {
			slog.Warn("AI message processor not available", "session_id", client.SessionID)
		}
	case "code":
		if h.aiMessageProcessor != nil {
			h.aiMessageProcessor.ProcessCodeMessage(client, msg.Content, msg.Language)
		} else {
			slog.Warn("AI message processor not available", "session_id", client.SessionID)
		}
	case "audio":
		// Handle both binary and Base64 audio data
		var audioData []byte
		if len(msg.AudioData) > 0 {
			// Binary audio data
			audioData = msg.AudioData
		} else if msg.AudioDataBase64 != "" {
			// Base64 audio data from frontend
			decoded, err := base64.StdEncoding.DecodeString(msg.AudioDataBase64)
			if err != nil {
				slog.Error("Failed to decode Base64 audio data", "error", err, "session_id", client.SessionID)
				return
			}
			audioData = decoded
		} else {
			slog.Error("No audio data provided", "session_id", client.SessionID)
			return
		}

		slog.Info("Audio message routed", "session_id", client.SessionID, "audio_size", len(audioData))
		if h.aiMessageProcessor != nil {
			h.aiMessageProcessor.ProcessAudioMessage(client, audioData)
		} else {
			slog.Warn("AI message processor not available", "session_id", client.SessionID)
		}
	case "audio_chunk":
		// Handle chunked audio data
		var audioData []byte
		if len(msg.AudioData) > 0 {
			// Binary audio data
			audioData = msg.AudioData
		} else if msg.AudioDataBase64 != "" {
			// Base64 audio data from frontend
			decoded, err := base64.StdEncoding.DecodeString(msg.AudioDataBase64)
			if err != nil {
				slog.Error("Failed to decode Base64 audio chunk data", "error", err, "session_id", client.SessionID)
				return
			}
			audioData = decoded
		} else {
			slog.Error("No audio chunk data provided", "session_id", client.SessionID)
			return
		}

		slog.Info("Audio chunk routed", "session_id", client.SessionID, "chunk_index", msg.ChunkIndex, "total_chunks", msg.TotalChunks)
		if h.aiMessageProcessor != nil {
			h.aiMessageProcessor.ProcessAudioChunk(client, audioData, msg.ChunkIndex, msg.TotalChunks, msg.IsLastChunk)
		} else {
			slog.Warn("AI message processor not available", "session_id", client.SessionID)
		}
	case "end_session":
		// End the session politely and generate summary
		slog.Info("Received end_session request", "session_id", client.SessionID)
		// Send confirmation message to client
		endMsg := map[string]any{
			"type":    "end_session",
			"content": "Thank you for your time. We'll wrap up the session and prepare your summary.",
		}
		if b, err := json.Marshal(endMsg); err == nil {
			safeSend(client.Send, b)
		}
		if h.timeoutService != nil {
			h.timeoutService.ConcludeSession(client.SessionID, "User ended interview")
		}
		// Close the WebSocket connection after a short delay to allow the message to be sent
		go func() {
			// Wait 200ms to ensure message is sent
			// (tune as needed for your infra)
			<-time.After(200 * time.Millisecond)
			client.Conn.Close()
		}()
	default:
		slog.Warn("Unknown message type", "type", msg.Type, "session_id", client.SessionID)
	}
}
