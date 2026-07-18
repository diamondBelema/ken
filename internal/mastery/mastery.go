package mastery

import "math"

const (
	MinConfidence   = 0.05
	MaxConfidence   = 0.995
	decayRatePerDay = 0.95
	inertia         = 0.8
	maxDailyDelta   = 0.08
)

type ConfidenceLevel int

const (
	Unknown ConfidenceLevel = iota
	KnownLittle
	KnownFairly
	KnownWell
	Mastered
)

type ConceptState struct {
	Confidence     float64
	LastReviewedAt *int64
}

func applyDecay(c ConceptState, now int64) ConceptState {
	if c.LastReviewedAt == nil {
		return c
	}
	days := float64(now-*c.LastReviewedAt) / (60 * 60 * 24)
	if days <= 0 {
		return c
	}
	rate := decayRatePerDay
	exponent := days
	if c.Confidence > 0.8 {
		exponent = days * 0.5
	}
	decayed := c.Confidence * math.Pow(rate, exponent)
	c.Confidence = clamp(decayed, MinConfidence, MaxConfidence)
	return c
}

func bayesianUpdate(prior, likelihood float64) float64 {
	numerator := prior * likelihood
	denominator := numerator + (1-prior)*(1-likelihood)
	return numerator / denominator
}

func applyInertia(prior, posterior float64) float64 {
	return prior*inertia + posterior*(1-inertia)
}

func capDelta(prior, updated float64) float64 {
	delta := clamp(updated-prior, -maxDailyDelta, maxDailyDelta)
	return prior + delta
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func LikelihoodForFlashcardGrade(level ConfidenceLevel) float64 {
	switch level {
	case Unknown:
		return 0.15
	case KnownLittle:
		return 0.35
	case KnownFairly:
		return 0.6
	case KnownWell:
		return 0.8
	case Mastered:
		return 0.95
	}
	return 0.15
}

func UpdateFromFlashcard(c ConceptState, grade ConfidenceLevel, now int64) ConceptState {
	decayed := applyDecay(c, now)
	likelihood := LikelihoodForFlashcardGrade(grade)
	posterior := bayesianUpdate(decayed.Confidence, likelihood)
	blended := applyInertia(decayed.Confidence, posterior)
	capped := capDelta(decayed.Confidence, blended)
	decayed.Confidence = clamp(capped, MinConfidence, MaxConfidence)
	decayed.LastReviewedAt = &now
	return decayed
}

func UpdateFromQuiz(c ConceptState, wasCorrect bool, now int64) ConceptState {
	decayed := applyDecay(c, now)
	var likelihood float64
	switch {
	case wasCorrect:
		likelihood = 0.95
	case decayed.Confidence > 0.75:
		likelihood = 0.45
	default:
		likelihood = 0.3
	}
	posterior := bayesianUpdate(decayed.Confidence, likelihood)
	blended := applyInertia(decayed.Confidence, posterior)
	capped := capDelta(decayed.Confidence, blended)
	decayed.Confidence = clamp(capped, MinConfidence, MaxConfidence)
	decayed.LastReviewedAt = &now
	return decayed
}
