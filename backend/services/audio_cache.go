package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

// AudioCache provides filesystem-based caching for ElevenLabs audio
type AudioCache struct {
	cacheDir string
	mutex    sync.RWMutex
}

// Common phrases that should be cached
var CommonPhrases = map[string]bool{
	"Hello! Welcome to your interview. Let's get started.":          true,
	"Great answer! Let's move on to the next question.":             true,
	"Thank you for that response. Here's your next question.":       true,
	"Excellent! Now let's discuss something different.":             true,
	"That's a thoughtful answer. Let's continue.":                   true,
	"Perfect! Moving forward to the next topic.":                    true,
	"Thank you for your time today. The interview is now complete.": true,
	"Well done! That concludes our interview session.":              true,
	"Take a moment to think about your response.":                   true,
	"Please take your time to consider this question.":              true,
	"Think carefully about this and share your thoughts.":           true,
	"Let me give you a moment to think about that.":                 true,
	"I'd like to hear your thoughts on this topic.":                 true,
	"What's your perspective on this matter?":                       true,
	"How would you approach this situation?":                        true,
	"Can you tell me more about your experience with this?":         true,
	"That's an interesting point. Can you elaborate?":               true,
	"I see. Could you provide a specific example?":                  true,
	"Thank you for sharing that. What else would you add?":          true,
	"Let's dive deeper into this topic.":                            true,
	"I'd like to explore this further with you.":                    true,
	"That brings up an important consideration.":                    true,
	"You've made some good points there.":                           true,
	"I appreciate your detailed response.":                          true,
}

// NewAudioCache creates a new audio cache with the specified directory
func NewAudioCache(cacheDir string) *AudioCache {
	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		slog.Error("Failed to create cache directory", "dir", cacheDir, "error", err)
	}

	return &AudioCache{
		cacheDir: cacheDir,
	}
}

// generateCacheKey creates a unique key for caching based on text and voice ID
func (ac *AudioCache) generateCacheKey(text, voiceID string) string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%s", text, voiceID)))
	return hex.EncodeToString(hash[:])
}

// getCachePath returns the full path for a cache file
func (ac *AudioCache) getCachePath(key string) string {
	return filepath.Join(ac.cacheDir, key+".mp3")
}

// IsCommonPhrase checks if the given text is a common phrase that should be cached
func (ac *AudioCache) IsCommonPhrase(text string) bool {
	return CommonPhrases[text]
}

// Get retrieves cached audio data if it exists
func (ac *AudioCache) Get(ctx context.Context, text, voiceID string) ([]byte, bool) {
	if !ac.IsCommonPhrase(text) {
		return nil, false
	}

	ac.mutex.RLock()
	defer ac.mutex.RUnlock()

	key := ac.generateCacheKey(text, voiceID)
	cachePath := ac.getCachePath(key)

	data, err := os.ReadFile(cachePath)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Error("Failed to read cached audio", "path", cachePath, "error", err)
		}
		return nil, false
	}

	slog.Info("Cache hit for common phrase", "text", text, "voice_id", voiceID)
	return data, true
}

// Set stores audio data in the cache
func (ac *AudioCache) Set(ctx context.Context, text, voiceID string, audioData []byte) error {
	if !ac.IsCommonPhrase(text) {
		return nil // Don't cache non-common phrases
	}

	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	key := ac.generateCacheKey(text, voiceID)
	cachePath := ac.getCachePath(key)

	err := os.WriteFile(cachePath, audioData, 0644)
	if err != nil {
		slog.Error("Failed to write audio to cache", "path", cachePath, "error", err)
		return err
	}

	slog.Info("Cached common phrase audio", "text", text, "voice_id", voiceID, "size", len(audioData))
	return nil
}

// GetOrGenerate gets cached audio or generates new audio and caches it
func (ac *AudioCache) GetOrGenerate(ctx context.Context, text, voiceID string, generator func() (io.ReadCloser, error)) ([]byte, error) {
	// Try to get from cache first
	if cachedData, found := ac.Get(ctx, text, voiceID); found {
		return cachedData, nil
	}

	// Generate new audio
	audioReader, err := generator()
	if err != nil {
		return nil, fmt.Errorf("failed to generate audio: %w", err)
	}
	defer audioReader.Close()

	// Read all data
	audioData, err := io.ReadAll(audioReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio data: %w", err)
	}

	// Cache if it's a common phrase
	if ac.IsCommonPhrase(text) {
		if err := ac.Set(ctx, text, voiceID, audioData); err != nil {
			slog.Warn("Failed to cache audio", "error", err)
		}
	}

	return audioData, nil
}

// ClearCache removes all cached files
func (ac *AudioCache) ClearCache() error {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	return os.RemoveAll(ac.cacheDir)
}

// GetCacheStats returns basic cache statistics
func (ac *AudioCache) GetCacheStats() (int, int64, error) {
	ac.mutex.RLock()
	defer ac.mutex.RUnlock()

	entries, err := os.ReadDir(ac.cacheDir)
	if err != nil {
		return 0, 0, err
	}

	var totalSize int64
	fileCount := 0

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".mp3" {
			fileCount++
			if info, err := entry.Info(); err == nil {
				totalSize += info.Size()
			}
		}
	}

	return fileCount, totalSize, nil
}
