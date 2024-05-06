package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tmc/langchaingo/llms"

	"github.com/bxxf/znvo-backend/gen/api/ai/v1"
	aiv1 "github.com/bxxf/znvo-backend/gen/api/ai/v1"
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
									"description": "Duration of the activity as a string (e.g., '30 minutes', '1 hour'). Can be empty if the user doesn't know the duration. DO NOT GUESS the duration - if the user doesn't know, it's better to leave it empty",
								},
								"time": map[string]any{
									"type":        "number",
									"description": "How long AGO the activity took place in MINUTES (e.g., 5, 10, 15, 120). Can be empty or 0 if the user doesn't know the time OR if it's happening now - 0 means the activity is happening now. If the activity is happening now, the time should be 0 + duration of the activity in minutes.",
								},
								"mood": map[string]any{
									"type":        "number",
									"description": "Mood level of the user during the activity (0-100) - can be on a scale 1-10 (times ten). If the user doesn't know the mood, it can be empty. DO NOT GUESS.",
								},
							},
							"required": []string{"name", "mood", "time"},
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
			Description: "End the session. Gets called after all steps are done or 'endSession' is prompted",
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
func (s *AiService) ExecuteToolCalls(ctx context.Context, messageHistory []llms.MessageContent, resp *llms.ContentResponse, streamID string, messageID string) ([]llms.MessageContent, error) {
	if len(resp.Choices[0].ToolCalls) == 0 {
		return messageHistory, nil
	}
	switch resp.Choices[0].ToolCalls[0].FunctionCall.Name {

	case "parseActivities":
		return messageHistory, s.handleParseActivities(resp.Choices[0].ToolCalls[0].FunctionCall.Arguments, streamID, messageID)
	case "endSession":
		return messageHistory, s.handleEndSession(resp.Choices[0].ToolCalls[0].FunctionCall.Arguments, streamID)
	default:
		s.logger.Info("Unknown tool call: ", resp.Choices[0].ToolCalls[0].FunctionCall.Name)
		return messageHistory, nil
	}

	return messageHistory, nil
}

func (s *AiService) handleParseActivities(args string, streamID string, messageId string) error {
	var activities struct {
		Activities []Activity `json:"activities"`
	}
	if err := json.Unmarshal([]byte(args), &activities); err != nil {
		return err
	}

	state, exists := s.streamStore.sessionState[streamID]
	s.streamStore.mu.Unlock()
	if !exists {
		return fmt.Errorf("session state not found")
	}

	if state.HasCalledParseActivities {
		s.logger.Info("parseActivities has already been called for this session: ", streamID)
		return nil
	}

	state.HasCalledParseActivities = true

	for i, activity := range activities.Activities {
		currentTime := time.Now()

		if activity.Time > 0 {
			currentTime = currentTime.Add(time.Duration(-activity.Time) * time.Minute)
		}

		activity.Time = int(currentTime.Unix())

		activities.Activities[i] = activity

	}

	responseJSON, err := json.Marshal(activities.Activities)
	if err != nil {
		return err
	}

	s.streamStore.SendMessage(streamID, &ai.StartSessionResponse{
		Message:     string(responseJSON),
		MessageId:   messageId,
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
		MessageId:   "end",
		MessageType: aiv1.MessageType_ENDSESSION,
	})

	s.chatService.DeleteChatHistory(streamID)

	return nil
}

type Activity struct {
	Name     string `json:"name"`
	Duration string `json:"duration"`
	Time     int    `json:"time"`
	Mood     string `json:"mood"`
}
