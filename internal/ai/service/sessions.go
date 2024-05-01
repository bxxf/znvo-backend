package service

import (
	"sync"

	"github.com/tmc/langchaingo/llms"
)

// SessionMap is a thread-safe map to store session data
var SessionMap = struct {
	sync.RWMutex
	sessions map[string]*[]llms.MessageContent
}{sessions: make(map[string]*[]llms.MessageContent)}

func SaveMessageHistory(data *[]llms.MessageContent, id string) string {
	SessionMap.Lock()
	SessionMap.sessions[id] = data
	SessionMap.Unlock()
	return id
}

func LoadMessageHistory(sessionID string) (*[]llms.MessageContent, bool) {
	SessionMap.RLock()
	data, ok := SessionMap.sessions[sessionID]
	SessionMap.RUnlock()

	return data, ok
}
