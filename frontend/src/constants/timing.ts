// Interview timing constants
export const INTERVIEW_TIMING = {
  // Time for user to think before speaking (in milliseconds)
  THINK_TIME: 10 * 1000, // 10 seconds
  
  // Maximum time for user to speak (in milliseconds)
  SPEAK_TIME: 30 * 1000, // 30 seconds
  
  // Buffer time before starting recording (in milliseconds)
  RECORDING_BUFFER: 1 * 1000, // 1 second
} as const

// Audio recording settings
export const AUDIO_SETTINGS = {
  // Audio chunk size for processing (in bytes)
  CHUNK_SIZE: 2 * 1024 * 1024, // 2MB
  
  // Maximum audio file size (in bytes)
  MAX_AUDIO_SIZE: 5 * 1024 * 1024, // 5MB
  
  // Audio format
  AUDIO_FORMAT: 'audio/webm',
} as const
