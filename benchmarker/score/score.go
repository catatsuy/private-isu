package score

import "sync"

type Score struct {
	sync.RWMutex
	score    int64
	sucesses int64
	fails    int64
}

var instance *Score
var once sync.Once

func GetInstance() *Score {
	once.Do(func() {
		instance = &Score{
			score:    0,
			sucesses: 0,
			fails:    0,
		}
	})

	return instance
}

func (s *Score) GetScore() int64 {
	s.RLock()
	score := s.score
	s.RUnlock()
	return score
}

func (s *Score) GetSucesses() int64 {
	s.RLock()
	sucesses := s.sucesses
	s.RUnlock()
	return sucesses
}

func (s *Score) GetFails() int64 {
	s.RLock()
	fails := s.fails
	s.RUnlock()
	return fails
}

func (s *Score) SetScore(score int64) {
	s.Lock()
	s.score += int64(score)
	s.sucesses += 1
	s.Unlock()
}

func (s *Score) SetFails() {
	s.Lock()
	s.fails += 1
	s.Unlock()
}
