package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tmc/langchaingo/llms"

	ai "github.com/bxxf/znvo-backend/gen/api/ai/v1"
	aiv1 "github.com/bxxf/znvo-backend/gen/api/ai/v1"
)

type Activity struct {
	Name     string `json:"name"`
	Duration string `json:"duration"`
	Time     int    `json:"time"`
	Mood     int    `json:"mood"`
}

type Meal struct {
	Name string `json:"name"`
	Time int    `json:"time"`
	Mood int    `json:"mood"`
}

// Tool represents a function that can be called by the AI
var AvailableTools = []llms.Tool{
	newTool("parseActivities", "Get user's activities for the day based on their responses and return it in a structured format", newActivitiesSchema()),
	newTool("parseFood", "Get user's food for the day based on their responses and return it in a structured format", newMealsSchema()),
	newTool("endSession", "End the session. This gets called at the end of the conversation to close the session or ENDSESSION prompt", newMessageSchema()),
}

// Utility functions for creating tools
func newTool(name, description string, parameters map[string]any) llms.Tool {
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        name,
			Description: description,
			Parameters:  parameters,
		},
	}
}

func newActivitiesSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"activities": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name":     newProperty("string", "Full name of the activity (e.g., 'Running', 'Reading', 'Cooking')"),
						"duration": newProperty("string", "Duration of the activity as a string (e.g., '30 minutes', '1 hour'). Can be empty if the user doesn't know the duration. DO NOT GUESS the duration - if the user doesn't know, it's better to leave it empty"),
						"time":     newProperty("number", "How long AGO the activity took place in MINUTES (e.g., 5, 10, 15, 120). Can be empty or 0 if the user doesn't know the time OR if it's happening now - 0 means the activity is happening now. If the activity is happening now, the time should be 0 + duration of the activity in minutes."),
						"mood":     newProperty("number", "Mood level of the user during the activity (0-100) - can be on a scale 1-10 (times ten). If the user doesn't know the mood, it can be empty. DO NOT GUESS."),
					},
					"required": []string{"name", "mood", "time"},
				},
			},
		},
		"required": []string{"activities"},
	}
}

func newMealsSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"meals": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": newProperty("string", "Full name of the food (e.g., 'Apple', 'Pizza', 'Salad')"),
						"time": newProperty("number", "How long AGO the food was eaten in MINUTES (e.g., 5, 10, 15, 120). Can be empty or 0 if the user doesn't know the time OR if it's happening now - 0 means the food was eaten now. If the food was eaten now, the time should be 0."),
						"mood": newProperty("number", "Mood level of the user after eating the food (0-100) - can be on a scale 1-10 (times ten). If the user doesn't know the mood, it can be empty. DO NOT GUESS."),
					},
					"required": []string{"name", "mood", "time"},
				},
			},
		},
		"required": []string{"meals"},
	}
}

func newMessageSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"message": newProperty("string", "A message to display to the user before ending the session"),
		},
		"required": []string{"message"},
	}
}

func newProperty(dataType, description string) map[string]any {
	return map[string]any{
		"type":        dataType,
		"description": description,
	}
}

func (s *AiService) ExecuteToolCalls(ctx context.Context, messageHistory []llms.MessageContent, resp *llms.ContentResponse, streamID string, messageID string) ([]llms.MessageContent, error) {
	if len(resp.Choices[0].ToolCalls) == 0 {
		return messageHistory, nil
	}

	s.handlers = map[string]func(string, string, string) error{
		"parseActivities":         s.handleParseActivities,
		"parseFood":               s.handleParseFood,
		"endSession":              s.handleEndSession,
		"multi_tool_use.parallel": s.handleMultiToolUseParallel,
	}

	if handler, ok := s.handlers[resp.Choices[0].ToolCalls[0].FunctionCall.Name]; ok {
		return messageHistory, handler(resp.Choices[0].ToolCalls[0].FunctionCall.Arguments, streamID, messageID)
	}

	s.logger.Info("Unknown tool call: ", resp.Choices[0].ToolCalls[0].FunctionCall.Name)
	return messageHistory, fmt.Errorf("unknown tool call: %s", resp.Choices[0].ToolCalls[0].FunctionCall.Name)
}

