package snake

import "fmt"

type Scoring struct {
	level          int
	maxLevel       int
	increaseLevel  bool
	endOnMaxLevel  bool
	total          int
	pointsPerLevel int
}

func NewScoring(level, maxLevel, pointsPerLevel int, increaseLevel, endOnMaxLevel bool) (*Scoring, error) {
	s := &Scoring{
		level:          level,
		maxLevel:       maxLevel,
		increaseLevel:  increaseLevel,
		endOnMaxLevel:  endOnMaxLevel,
		pointsPerLevel: pointsPerLevel,
		total:          0,
	}
	return s, s.validate()
}

func (s *Scoring) validate() error {
	if s.level <= 0 {
		return fmt.Errorf("invalid level '%d'", s.level)
	}
	if s.maxLevel <= 0 {
		return fmt.Errorf("invalid max level '%d'", s.level)
	}
	if s.total < 0 {
		return fmt.Errorf("invalid total '%d'", s.total)
	}
	if s.pointsPerLevel <= 0 {
		return fmt.Errorf("invalid points per level '%d'", s.pointsPerLevel)
	}
	if s.level > s.maxLevel {
		return fmt.Errorf("level '%d' cannot exceed max level '%d'", s.level, s.maxLevel)
	}
	return nil
}

func (s *Scoring) Level() int {
	return s.level
}

func (s *Scoring) Total() int {
	return s.total
}

func (s *Scoring) PointsPerLevel() int {
	return s.pointsPerLevel
}

func (s *Scoring) AddPoints(points int) {
	s.total += points
	if s.increaseLevel {
		s.checkLevelUp()
	}
}

func (s *Scoring) checkLevelUp() {
	newLevel := (s.total / s.pointsPerLevel) + 1
	if newLevel > s.level {
		s.level = min(newLevel, s.maxLevel)
	}
}

func (s *Scoring) IsMaxLevel() bool {
	return s.level >= s.maxLevel
}
