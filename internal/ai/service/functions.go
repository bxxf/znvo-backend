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

var availableTools = []llms.Tool{
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
						"items": map[string]any{ // Each item in the array should be an object
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
									"description": "Time of the activity when it was done as a 24 hour string (e.g., '8:00', '12:00'). If it's now, do not provide this field.",
								},
							},
							"required": []string{"name"},
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

type Activity struct {
	Name     string `json:"name"`
	Duration string `json:"duration"`
	Time     string `json:"time"`
}

func (s *AiService) executeToolCalls(ctx context.Context, messageHistory []llms.MessageContent, resp *llms.ContentResponse, streamID string) []llms.MessageContent {
	logger := logger.NewLogger()
	done := make(chan bool)
	errors := make(chan error)

	for _, toolCall := range resp.Choices[0].ToolCalls {

		go func(toolCall llms.ToolCall) {
			logger.Info("Tool call: ", toolCall.FunctionCall.Name)
			fmt.Println("Tool args: ", toolCall.FunctionCall.Arguments)

			switch toolCall.FunctionCall.Name {
			case "parseActivities":
				var args struct {
					Activities []Activity `json:"activities"`
				}
				err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &args)
				if err != nil {
					errors <- err
					return
				}
				responseJSON, err := json.Marshal(args.Activities)
				if err != nil {
					errors <- err
					return
				}
				s.streamStore.SendMessage(streamID, &ai.StartSessionResponse{
					Message:     string(responseJSON),
					SessionId:   streamID,
					MessageType: aiv1.MessageType_ACTIVITIES,
				})

			case "endSession":
				var args struct {
					Message string `json:"message"`
				}
				err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &args)
				if err != nil {
					errors <- err
					return
				}
				s.streamStore.SendMessage(streamID, &ai.StartSessionResponse{
					Message:     args.Message,
					SessionId:   streamID,
					MessageType: aiv1.MessageType_CHAT,
				})
				s.streamStore.CloseSession(streamID)
			default:
				logger.Info("Unknown tool call: ", toolCall.FunctionCall.Name)
			}
			done <- true
		}(toolCall)
	}

	for i := 0; i < len(resp.Choices[0].ToolCalls); i++ {
		select {
		case <-done:
			// continue
		case err := <-errors:
			logger.Error("Error executing tool call: ", err)
			return messageHistory
		}
	}
	return messageHistory
}
