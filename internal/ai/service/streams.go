package service

import (
	"fmt"
	"sync"

	"connectrpc.com/connect"

	aiv1 "github.com/bxxf/znvo-backend/gen/api/ai/v1"
)

type StreamStore struct {
	streams map[string]*connect.ServerStream[aiv1.StartSessionResponse]
	mu      sync.Mutex
	msgChan map[string]chan *aiv1.StartSessionResponse
}

func NewStreamStore() *StreamStore {
	return &StreamStore{
		streams: make(map[string]*connect.ServerStream[aiv1.StartSessionResponse]),
		msgChan: make(map[string]chan *aiv1.StartSessionResponse),
	}
}

func (s *StreamStore) SaveStream(sessionID string, stream *connect.ServerStream[aiv1.StartSessionResponse]) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.streams[sessionID] = stream
	s.msgChan[sessionID] = make(chan *aiv1.StartSessionResponse, 10)
	go s.handleStream(sessionID)
}

func (s *StreamStore) GetStream(sessionID string) (*connect.ServerStream[aiv1.StartSessionResponse], bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stream, exists := s.streams[sessionID]
	return stream, exists
}

func (s *StreamStore) handleStream(sessionID string) {
	for msg := range s.msgChan[sessionID] {
		if stream, exists := s.streams[sessionID]; exists {
			stream.Send(msg)
		}
	}
}

func (s *StreamStore) SendMessage(sessionID string, msg *aiv1.StartSessionResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ch, ok := s.msgChan[sessionID]; ok {
		ch <- msg
	}
}

func (s *StreamStore) CloseSession(sessionID string) {
	fmt.Printf("Closing session %s\n", sessionID)
	if _, exists := s.streams[sessionID]; !exists {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	close(s.msgChan[sessionID])
	delete(s.streams, sessionID)
	delete(s.msgChan, sessionID)
}
