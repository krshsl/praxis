package websocket

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	mu         sync.RWMutex
}

type Client struct {
	Hub                 *Hub
	Conn                *websocket.Conn
	Send                chan []byte
	UserID              string
	SessionID           string
	ConversationHistory []string
	MessageHandler      func(*Client, []byte) // Function to handle incoming messages
	mu                  sync.RWMutex
}

type Message struct {
	Type            string `json:"type"` // "text", "code", "audio", "audio_chunk", "user_message"
	Content         string `json:"content"`
	Language        string `json:"language,omitempty"`
	AudioData       []byte `json:"audio_data,omitempty"`
	AudioDataBase64 string `json:"audio_data_base64,omitempty"` // For Base64 encoded audio from frontend
	ChunkIndex      int    `json:"chunk_index,omitempty"`       // For audio chunks
	TotalChunks     int    `json:"total_chunks,omitempty"`      // For audio chunks
	IsLastChunk     bool   `json:"is_last_chunk,omitempty"`     // For audio chunks
	SessionID       string `json:"session_id,omitempty"`
}

type AudioMessage struct {
	Type      string `json:"type"` // "audio"
	AudioData []byte `json:"audio_data"`
	SessionID string `json:"session_id,omitempty"`
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			slog.Info("Client registered", "user_id", client.UserID, "session_id", client.SessionID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mu.Unlock()
			slog.Info("Client unregistered", "user_id", client.UserID, "session_id", client.SessionID)

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) RegisterClient(conn *websocket.Conn, userID string) *Client {
	sessionID := uuid.New().String()
	client := &Client{
		Hub:                 h,
		Conn:                conn,
		Send:                make(chan []byte, 256),
		UserID:              userID,
		SessionID:           sessionID,
		ConversationHistory: []string{},
		MessageHandler:      nil, // Will be set by the main.go handler
	}

	h.register <- client
	return client
}

func (c *Client) ReadPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(10 * 1024 * 1024) // 10MB limit for large audio recordings
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, messageBytes, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Error("WebSocket error", "error", err)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			slog.Error("Failed to unmarshal message", "error", err)
			continue
		}

		slog.Info("Message received", "type", msg.Type, "session_id", c.SessionID, "content_length", len(msg.Content))

		// Use message handler if available, otherwise fall back to default handling
		if c.MessageHandler != nil {
			// Run message handler asynchronously to avoid blocking
			go c.MessageHandler(c, messageBytes)
		} else {
			// Fallback to default message handling
			switch msg.Type {
			case "text":
				c.handleTextMessage(msg)
			case "code":
				c.handleCodeMessage(msg)
			default:
				slog.Warn("Unknown message type", "type", msg.Type)
			}
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) handleTextMessage(msg Message) {
	// Add to conversation history
	c.mu.Lock()
	c.ConversationHistory = append(c.ConversationHistory, msg.Content)
	c.mu.Unlock()

	// Trigger AI conversation processing
	// This will be handled by the AI message processor
	slog.Info("Text message received for AI processing", "content", msg.Content, "user_id", c.UserID, "session_id", c.SessionID)
}

func (c *Client) handleCodeMessage(msg Message) {
	// Add to conversation history
	c.mu.Lock()
	c.ConversationHistory = append(c.ConversationHistory, fmt.Sprintf("Code submission in %s: %s", msg.Language, msg.Content))
	c.mu.Unlock()

	// Trigger code analysis processing
	slog.Info("Code message received for AI analysis", "language", msg.Language, "user_id", c.UserID)
}

func (c *Client) SendAudio(audioData []byte) {
	audioMsg := AudioMessage{
		Type:      "audio",
		AudioData: audioData,
		SessionID: c.SessionID,
	}

	audioBytes, err := json.Marshal(audioMsg)
	if err != nil {
		slog.Error("Failed to marshal audio message", "error", err)
		return
	}

	c.Send <- audioBytes
}

func (c *Client) GetConversationHistory() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ConversationHistory
}
