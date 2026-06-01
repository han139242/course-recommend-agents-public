package service

import (
	"math"
	"testing"
)

func TestHitAtK(t *testing.T) {
	if h := ComputeHitAtK(nil, nil, 5); h != 0 {
		t.Errorf("nil: got %.4f", h)
	}
	if h := ComputeHitAtK([]string{"a", "b"}, []string{"a"}, 1); h != 1.0 {
		t.Errorf("hit@1: got %.4f", h)
	}
	if h := ComputeHitAtK([]string{"c", "a"}, []string{"a"}, 1); h != 0.0 {
		t.Errorf("miss@1: got %.4f", h)
	}
}

func TestNDCGAtK(t *testing.T) {
	if n := ComputeNDCGAtK(nil, nil, 5); n != 0 {
		t.Errorf("nil: got %.4f", n)
	}
	rec := []string{"a", "b", "c"}
	truth := []string{"a", "b"}
	n := ComputeNDCGAtK(rec, truth, 5)
	if n < 0.9 || n > 1.01 {
		t.Errorf("near-perfect: got %.4f", n)
	}
}

func TestMRR(t *testing.T) {
	if m := ComputeMRR(nil, nil); m != 0 {
		t.Errorf("nil: got %.4f", m)
	}
	if m := ComputeMRR([]string{"c", "a", "b"}, []string{"a"}); math.Abs(m-0.5) > 0.001 {
		t.Errorf("rank 2: got %.4f", m)
	}
	if m := ComputeMRR([]string{"a"}, []string{"z"}); m != 0 {
		t.Errorf("miss: got %.4f", m)
	}
}

func TestPrerequisiteViolation(t *testing.T) {
	if v := ComputePrerequisiteViolationRate(nil, nil); v != 0 {
		t.Errorf("nil: got %.4f", v)
	}
	if v := ComputePrerequisiteViolationRate([]string{"a"}, nil); v != 0 {
		t.Errorf("no edges: got %.4f", v)
	}
}

func TestDifficultySmoothness(t *testing.T) {
	if s := ComputeDifficultySmoothness(nil, nil); s != 1.0 {
		t.Errorf("nil: got %.4f", s)
	}
	if s := ComputeDifficultySmoothness([]string{"a"}, nil); s != 1.0 {
		t.Errorf("short: got %.4f", s)
	}
}
