package main

import "sync"

type score struct {
	sync.RWMutex
	score    int64
	sucesses int32
	fails    int32
}

func NewScore() *score {
	s := &score{
		score:    0,
		sucesses: 0,
		fails:    0,
	}
	return s
}

func (s *score) GetScore() int64 {
	s.RLock()
	score := s.score
	s.RUnlock()
	return score
}

func (s *score) GetSucesses() int32 {
	s.RLock()
	sucesses := s.sucesses
	s.RUnlock()
	return sucesses
}

func (s *score) GetFails() int32 {
	s.RLock()
	fails := s.fails
	s.RUnlock()
	return fails
}

func (s *score) SetScore(score int64) {
	s.Lock()
	s.score += int64(score)
	s.sucesses += 1
	s.Unlock()
}

func (s *score) SetFails() {
	s.Lock()
	s.fails += 1
	s.Unlock()
}
