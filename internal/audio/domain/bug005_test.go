package domain

import "testing"

func TestSamplesMonoReturnsDetachedSlice(t *testing.T) {
	s := Samples{Channels: 1, Values: []float32{1, 2, 3}}
	got := s.Mono()
	s.Values[0] = 99
	if got[0] == 99 {
		t.Fatal("Mono returned aliased slice for mono input")
	}
}

func TestNormalizeDoesNotChangeMonoSnapshot(t *testing.T) {
	s := Samples{Rate: 8000, Channels: 1, Values: []float32{.5, .5}}
	snapshot := s.Mono()
	s.Normalize()
	if snapshot[0] == s.Values[0] && snapshot[0] != .5 {
		t.Fatal("Normalize changed the previously returned mono snapshot")
	}
}

func TestSamplesMonoStereoReturnsDetachedSlice(t *testing.T) {
	s := Samples{Rate: 8000, Channels: 2, Values: []float32{.2, .4}}
	got := s.Mono()
	if len(got) != 1 || got[0] < .299 || got[0] > .301 {
		t.Fatalf("stereo mono = %v, want average 0.3", got)
	}
	s.Values[0] = 99
	if got[0] == 99 {
		t.Fatal("stereo Mono returned aliased slice")
	}
}
