package services

import (
	"encoding/json"
	"log/slog"

	ws "github.com/krshsl/praxis/backend/websocket"
)

type WebSocketHandler struct {
	aiMessageProcessor *AIMessageProcessor
}

func NewWebSocketHandler(aiMessageProcessor *AIMessageProcessor) *WebSocketHandler {
	return &WebSocketHandler{
		aiMessageProcessor: aiMessageProcessor,
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
		if h.aiMessageProcessor != nil {
			h.aiMessageProcessor.ProcessAudioMessage(client, msg.AudioData)
		} else {
			slog.Warn("AI message processor not available", "session_id", client.SessionID)
		}
	default:
		slog.Warn("Unknown message type", "type", msg.Type, "session_id", client.SessionID)
	}
}
