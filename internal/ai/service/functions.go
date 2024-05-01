package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/bxxf/znvo-backend/gen/api/ai/v1"
	aiv1 "github.com/bxxf/znvo-backend/gen/api/ai/v1"
	"github.com/bxxf/znvo-backend/internal/logger"
	"github.com/tmc/langchaingo/llms"
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
}

type Activity struct {
	Name     string `json:"name"`
	Duration string `json:"duration"`
	Time     string `json:"time"`
}

func (s *AiService) executeToolCalls(ctx context.Context, llm llms.Model, messageHistory []llms.MessageContent, resp *llms.ContentResponse, streamID string) []llms.MessageContent {
	logger := logger.NewLogger()
	for _, toolCall := range resp.Choices[0].ToolCalls {
		logger.Info("Tool call: ", toolCall.FunctionCall.Name)
		fmt.Println("Tool args: ", toolCall.FunctionCall.Arguments)
		switch toolCall.FunctionCall.Name {
		case "parseActivities":
			var args struct {
				Activities []Activity `json:"activities"`
			}
			if err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &args); err != nil {
				log.Fatal(err)
			}

			// Convert the response back to a JSON string
			responseJSON, err := json.Marshal(args.Activities)

			if err != nil {
				log.Fatal(err)
			}

			fmt.Printf("Response: %s\n", responseJSON)

			s.streamStore.SendMessage(streamID, &ai.StartSessionResponse{
				Message:     string(responseJSON),
				MessageType: aiv1.MessageType_ACTIVITIES,
			})
			msgHistory, ok := LoadMessageHistory(streamID)
			if !ok {
				panic("Could not load message history")
			}

			noPointer := *msgHistory
			noPointer = append(noPointer, []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, "Send another question or exit the conversation")}...)
			//	resp, err := s.llm.GenerateContent(context.Background(), noPointer, llms.WithTools(availableTools))

			if err != nil {
				log.Fatal(err)
			}
			/*
				s.streamStore.SendMessage(streamID, &ai.StartSessionResponse{
					Message:     string(resp.Choices[0].Content),
					MessageType: aiv1.MessageType_CHAT,
				})

				/*

					activityCallRes := llms.MessageContent{
						Role: llms.ChatMessageTypeTool,
						Parts: []llms.ContentPart{
							llms.ToolCallResponse{
								ToolCallID: toolCall.ID,
								Name:       toolCall.FunctionCall.Name,
								Content:    string(responseJSON),
							},
						},
					}
					messageHistory = append(messageHistory, activityCallRes)
			*/

		default:
			log.Fatalf("Unsupported tool: %s", toolCall.FunctionCall.Name)
		}
	}

	return messageHistory
}
