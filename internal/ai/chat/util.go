package chat

import "github.com/tmc/langchaingo/llms"

func convertToLLMS(messages []CustomMessageContent) []llms.MessageContent {
	var llmsMessages []llms.MessageContent
	for _, msg := range messages {
		var role llms.ChatMessageType
		var parts []llms.ContentPart
		if msg.Role == "human" {
			role = llms.ChatMessageTypeHuman
		} else if msg.Role == "system" {
			role = llms.ChatMessageTypeSystem
		} else if msg.Role == "ai" {
			role = llms.ChatMessageTypeAI
		} else {
			role = llms.ChatMessageTypeSystem
		}

		for _, part := range msg.Parts {
			parts = append(parts, llms.TextPart(part.Text))
		}
		llmsMessages = append(llmsMessages, llms.MessageContent{
			Role:  role,
			Parts: parts,
		})
	}
	return llmsMessages
}
