package mastery

import (
	"math"
	"testing"
)

const tolerance = 1e-6

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < tolerance
}

func TestMasteredGradeFromNeutral(t *testing.T) {
	start := ConceptState{Confidence: 0.5, LastReviewedAt: nil}
	now := int64(1752835200) // fixed timestamp

	result := UpdateFromFlashcard(start, Mastered, now)

	if result.Confidence <= 0.5 {
		t.Errorf("expected confidence to increase from 0.5, got %f", result.Confidence)
	}
	if result.Confidence > 0.5+maxDailyDelta {
		t.Errorf("confidence %f exceeded max daily delta from 0.5", result.Confidence)
	}
	if result.LastReviewedAt == nil || *result.LastReviewedAt != now {
		t.Errorf("expected LastReviewedAt to be set to %d", now)
	}
}

func TestQuizMissHighConfidenceAnomalyTolerance(t *testing.T) {
	start := ConceptState{Confidence: 0.8, LastReviewedAt: nil}
	now := int64(1752835200)

	// Quiz miss at confidence > 0.75 should use likelihood 0.45
	result := UpdateFromQuiz(start, false, now)

	// Compute expected with anomaly tolerance likelihood 0.45
	likelihood := 0.45
	posterior := bayesianUpdate(0.8, likelihood)
	blended := applyInertia(0.8, posterior)
	capped := capDelta(0.8, blended)
	expected := clamp(capped, MinConfidence, MaxConfidence)

	if !almostEqual(result.Confidence, expected) {
		t.Errorf("anomaly tolerance: got %f, expected %f", result.Confidence, expected)
	}
}

func TestQuizMissLowConfidence(t *testing.T) {
	start := ConceptState{Confidence: 0.5, LastReviewedAt: nil}
	now := int64(1752835200)

	// Quiz miss at confidence <= 0.75 should use likelihood 0.3
	result := UpdateFromQuiz(start, false, now)

	likelihood := 0.3
	posterior := bayesianUpdate(0.5, likelihood)
	blended := applyInertia(0.5, posterior)
	capped := capDelta(0.5, blended)
	expected := clamp(capped, MinConfidence, MaxConfidence)

	if !almostEqual(result.Confidence, expected) {
		t.Errorf("low confidence miss: got %f, expected %f", result.Confidence, expected)
	}
}

func TestDecayAfterMultiDayGap(t *testing.T) {
	lastReview := int64(1752835200)
	start := ConceptState{Confidence: 0.7, LastReviewedAt: &lastReview}
	now := lastReview + 7*24*60*60 // 7 days later

	result := applyDecay(start, now)

	if result.Confidence >= 0.7 {
		t.Errorf("expected decay, got %f >= 0.7", result.Confidence)
	}
	if result.Confidence <= 0.0 {
		t.Errorf("confidence should not decay to %f", result.Confidence)
	}
}

func TestDecaySlowerAbovePoint8(t *testing.T) {
	lastReview := int64(1752835200)
	now := lastReview + 10*24*60*60 // 10 days

	// Below 0.8 — normal decay
	low := ConceptState{Confidence: 0.7, LastReviewedAt: &lastReview}
	lowResult := applyDecay(low, now)

	// Above 0.8 — slower decay (exponent halved)
	high := ConceptState{Confidence: 0.9, LastReviewedAt: &lastReview}
	highResult := applyDecay(high, now)

	lowDrop := 0.7 - lowResult.Confidence
	highDrop := 0.9 - highResult.Confidence

	if highDrop > lowDrop {
		t.Errorf("decay above 0.8 should be slower: high dropped %f, low dropped %f", highDrop, lowDrop)
	}
}

func TestConfidenceBounds(t *testing.T) {
	now := int64(1752835200)

	// Start at min, grade Mastered many times
	state := ConceptState{Confidence: MinConfidence, LastReviewedAt: nil}
	for i := 0; i < 100; i++ {
		state = UpdateFromFlashcard(state, Mastered, now+int64(i)*86400)
	}
	if state.Confidence > MaxConfidence {
		t.Errorf("confidence %f exceeded max %f", state.Confidence, MaxConfidence)
	}

	// Start at max, grade Unknown many times
	state = ConceptState{Confidence: MaxConfidence, LastReviewedAt: nil}
	for i := 0; i < 100; i++ {
		state = UpdateFromFlashcard(state, Unknown, now+int64(i)*86400)
	}
	if state.Confidence < MinConfidence {
		t.Errorf("confidence %f below min %f", state.Confidence, MinConfidence)
	}
}

func TestFixtureSequence(t *testing.T) {
	t0 := int64(1752835200)
	day := int64(86400)

	state := ConceptState{Confidence: 0.5, LastReviewedAt: nil}

	// Event 1: Mastered flashcard at t0
	state = UpdateFromFlashcard(state, Mastered, t0)

	// Event 2: Correct quiz at t0+1day
	state = UpdateFromQuiz(state, true, t0+day)

	// Event 3: KnownFairly flashcard at t0+2days
	state = UpdateFromFlashcard(state, KnownFairly, t0+2*day)

	// Event 4: Quiz miss at t0+5days (confidence likely > 0.75, anomaly tolerance)
	state = UpdateFromQuiz(state, false, t0+5*day)

	// Event 5: KnownWell flashcard at t0+6days
	state = UpdateFromFlashcard(state, KnownWell, t0+6*day)

	// Verify confidence is within valid range
	if state.Confidence < MinConfidence || state.Confidence > MaxConfidence {
		t.Errorf("final confidence %f out of range [%f, %f]", state.Confidence, MinConfidence, MaxConfidence)
	}

	// Verify it moved from neutral
	if almostEqual(state.Confidence, 0.5) {
		t.Errorf("confidence should have moved from 0.5, still at %f", state.Confidence)
	}

	// Verify timestamp was updated
	if state.LastReviewedAt == nil || *state.LastReviewedAt != t0+6*day {
		t.Errorf("expected LastReviewedAt = %d, got %v", t0+6*day, state.LastReviewedAt)
	}
}
