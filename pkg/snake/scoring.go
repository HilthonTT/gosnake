package snake

import "fmt"

const (
	// MaxLevel is the highest level any mode can reach.
	MaxLevel = 10

	// PointsPerFood is awarded every time a snake eats.
	PointsPerFood = 10

	// PointsPerLevel is how many points it takes to advance one level.
	PointsPerLevel = 100
)

type Scoring struct {
	startLevel     int
	level          int
	maxLevel       int
	increaseLevel  bool
	total          int
	pointsPerLevel int
}

func NewScoring(level, maxLevel, pointsPerLevel int, increaseLevel bool) (*Scoring, error) {
	s := &Scoring{
		startLevel:     level,
		level:          level,
		maxLevel:       maxLevel,
		increaseLevel:  increaseLevel,
		pointsPerLevel: pointsPerLevel,
	}
	return s, s.validate()
}

// NewDefaultScoring returns the scoring rules shared by every mode, starting
// at the given level.
func NewDefaultScoring(level int) (*Scoring, error) {
	return NewScoring(level, MaxLevel, PointsPerLevel, true)
}

func (s *Scoring) validate() error {
	if s.level <= 0 {
		return fmt.Errorf("invalid level '%d'", s.level)
	}
	if s.maxLevel <= 0 {
		return fmt.Errorf("invalid max level '%d'", s.maxLevel)
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
	// Levels are earned relative to the starting level, so picking a higher
	// starting level doesn't freeze progression until the score catches up.
	newLevel := s.startLevel + s.total/s.pointsPerLevel
	if newLevel > s.level {
		s.level = min(newLevel, s.maxLevel)
	}
}

func (s *Scoring) IsMaxLevel() bool {
	return s.level >= s.maxLevel
}
