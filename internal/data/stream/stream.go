package stream

import (
	"fmt"
	"sync"

	"connectrpc.com/connect"

	datav1 "github.com/bxxf/znvo-backend/gen/api/data/v1"
)

type StreamStore struct {
	Streams map[string]*connect.ServerStream[datav1.GetSharedDataResponse]
	mu      sync.Mutex
	msgChan map[string]chan *datav1.GetSharedDataResponse
}

func NewStreamStore() *StreamStore {
	return &StreamStore{
		Streams: make(map[string]*connect.ServerStream[datav1.GetSharedDataResponse]),
		msgChan: make(map[string]chan *datav1.GetSharedDataResponse),
	}
}

func (s *StreamStore) SaveStream(stream *connect.ServerStream[datav1.GetSharedDataResponse], userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Streams[userID] = stream
	s.msgChan[userID] = make(chan *datav1.GetSharedDataResponse)
	go s.handleStream(userID)
}

func (s *StreamStore) GetStream(userID string) (*connect.ServerStream[datav1.GetSharedDataResponse], bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stream, exists := s.Streams[userID]
	return stream, exists
}

func (s *StreamStore) SendMessage(userID string, msg *datav1.GetSharedDataResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ch, ok := s.msgChan[userID]; ok {
		ch <- msg
	}
}

func (s *StreamStore) CloseSession(userID string) {
	fmt.Printf("Closing stream for data for %s\n", userID)
	s.mu.Lock()
	if _, exists := s.Streams[userID]; !exists {
		s.mu.Unlock()
		return
	}
	close(s.msgChan[userID])
	delete(s.Streams, userID)
	delete(s.msgChan, userID)
	s.mu.Unlock()
}

func (s *StreamStore) handleStream(userID string) {
	ch, ok := s.msgChan[userID]
	if !ok {
		return // Safe-guard against missing channel initialization
	}

	for msg := range ch {
		s.mu.Lock()
		stream, exists := s.Streams[userID]
		s.mu.Unlock()

		if exists {
			err := stream.Send(msg)
			if err != nil {
				fmt.Printf("Error sending message to stream: %v\n", err)
				s.CloseSession(userID)
				return
			}
		}
	}
}
