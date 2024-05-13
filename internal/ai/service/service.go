package service

import (
	"context"
	"strings"
	"time"

	"github.com/nrednav/cuid2"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"

	"github.com/bxxf/znvo-backend/gen/api/ai/v1"
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
	MessageId string
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

	messageId := cuid2.Generate()
	skipStreaming := false

	// Generate content based on the message history
	resp, err := s.llm.GenerateContent(ctx, msgHistory, llms.WithTools(AvailableTools), llms.WithTemperature(0.35), llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {

		// if chunk is json skip streaming for whole message id
		if strings.Contains(string(chunk), "{") {
			skipStreaming = true
		}

		if !skipStreaming {

			s.streamStore.SendMessage(sessionID, &ai.StartSessionResponse{
				Message:     string(chunk),
				SessionId:   sessionID,
				MessageId:   messageId,
				MessageType: ai.MessageType_CHAT_PARTIAL,
			})
		}

		return nil
	}))
	if err != nil {
		s.logger.Error("Failed to generate content: ", err)
		return nil, err
	}

	// Execute the tool calls (functions)
	newHistory, err := s.ExecuteToolCalls(ctx, msgHistory, resp, sessionID, messageId)
	if err != nil {
		s.logger.Error("Failed to execute tool calls: ", err)
		return nil, err
	}

	if resp.Choices[0].FuncCall == nil {
		newHistory = append(newHistory, llms.TextParts(llms.ChatMessageTypeAI, resp.Choices[0].Content))
		outputMessage = resp.Choices[0].Content
		s.chatService.SaveMessageHistory(&newHistory, sessionID)

	} else if resp.Choices[0].FuncCall.Name != endSessionFuncName {
		s.chatService.SaveMessageHistory(&newHistory, sessionID)

		var afterFuncRes *StartConversationResponse
		afterFuncRes, err = s.SendMessage(ctx, sessionID, resp.Choices[0].FuncCall.Name+" completed. Continue to another step.", MessageTypeAI)
		if err != nil {
			return nil, err
		}
		messageHistoryPointer, err := s.chatService.LoadMessageHistory(sessionID)

		if err != nil {
			s.logger.Error("Failed to load message history: ", err)
			return nil, err
		}
		msgHistory = *messageHistoryPointer
		s.chatService.SaveMessageHistory(&msgHistory, sessionID)
		outputMessage = afterFuncRes.Message
	} else {
		s.streamStore.CloseSession(sessionID)
		s.chatService.DeleteChatHistory(sessionID)
	}

	//s.logger.Debug("Message sent: ", resp.Choices[0].Content)
	//s.logger.Debug("Function call: ", resp.Choices[0].FuncCall)
	//s.logger.Debug("Message history: ", msgHistory)

	return &StartConversationResponse{
		Message:   outputMessage,
		MessageId: messageId,
		SessionID: sessionID,
	}, nil
}

func (s *AiService) generateUniqueSessionID() string {
	sessionID := cuid2.Generate()
	for {
		_, found := s.streamStore.GetStream(sessionID)
		if !found {
			break
		}
		s.logger.Error("Stream already exists")
		sessionID = cuid2.Generate()
	}
	return sessionID
}
