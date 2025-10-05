package services

import (
	"crypto/sha1"
	"encoding/binary"
	"strings"
)

// List of stock ElevenLabs voice IDs for each gender
var femaleVoices = []string{
	"EXAVITQu4vr4xnSDxMaL", // Rachel
	"21m00Tcm4TlvDq8ikWAM", // Domi
	"AZnzlk1XvdvUeBnXmlld", // Bella
	"ErXwobaYiN019PkySvjV", // Elli
	"MF3mGyEYCl7XYWbV9V6O", // Dorothy
}

var maleVoices = []string{
	"pNInz6obpgDQGcFmaJgB", // Adam
	"TxGEqnHWrfWFTfGW9XjX", // Antoni
	"VR6AewLTigWG4xSOukaG", // Josh
	"yoZ06aMxZJJ28mfd3POQ", // Arnold
	"bVMeCyTHy58xNoL34h3p", // Clyde
}

// PickDeterministicVoice returns a stock ElevenLabs voice ID based on name and gender
func PickDeterministicVoice(name, gender string) string {
	var pool []string
	switch strings.ToLower(gender) {
	case "female":
		pool = femaleVoices
	case "male":
		pool = maleVoices
	default:
		pool = append(femaleVoices, maleVoices...)
	}
	if len(pool) == 0 {
		return "pNInz6obpgDQGcFmaJgB" // fallback Adam
	}
	// Hash the name to pick a voice
	h := sha1.New()
	h.Write([]byte(strings.ToLower(name)))
	sum := h.Sum(nil)
	idx := binary.BigEndian.Uint16(sum) % uint16(len(pool))
	return pool[idx]
}
