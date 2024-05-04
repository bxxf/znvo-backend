package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tmc/langchaingo/llms"

	"github.com/bxxf/znvo-backend/gen/api/ai/v1"
	aiv1 "github.com/bxxf/znvo-backend/gen/api/ai/v1"
	"github.com/bxxf/znvo-backend/internal/logger"
)

// AvailableTools defines a list of functions that can be called by the LLM
var AvailableTools = []llms.Tool{
	{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "parseActivities",
			Description: "Get user's activities for the day based on their responses and return it in a structured format",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"activities": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"name": map[string]any{
									"type":        "string",
									"description": "Full name of the activity (e.g., 'Running', 'Reading', 'Cooking')",
								},
								"duration": map[string]any{
									"type":        "string",
									"description": "Duration of the activity as a string (e.g., '30 minutes', '1 hour')",
								},
								"time": map[string]any{
									"type":        "string",
									"description": "Exact time of the activity as a string (e.g., '8:00', '21:30')",
								},
								"mood": map[string]any{
									"type":        "string",
									"description": "Mood level of the user during the activity (0-100)",
								},
							},
							"required": []string{"name", "mood"},
						},
					},
				},
				"required": []string{"activities"},
			},
		},
	},
	{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "endSession",
			Description: "End the session",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"message": map[string]any{
						"type":        "string",
						"description": "A message to display to the user before ending the session",
					},
				},
				"required": []string{"message"},
			},
		},
	},
}

// ExecuteToolCalls handles the invocation of tools based on the response choices and manages concurrency and error handling.
func (s *AiService) ExecuteToolCalls(ctx context.Context, messageHistory []llms.MessageContent, resp *llms.ContentResponse, streamID string) []llms.MessageContent {
	logger := logger.NewLogger()
	toolCallCount := len(resp.Choices[0].ToolCalls)
	done := make(chan bool, toolCallCount)
	errs := make(chan error, toolCallCount)

	for _, toolCall := range resp.Choices[0].ToolCalls {
		go func(toolCall llms.ToolCall) {
			logger.Info("Tool call: ", toolCall.FunctionCall.Name)
			fmt.Println("Tool args: ", toolCall.FunctionCall.Arguments)

			if err := s.handleToolCall(toolCall, streamID); err != nil {
				errs <- err
			} else {
				done <- true
			}
		}(toolCall)
	}

	for i := 0; i < toolCallCount; i++ {
		select {
		case <-done:
		case err := <-errs:
			logger.Error("Error executing tool call: ", err)
			return messageHistory
		}
	}

	return messageHistory
}

func (s *AiService) handleToolCall(toolCall llms.ToolCall, streamID string) error {
	switch toolCall.FunctionCall.Name {
	case "parseActivities":
		return s.handleParseActivities(toolCall.FunctionCall.Arguments, streamID)
	case "endSession":
		return s.handleEndSession(toolCall.FunctionCall.Arguments, streamID)
	default:
		logger.NewLogger().Info("Unknown tool call: ", toolCall.FunctionCall.Name)
		return nil
	}
}

func (s *AiService) handleParseActivities(args string, streamID string) error {
	var activities struct {
		Activities []Activity `json:"activities"`
	}
	if err := json.Unmarshal([]byte(args), &activities); err != nil {
		return err
	}

	responseJSON, err := json.Marshal(activities.Activities)
	if err != nil {
		return err
	}

	s.streamStore.SendMessage(streamID, &ai.StartSessionResponse{
		Message:     string(responseJSON),
		SessionId:   streamID,
		MessageType: aiv1.MessageType_ACTIVITIES,
	})

	return nil
}

func (s *AiService) handleEndSession(args string, streamID string) error {
	var message struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(args), &message); err != nil {
		return err
	}

	s.streamStore.SendMessage(streamID, &ai.StartSessionResponse{
		Message:     message.Message,
		SessionId:   streamID,
		MessageType: aiv1.MessageType_ENDSESSION,
	})

	s.chatService.DeleteChatHistory(streamID)

	return nil
}

type Activity struct {
	Name     string `json:"name"`
	Duration string `json:"duration"`
	Time     string `json:"time"`
	Mood     string `json:"mood"`
}
