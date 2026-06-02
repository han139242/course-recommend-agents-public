package repository

import (
	"testing"
)

func TestEnabledWithNilClient(t *testing.T) {
	repo := NewRedisRepository(nil)
	if repo.Enabled() {
		t.Fatal("expected disabled redis repository")
	}
}

func TestEvidenceAdjustedMasteryRequiresEnoughAnswers(t *testing.T) {
	score := evidenceAdjustedMastery(3, 3)
	if score >= 80 {
		t.Fatalf("expected three correct answers to remain below solid mastery, got %.2f", score)
	}

	fullEvidenceScore := evidenceAdjustedMastery(8, 8)
	if fullEvidenceScore != 100 {
		t.Fatalf("expected enough perfect evidence to reach 100, got %.2f", fullEvidenceScore)
	}
}

func TestEvidenceAdjustedMasteryCombinesCorrectRateAndConfidence(t *testing.T) {
	score := evidenceAdjustedMastery(6, 8)
	if score != 75 {
		t.Fatalf("expected full-evidence score to follow correct rate, got %.2f", score)
	}

	lowEvidenceScore := evidenceAdjustedMastery(1, 1)
	if lowEvidenceScore <= 0 || lowEvidenceScore >= 50 {
		t.Fatalf("expected low-evidence correct answer to be positive but cautious, got %.2f", lowEvidenceScore)
	}
}
