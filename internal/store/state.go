package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type State struct {
	Read      map[int]bool `json:"read"`
	ReadLater map[int]bool `json:"read_later"`
}

type StateStore struct {
	mu   sync.RWMutex
	path string
	data State
}

func NewStateStore(path string) (*StateStore, error) {
	s := &StateStore{path: path}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *StateStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.data = State{Read: map[int]bool{}, ReadLater: map[int]bool{}}
			return nil
		}
		return fmt.Errorf("read state: %w", err)
	}

	var state State
	if err := json.Unmarshal(b, &state); err != nil {
		return fmt.Errorf("unmarshal state: %w", err)
	}
	if state.Read == nil {
		state.Read = map[int]bool{}
	}
	if state.ReadLater == nil {
		state.ReadLater = map[int]bool{}
	}
	s.data = state
	return nil
}

func (s *StateStore) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("mkdir state dir: %w", err)
	}
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return fmt.Errorf("write state tmp: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("rename state: %w", err)
	}
	return nil
}

func (s *StateStore) MarkRead(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Read[id] = true
	return s.saveLocked()
}

func (s *StateStore) IsRead(id int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.Read[id]
}

func (s *StateStore) ToggleReadLater(id int) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := !s.data.ReadLater[id]
	s.data.ReadLater[id] = next
	if !next {
		delete(s.data.ReadLater, id)
	}
	if err := s.saveLocked(); err != nil {
		return false, err
	}
	return next, nil
}

func (s *StateStore) IsReadLater(id int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.ReadLater[id]
}
