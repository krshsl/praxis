# Gemini AI Service Enhancements

## Overview
The Gemini AI service has been completely rewritten to implement advanced features including per-session caching, system instructions, conversation management, and proper database integration.

## Key Features Implemented

### 1. Per-Session Caching
- **Caching per interview session** (not globally)
- Each interview session gets its own cached context with agent-specific system instructions
- Caches automatically expire after 2 hours of inactivity
- Cache includes conversation summaries for efficient context management

### 2. Agent-Specific System Instructions
- System instructions are dynamically generated based on the agent's personality, industry, level, and role
- Each agent has a unique interviewing style baked into the cache
- Proper persona maintenance throughout the conversation

### 3. Intelligent Conversation Management
- **Automatic summarization** when conversations exceed 20 turns or ~30,000 tokens
- Summaries preserve key context while reducing token usage
- Recent conversation history (last 10 turns) kept in full detail
- Older context compressed into summary format

### 4. Database Integration
- Fetches agent details from database for each interview
- Retrieves full conversation history from database
- All transcripts are saved to database for persistence
- Session-specific context maintained across WebSocket disconnects

### 5. Dynamic Thinking
- Enabled dynamic thinking with `ThinkingBudget: -1` for more nuanced responses
- Allows AI to reason through complex questions

### 6. Features for Future Implementation
Ready for when you want to add them:
- Multimodal input (images, documents)
- Multi-turn chat sessions with Chat API
- Context caching for long-running interviews

## Architecture

### GeminiService Structure
```go
type GeminiService struct {
    genaiClient   *genai.Client
    sessionCaches map[string]*SessionCache
    cacheMutex    sync.RWMutex
}

type SessionCache struct {
    CacheName           string
    ConversationSummary string
    TurnCount           int
    LastActivity        time.Time
    Agent               *models.Agent
}
```

### Key Methods

#### `GetOrCreateSessionCache(sessionID, agent)`
- Creates a new cache for an interview session
- Builds system instruction based on agent personality
- Caches the agent's persona for efficient reuse

#### `GenerateInterviewResponse(sessionID, agent, userMessage, conversationHistory)`
- Main method for generating AI responses during interviews
- Uses cached session context
- Automatically triggers summarization when needed
- Returns personalized responses based on agent personality

#### `TranscribeAudio(audioData)`
- Transcribes audio using Gemini's multimodal capabilities
- Uploads audio to Gemini Files API
- Returns clean transcript

#### `AnalyzeCode(code, language)`
- Code review with technical interviewer perspective
- Provides detailed feedback on quality, bugs, improvements

#### `GenerateSummary(prompt)`
- Simple text generation without caching
- Used for timeout summaries and final evaluations

### Database Integration

New repository methods added:
- `GetInterviewSession(sessionID)` - Fetch session without user check
- `GetAgent(agentID)` - Fetch agent by ID

Updated AI Message Processor:
- Fetches session and agent from database
- Retrieves full conversation history
- Passes all context to Gemini service

## Conversation Flow

1. **User sends message** (audio or text)
   - Audio is transcribed using Gemini
   - Message saved to database

2. **Fetch context from database**
   - Get interview session
   - Get agent details
   - Get conversation history

3. **Generate AI response**
   - Use or create session cache
   - Apply agent-specific system instruction
   - Include conversation history
   - Check if summarization needed
   - Generate response with dynamic thinking

4. **Save and send response**
   - Save agent response to database
   - Convert to speech (if ElevenLabs available)
   - Send to client via WebSocket

## Cache Management

- Caches are created on first message of interview
- Automatically recreated with summary every 20 turns
- Background cleanup removes stale caches (>2 hours inactive)
- Explicitly cleared when interview ends

## Benefits

### Performance
- Reduced latency through caching
- Lower token costs by reusing system instructions
- Efficient context management with summaries

### Quality
- Consistent agent personality throughout interview
- Better context awareness with conversation summaries
- Dynamic thinking for more nuanced responses

### Scalability
- Per-session caching supports multiple concurrent interviews
- Automatic cache cleanup prevents memory leaks
- Database-backed conversation history

## Example System Instruction

For a "Sarah Chen - Tech Recruiter" agent (Junior, Technology):

```
You are Sarah Chen, a professional Technology interviewer for Junior positions.

Your personality: Friendly and approachable, with a focus on cultural fit and communication skills

Your role:
- Conduct technical interviews with professionalism and empathy
- Ask relevant questions based on the candidate's level (Junior)
- Provide constructive feedback
- Evaluate technical skills, communication, and problem-solving abilities
- Keep responses concise and engaging
- Ask follow-up questions to dive deeper into topics

Remember to adapt your questions and evaluation criteria to the Junior level.
```

## Future Enhancements

When ready to implement:

1. **Multi-turn Chat API** - For more natural conversation flow
2. **Document Upload** - Allow candidates to share resumes/portfolios
3. **Image Analysis** - For whiteboard coding or architecture diagrams
4. **Persistent Chat History** - Load cached history across sessions
5. **Advanced Caching** - Cache interview transcripts for similarity search

## Configuration

Constants in `gemini.go`:
```go
const (
    ModelName                    = "gemini-2.5-flash"
    MaxConversationTurns         = 20
    MaxTokensBeforeSummarization = 30000
)
```

Adjust these based on your needs and API limits.