func (s *AiService) handleMultiToolUseParallel(args string, streamID string, messageId string) error {
	var toolCalls struct {
		ToolCalls []llms.ToolCall `json:"toolCalls"`
	}
	if err := json.Unmarshal([]byte(args), &toolCalls); err != nil {
		return fmt.Errorf("failed to unmarshal tool calls: %v", err)
	}
	fmt.Printf("toolCalls: %v\n", toolCalls)

	for _, toolCall := range toolCalls.ToolCalls {
		if handler, ok := s.handlers[toolCall.FunctionCall.Name]; ok {
			if err := handler(toolCall.FunctionCall.Arguments, streamID, messageId); err != nil {
				return err
			}
		} else {
			s.logger.Info("Unknown tool call: ", toolCall.FunctionCall.Name)
			return fmt.Errorf("unknown tool call: %s", toolCall.FunctionCall.Name)
		}
	}
	return nil
}

func (s *AiService) handleParseActivities(args string, streamID string, messageId string) error {
	var activities struct {
		Activities []Activity `json:"activities"`
	}
	if err := json.Unmarshal([]byte(args), &activities); err != nil {
		return fmt.Errorf("failed to unmarshal activities: %v", err)
	}

	activities.Activities = updateActivityTimes(activities.Activities)

	responseJSON, err := json.Marshal(activities.Activities)
	if err != nil {
		return fmt.Errorf("failed to marshal activities: %v", err)
	}

	s.mu.Lock()
	sessionChannel, ok := s.sessionChannels[streamID]
	s.mu.Unlock()
	if !ok {
		s.logger.Error("No active session for this ID")
		return nil
	}
	message := &ai.StartSessionResponse{
		Message:     string(responseJSON),
		SessionId:   streamID,
		MessageId:   messageId,
		MessageType: ai.MessageType_ACTIVITIES,
	}
	sessionChannel <- message

	return nil
}

func (s *AiService) handleParseFood(args string, streamID string, messageId string) error {
	var meals struct {
		Meals []Meal `json:"meals"`
	}
	if err := json.Unmarshal([]byte(args), &meals); err != nil {
		return fmt.Errorf("failed to unmarshal meals: %v", err)
	}

	meals.Meals = updateMealTimes(meals.Meals)

	responseJSON, err := json.Marshal(meals.Meals)
	if err != nil {
		return fmt.Errorf("failed to marshal meals: %v", err)
	}

	s.logger.Info("Adding food to session: ", streamID)

	s.sendMessageViaChannel(&ai.StartSessionResponse{
		Message:     string(responseJSON),
		SessionId:   streamID,
		MessageId:   messageId,
		MessageType: ai.MessageType_NUTRITION,
	})
	return nil
}

func (s *AiService) handleEndSession(args string, streamID string, placeholder string) error {
	var message struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(args), &message); err != nil {
		return fmt.Errorf("failed to unmarshal end session message: %v", err)
	}

	s.sendMessageViaChannel(&ai.StartSessionResponse{
		Message:     message.Message,
		SessionId:   streamID,
		MessageId:   "end",
		MessageType: aiv1.MessageType_ENDSESSION,
	})

	s.chatService.DeleteChatHistory(streamID)
	return nil
}

func updateActivityTimes(activities []Activity) []Activity {
	currentTime := time.Now()
	for i, activity := range activities {
		if activity.Time > 0 {
			currentTime = currentTime.Add(time.Duration(-activity.Time) * time.Minute)
		}
		activities[i].Time = int(currentTime.Unix())
	}
	return activities
}

func updateMealTimes(meals []Meal) []Meal {
	currentTime := time.Now()
	for i, meal := range meals {
		if meal.Time > 0 {
			currentTime = currentTime.Add(time.Duration(-meal.Time) * time.Minute)
		}
		meals[i].Time = int(currentTime.Unix())
	}
	return meals
}
