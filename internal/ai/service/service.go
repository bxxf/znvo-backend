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
	messageHistory := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, prompt),
	}

	resp, err := s.llm.GenerateContent(ctx, messageHistory, llms.WithTools(availableTools))

	if err != nil {
		s.logger.Error("Failed to generate content: ", err)
		return nil, err
	}

	newContent := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeAI, resp.Choices[0].Content),
	}

	messageHistory = append(messageHistory, newContent...)
	uid := uuid.New().String()

	_, found := s.streamStore.GetStream(uid)
	if found {
		s.logger.Error("Stream already exists")
		uid = uuid.New().String()
	}

	SaveMessageHistory(&messageHistory, uid)

	return &StartConversationResponse{
		Message:   resp.Choices[0].Content,
		SessionID: uid,
	}, nil
}

func (s *AiService) SendMessage(sessionId string, message string) (*StartConversationResponse, error) {
	var outMsg string
	messageHistory, ok := LoadMessageHistory(sessionId)
	if !ok {
		s.logger.Error("Failed to load message history")
		return nil, errors.New("Failed to load message history")
	}

	msgHistory := *messageHistory

	msg := llms.MessageContent{
		Role: llms.ChatMessageTypeHuman,
		Parts: []llms.ContentPart{
			llms.TextPart(message),
		},
	}

	msgHistory = append(msgHistory, msg)

	resp, err := s.llm.GenerateContent(context.Background(), msgHistory, llms.WithTools(availableTools))
	if err != nil {
		s.logger.Error("Failed to generate content: ", err)
		return nil, err
	}

	s.executeToolCalls(context.Background(), s.llm, msgHistory, resp, sessionId)

	msgHistory = append(msgHistory, []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeAI, resp.Choices[0].Content)}...)
	if resp.Choices[0].Content != "" {
		outMsg = resp.Choices[0].Content
	} else {

		msgHistory = append(msgHistory, []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, "Send another question or exit the conversation")}...)
		resp, err := s.llm.GenerateContent(context.Background(), msgHistory, llms.WithTools(availableTools))
		if err != nil {
			s.logger.Error("Failed to generate content: ", err)
			return nil, err
		}

		msgHistory = append(msgHistory, []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeAI, resp.Choices[0].Content)}...)

		outMsg = resp.Choices[0].Content

	}

	SaveMessageHistory(&msgHistory, sessionId)

	return &StartConversationResponse{
		Message:   outMsg,
		SessionID: sessionId,
	}, nil
}
