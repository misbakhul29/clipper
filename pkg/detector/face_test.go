package detector

import (
	"math"
	"strings"
	"testing"
)

func TestInitFaceClassifier(t *testing.T) {
	cls, err := InitFaceClassifier()
	if err != nil {
		t.Fatalf("expected classifier to unpack without error, got: %v", err)
	}
	if cls == nil {
		t.Fatalf("expected classifier to be non-nil")
	}
}

func TestPlanCameraTransitions(t *testing.T) {
	ft := NewFaceTracker("")
	ft.MinMoveDistance = 0.08
	ft.MinHoldDuration = 1.5
	ft.PanDuration = 0.8

	// Detections:
	// t=0: face at 0.5 (center)
	// t=1: small tilt to 0.53 (diff = 0.03 < 0.08, should be ignored)
	// t=2: speaker turns/moves to 0.70 (diff = 0.20 >= 0.08, and time=2s >= 1.5s, should trigger pan)
	// t=2.5: sudden blip to 0.40 (time diff = 0.5s < 1.5s, should be ignored due to hold duration)
	// t=4.0: switch to left speaker at 0.25 (diff = 0.45 >= 0.08, time=2.0s >= 1.5s, should trigger pan)

	dets := []FaceDetection{
		{TimeSec: 0.0, NormX: 0.50},
		{TimeSec: 1.0, NormX: 0.53},
		{TimeSec: 2.0, NormX: 0.70},
		{TimeSec: 2.5, NormX: 0.40},
		{TimeSec: 4.0, NormX: 0.25},
	}

	transitions, baseX := ft.PlanCameraTransitions(dets, 5.0)

	if baseX != 0.50 {
		t.Errorf("expected baseX=0.50, got %.2f", baseX)
	}

	if len(transitions) != 2 {
		t.Fatalf("expected exactly 2 transitions, got %d", len(transitions))
	}

	// First transition at t=2.0, delta = +0.20
	if transitions[0].StartTimeSec != 2.0 || math.Abs(transitions[0].DeltaX-0.20) > 1e-4 {
		t.Errorf("transition 0 unexpected: %+v", transitions[0])
	}

	// Second transition at t=4.0, delta = 0.25 - 0.70 = -0.45
	if transitions[1].StartTimeSec != 4.0 || math.Abs(transitions[1].DeltaX-(-0.45)) > 1e-4 {
		t.Errorf("transition 1 unexpected: %+v", transitions[1])
	}
}

func TestBuildDynamicCropFilter(t *testing.T) {
	// Case 1: No transitions (static face)
	staticFilter := BuildDynamicCropFilter(0.35, nil)
	if !strings.Contains(staticFilter, "clip(0.3500*iw - 0.5*ow, 0, iw-ow)") {
		t.Errorf("unexpected static filter: %s", staticFilter)
	}

	// Case 2: Dynamic transitions
	transitions := []CameraTransition{
		{StartTimeSec: 2.0, DeltaX: 0.20, DurationSec: 0.8},
		{StartTimeSec: 4.5, DeltaX: -0.30, DurationSec: 0.8},
	}
	dynamicFilter := BuildDynamicCropFilter(0.50, transitions)

	if !strings.Contains(dynamicFilter, "0.5000 + 0.2000*clip((t-2.00)/0.80, 0, 1) - 0.3000*clip((t-4.50)/0.80, 0, 1)") {
		t.Errorf("unexpected dynamic filter: %s", dynamicFilter)
	}
	if !strings.Contains(dynamicFilter, "scale=1080:1920:flags=lanczos") {
		t.Errorf("dynamic filter missing scale: %s", dynamicFilter)
	}
}

func TestDefaultCenterCropFilter(t *testing.T) {
	def := DefaultCenterCropFilter()
	if def != "crop=w='ih*(9/16)':h='ih':x='(iw-ow)/2':y=0,scale=1080:1920:flags=lanczos" {
		t.Errorf("unexpected default filter: %s", def)
	}
}
