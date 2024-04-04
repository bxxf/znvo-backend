package util

import (
	"sync"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

// SessionMap is a thread-safe map to store session data
var SessionMap = struct {
	sync.RWMutex
	sessions map[string]*webauthn.SessionData
}{sessions: make(map[string]*webauthn.SessionData)}

func NewSession(data *webauthn.SessionData) string {
	id := uuid.New().String()
	SessionMap.Lock()
	SessionMap.sessions[id] = data
	SessionMap.Unlock()
	return id
}

func GetSession(sessionID string) (*webauthn.SessionData, bool) {
	SessionMap.RLock()
	data, ok := SessionMap.sessions[sessionID]
	SessionMap.RUnlock()
	// remove session from map to prevent replay attacks
	go func() {
		if ok {
			SessionMap.Lock()
			delete(SessionMap.sessions, sessionID)
			SessionMap.Unlock()
		}
	}()
	return data, ok
}
