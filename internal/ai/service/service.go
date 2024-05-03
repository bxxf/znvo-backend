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

type AiService struct {
	logger      *logger.LoggerInstance
	llm         *openai.LLM
	streamStore *StreamStore
	chatService *chat.ChatService
}

type StartConversationResponse struct {
	Message   string
	SessionID string
}

func NewAiService(logger *logger.LoggerInstance, streamStore *StreamStore, chatService *chat.ChatService) *AiService {
	llm := InitializeModel()
	return &AiService{
		logger:      logger,
		streamStore: streamStore,
		llm:         llm,
		chatService: chatService,
	}
}

func InitializeModel() *openai.LLM {
	llm, err := openai.New(openai.WithModel("gpt-3.5-turbo"))
	if err != nil {
		panic(err)
	}
	return llm
}

/**
 * StartConversation - starts a conversation with the AI model and returns the stream
 * @return StartConversationResponse - the response containing the message and session id
 * @return error - error if the conversation fails to start
 */
func (s *AiService) StartConversation() (*StartConversationResponse, error) {

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	// define initial message history with prompt
	messageHistory := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, prompt.Prompt),
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

	// generate a new session id
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
	_, err = s.chatService.SaveMessageHistory(&messageHistory, sessionId)
	if err != nil {
		s.logger.Error("Failed to save message history: ", err)
		return nil, err
	}

	return &StartConversationResponse{
		Message:   resp.Choices[0].Content,
		SessionID: sessionId,
	}, nil
}

/**
 * SendMessage - sends a message to the AI model and returns the response
 * @param sessionId string - the session id for the conversation
 * @param message string - the message to send
 * @param mtype string - the type of message (ai or user)
 * @return StartConversationResponse - the response containing the message and session id
 * @return error - error if the message fails to send
 */
func (s *AiService) SendMessage(sessionId string, message string, mtype string) (*StartConversationResponse, error) {
	var outputMessage string
	var msg llms.MessageContent
	messageHistory, error := s.chatService.LoadMessageHistory(sessionId)

	if error != nil {
		s.logger.Error("Failed to load message history: ", error)
		return nil, error
	}

	msgHistory := *messageHistory

	// check if the message type is ai or user - if ai, set the role to system
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

	// generate content based on the message history
	resp, err := s.llm.GenerateContent(context.Background(), msgHistory, llms.WithTools(availableTools))
	if err != nil {
		s.logger.Error("Failed to generate content: ", err)
		return nil, err
	}

	// execute the tool calls (functions)
	s.executeToolCalls(context.Background(), msgHistory, resp, sessionId)

	// if the response contains function calls, run the model again with prompt to go to another step to prevent stopping the conversation
	if resp.Choices[0].FuncCall == nil {
		msgHistory = append(msgHistory, llms.TextParts(llms.ChatMessageTypeAI, resp.Choices[0].Content))
		s.chatService.SaveMessageHistory(&msgHistory, sessionId)
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

	return &StartConversationResponse{
		Message:   outputMessage,
		SessionID: sessionId,
	}, nil
}

func (s *AiService) CloseSession(sessionId string) {
	s.streamStore.CloseSession(sessionId)
	s.chatService.DeleteChatHistory(sessionId)
}
