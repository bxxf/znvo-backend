package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"

	"github.com/bxxf/znvo-backend/internal/logger"
)

type AiService struct {
	logger      *logger.LoggerInstance
	llm         *openai.LLM
	streamStore *StreamStore
}

type StartConversationResponse struct {
	Message   string
	SessionID string
}

func NewAiService(logger *logger.LoggerInstance, streamStore *StreamStore) *AiService {
	llm := InitializeModel()
	return &AiService{
		logger:      logger,
		streamStore: streamStore,
		llm:         llm,
	}
}

func InitializeModel() *openai.LLM {
	llm, err := openai.New(openai.WithModel("gpt-3.5-turbo"))
	if err != nil {
		panic(err)
	}
	return llm
}

func (s *AiService) StartConversation() (*StartConversationResponse, error) {

	ctx := context.Background()
	// define initial message history with prompt
	messageHistory := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, prompt),
	}
	// generate first message based on the prompt
	resp, err := s.llm.GenerateContent(ctx, messageHistory, llms.WithTools(availableTools))

	if err != nil {
		s.logger.Error("Failed to generate content: ", err)
		return nil, err
	}

	// create message content struct for the response
	newContent := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeAI, resp.Choices[0].Content),
	}

	// append the response to the message history
	messageHistory = append(messageHistory, newContent...)
	sessionId := uuid.New().String()

	// if the stream already exists, generate a new uid
	_, found := s.streamStore.GetStream(sessionId)

	// loop until a unique stream id is found
	for found {
		s.logger.Error("Stream already exists")
		sessionId = uuid.New().String()
		_, found = s.streamStore.GetStream(sessionId)
	}

	// save the message history to the stream store
	SaveMessageHistory(&messageHistory, sessionId)

	return &StartConversationResponse{
		Message:   resp.Choices[0].Content,
		SessionID: sessionId,
	}, nil
}

func (s *AiService) SendMessage(sessionId string, message string, mtype string) (*StartConversationResponse, error) {
	var outputMessage string
	var msg llms.MessageContent
	messageHistory, ok := LoadMessageHistory(sessionId)
	if !ok {
		s.logger.Error("Failed to load message history")
		return nil, errors.New("Failed to load message history")
	}

	msgHistory := *messageHistory

	if mtype == "ai" {
		msg = llms.MessageContent{
			Role: llms.ChatMessageTypeSystem,
			Parts: []llms.ContentPart{
				llms.TextPart(message),
			},
		}
	} else {

		msg = llms.MessageContent{
			Role: llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{
				llms.TextPart(message),
			},
		}
	}

	msgHistory = append(msgHistory, msg)

	resp, err := s.llm.GenerateContent(context.Background(), msgHistory, llms.WithTools(availableTools))
	if err != nil {
		s.logger.Error("Failed to generate content: ", err)
		return nil, err
	}
	s.executeToolCalls(context.Background(), msgHistory, resp, sessionId)

	if resp.Choices[0].FuncCall == nil {
		msgHistory = append(msgHistory, llms.TextParts(llms.ChatMessageTypeAI, resp.Choices[0].Content))
		SaveMessageHistory(&msgHistory, sessionId)
		outputMessage = resp.Choices[0].Content

		// if the function call is not endSession then recursively call message sending
	} else if resp.Choices[0].FuncCall.Name != "endSession" {

		nRes, err := s.SendMessage(sessionId, resp.Choices[0].FuncCall.Name+" completed. Continue to another step.", "ai")
		if err != nil {
			return nil, err
		}
		outputMessage = nRes.Message
	}

	s.logger.Debug("Message sent: ", resp.Choices[0].Content)
	s.logger.Debug("Function call: ", resp.Choices[0].FuncCall)
	s.logger.Debug("Message history: ", msgHistory)

	return &StartConversationResponse{
		Message:   outputMessage,
		SessionID: sessionId,
	}, nil
}
