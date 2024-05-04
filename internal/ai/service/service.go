package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"

	"github.com/bxxf/znvo-backend/internal/ai/chat"
	"github.com/bxxf/znvo-backend/internal/ai/prompt"
	"github.com/bxxf/znvo-backend/internal/logger"
)

const (
	conversationTimeout = 5 * time.Second
	endSessionFuncName  = "endSession"
)

// MessageType represents the type of message (AI or User)
type MessageType int

const (
	MessageTypeAI MessageType = iota
	MessageTypeUser
)

// AiService represents the AI service
type AiService struct {
	logger      *logger.LoggerInstance
	llm         *openai.LLM
	streamStore *StreamStore
	chatService *chat.ChatService
}

// StartConversationResponse represents the response from starting a conversation
type StartConversationResponse struct {
	Message   string
	SessionID string
}

// NewAiService creates a new instance of the AI service
func NewAiService(logger *logger.LoggerInstance, streamStore *StreamStore, chatService *chat.ChatService) *AiService {
	llm := InitializeModel()
	return &AiService{
		logger:      logger,
		streamStore: streamStore,
		chatService: chatService,
		llm:         llm,
	}
}

// InitializeModel initializes the AI model
func InitializeModel() *openai.LLM {
	llm, err := openai.New(openai.WithModel("gpt-3.5-turbo"))
	if err != nil {
		panic(err)
	}
	return llm
}

// StartConversation starts a conversation with the AI model and returns the response
func (s *AiService) StartConversation(ctx context.Context) (*StartConversationResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, conversationTimeout)
	defer cancel()

	// Define initial message history with prompt
	messageHistory := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, prompt.Prompt),
	}

	// Generate first message based on the prompt
	resp, err := s.llm.GenerateContent(ctx, messageHistory, llms.WithTools(AvailableTools))
	if err != nil {
		s.logger.Error("Failed to generate content: ", err)
		return nil, err
	}

	// Create message content struct for the response
	newContent := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeAI, resp.Choices[0].Content),
	}
	messageHistory = append(messageHistory, newContent...)

	// Generate a unique session ID
	sessionID := s.generateUniqueSessionID()

	// Save the message history to the stream store
	_, err = s.chatService.SaveMessageHistory(&messageHistory, sessionID)
	if err != nil {
		s.logger.Error("Failed to save message history: ", err)
		return nil, err
	}

	return &StartConversationResponse{
		Message:   resp.Choices[0].Content,
		SessionID: sessionID,
	}, nil
}

// generateUniqueSessionID generates a unique session ID
func (s *AiService) generateUniqueSessionID() string {
	sessionID := uuid.New().String()
	for {
		_, found := s.streamStore.GetStream(sessionID)
		if !found {
			break
		}
		s.logger.Error("Stream already exists")
		sessionID = uuid.New().String()
	}
	return sessionID
}

// SendMessage sends a message to the AI model and returns the response
func (s *AiService) SendMessage(ctx context.Context, sessionID, message string, messageType MessageType) (*StartConversationResponse, error) {
	var outputMessage string
	messageHistoryPointer, err := s.chatService.LoadMessageHistory(sessionID)
	if err != nil {
		s.logger.Error("Failed to load message history: ", err)
		return nil, err
	}

	// Create message content based on the message type
	var role llms.ChatMessageType
	if messageType == MessageTypeAI {
		role = llms.ChatMessageTypeSystem
	} else {
		role = llms.ChatMessageTypeHuman
	}

	msg := llms.MessageContent{
		Role: role,
		Parts: []llms.ContentPart{
			llms.TextPart(message),
		},
	}

	msgHistory := *messageHistoryPointer
	msgHistory = append(msgHistory, msg)

	// Generate content based on the message history
	resp, err := s.llm.GenerateContent(ctx, msgHistory, llms.WithTools(AvailableTools))
	if err != nil {
		s.logger.Error("Failed to generate content: ", err)
		return nil, err
	}

	// Execute the tool calls (functions)
	newHistory, err := s.ExecuteToolCalls(ctx, msgHistory, resp, sessionID)
	if err != nil {
		s.logger.Error("Failed to execute tool calls: ", err)
		return nil, err
	}

	if resp.Choices[0].FuncCall == nil {
		newHistory = append(newHistory, llms.TextParts(llms.ChatMessageTypeAI, resp.Choices[0].Content))
		outputMessage = resp.Choices[0].Content
	} else if resp.Choices[0].FuncCall.Name != endSessionFuncName {
		// If the function call is not endSession, recursively call message sending
		afterFuncRes, err := s.SendMessage(ctx, sessionID, resp.Choices[0].FuncCall.Name+" completed. Continue to another step.", MessageTypeAI)
		if err != nil {
			return nil, err
		}
		outputMessage = afterFuncRes.Message
	} else {
		// If the function call is endSession, close the session
		s.CloseSession(sessionID)

	}

	s.chatService.SaveMessageHistory(&newHistory, sessionID)

	s.logger.Debug("Message sent: ", resp.Choices[0].Content)
	s.logger.Debug("Function call: ", resp.Choices[0].FuncCall)
	s.logger.Debug("Message history: ", newHistory)

	return &StartConversationResponse{
		Message:   outputMessage,
		SessionID: sessionID,
	}, nil
}

// CloseSession closes the session and deletes the chat history
func (s *AiService) CloseSession(sessionID string) {
	s.streamStore.CloseSession(sessionID)
	s.chatService.DeleteChatHistory(sessionID)
}
