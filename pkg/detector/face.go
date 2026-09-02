package detector

import (
	_ "embed"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	pigo "github.com/esimov/pigo/core"
)

//go:embed cascade/facefinder
var faceFinderCascade []byte

var (
	classifierOnce sync.Once
	classifier     *pigo.Pigo
	classifierErr  error
)

// InitFaceClassifier initializes and unpacks the embedded Pigo cascade classifier.
func InitFaceClassifier() (*pigo.Pigo, error) {
	classifierOnce.Do(func() {
		if len(faceFinderCascade) == 0 {
			classifierErr = fmt.Errorf("embedded facefinder cascade data is empty")
			return
		}
		p := pigo.NewPigo()
		c, err := p.Unpack(faceFinderCascade)
		if err != nil {
			classifierErr = fmt.Errorf("failed unpacking facefinder cascade: %w", err)
			return
		}
		classifier = c
	})
	return classifier, classifierErr
}

// FaceDetection represents a detected face within a frame.
type FaceDetection struct {
	TimeSec float64 `json:"time_sec"`
	NormX   float64 `json:"norm_x"` // 0.0 (left) to 1.0 (right)
	NormY   float64 `json:"norm_y"` // 0.0 (top) to 1.0 (bottom)
	Scale   float64 `json:"scale"`
	Score   float32 `json:"score"`
}

// CameraTransition represents a smooth pan from one horizontal focal point to another.
type CameraTransition struct {
	StartTimeSec float64
	DeltaX       float64
	DurationSec  float64
}

// FaceTracker manages sampling frames from video, running face detection, and building dynamic crop filter expressions.
type FaceTracker struct {
	FFmpegPath      string
	SampleFPS       float64 // default: 1.0 frame per second
	MinMoveDistance float64 // threshold to ignore small head tilts (default: 0.08 = 8% width)
	MinHoldDuration float64 // minimum time in seconds before allowing another camera shift (default: 1.5s)
	PanDuration     float64 // duration of the camera pan interpolation (default: 0.8s)
}

// NewFaceTracker creates a new FaceTracker instance with optimal defaults.
func NewFaceTracker(ffmpegPath string) *FaceTracker {
	if ffmpegPath == "" {
		if p, err := exec.LookPath("ffmpeg"); err == nil {
			ffmpegPath = p
		}
	}
	return &FaceTracker{
		FFmpegPath:      ffmpegPath,
		SampleFPS:       1.0,
		MinMoveDistance: 0.08,
		MinHoldDuration: 1.5,
		PanDuration:     0.8,
	}
}

// TrackFacesInSegment extracts sample frames from [startSec, startSec+durationSec], detects faces,
// and produces a smooth dynamic FFmpeg crop expression for vertical 9:16 Shorts.
func (ft *FaceTracker) TrackFacesInSegment(inputFile string, startSec, durationSec float64) (string, error) {
	cls, err := InitFaceClassifier()
	if err != nil {
		return DefaultCenterCropFilter(), fmt.Errorf("face classifier init failed: %w", err)
	}

	if durationSec <= 0 {
		return DefaultCenterCropFilter(), nil
	}

	tmpDir, err := os.MkdirTemp("", "clipper_face_*")
	if err != nil {
		return DefaultCenterCropFilter(), fmt.Errorf("failed to create temp dir for face frames: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Extract frames at 1 fps scaled down to 480px width for ultra-fast processing
	outPattern := filepath.Join(tmpDir, "frame_%04d.jpg")
	args := []string{
		"-y",
		"-ss", fmt.Sprintf("%.3f", startSec),
		"-t", fmt.Sprintf("%.3f", durationSec),
		"-i", inputFile,
		"-vf", fmt.Sprintf("fps=%.2f,scale=480:-2", ft.SampleFPS),
		"-q:v", "3",
		outPattern,
	}

	cmd := exec.Command(ft.FFmpegPath, args...)
	if err := cmd.Run(); err != nil {
		// If frame extraction fails, fallback safely to center crop
		return DefaultCenterCropFilter(), fmt.Errorf("frame extraction failed: %w", err)
	}

	files, err := filepath.Glob(filepath.Join(tmpDir, "frame_*.jpg"))
	if err != nil || len(files) == 0 {
		return DefaultCenterCropFilter(), nil
	}
	sort.Strings(files)

	var detections []FaceDetection

	for i, fPath := range files {
		timeOffset := float64(i) / ft.SampleFPS
		det, found := detectPrimaryFace(cls, fPath)
		if found {
			det.TimeSec = timeOffset
			detections = append(detections, det)
		}
	}

	if len(detections) == 0 {
		// No faces found in this segment (e.g. landscape, gaming, scenery) -> Fallback to center-crop
		return DefaultCenterCropFilter(), nil
	}

	// Interpolate and smooth face positions over the timeline
	transitions, baseX := ft.PlanCameraTransitions(detections, durationSec)

	return BuildDynamicCropFilter(baseX, transitions), nil
}

// detectPrimaryFace detects faces in an image file and returns the most prominent face.
func detectPrimaryFace(cls *pigo.Pigo, imgPath string) (FaceDetection, bool) {
	f, err := os.Open(imgPath)
	if err != nil {
		return FaceDetection{}, false
	}
	defer f.Close()

	src, _, err := image.Decode(f)
	if err != nil {
		return FaceDetection{}, false
	}

	bounds := src.Bounds()
	cols, rows := bounds.Dx(), bounds.Dy()
	if cols <= 0 || rows <= 0 {
		return FaceDetection{}, false
	}

	pixels := pigo.RgbToGrayscale(src)

	cParams := pigo.CascadeParams{
		MinSize:     20,
		MaxSize:     int(math.Min(float64(cols), float64(rows))),
		ShiftFactor: 0.1,
		ScaleFactor: 1.1,
		ImageParams: pigo.ImageParams{
			Pixels: pixels,
			Rows:   rows,
			Cols:   cols,
			Dim:    cols,
		},
	}

	dets := cls.RunCascade(cParams, 0.0)
	dets = cls.ClusterDetections(dets, 0.2)
	if len(dets) == 0 {
		return FaceDetection{}, false
	}

	// Pick the most prominent face (highest scale * quality score)
	sort.Slice(dets, func(i, j int) bool {
		scoreI := float64(dets[i].Scale) * float64(dets[i].Q)
		scoreJ := float64(dets[j].Scale) * float64(dets[j].Q)
		return scoreI > scoreJ
	})

	primary := dets[0]
	normX := float64(primary.Col) / float64(cols)
	normY := float64(primary.Row) / float64(rows)

	// Clamp normalized values to [0.0, 1.0]
	if normX < 0.0 {
		normX = 0.0
	} else if normX > 1.0 {
		normX = 1.0
	}
	if normY < 0.0 {
		normY = 0.0
	} else if normY > 1.0 {
		normY = 1.0
	}

	return FaceDetection{
		NormX: normX,
		NormY: normY,
		Scale: float64(primary.Scale),
		Score: primary.Q,
	}, true
}

// PlanCameraTransitions filters detections and constructs a list of smooth camera pan transitions.
func (ft *FaceTracker) PlanCameraTransitions(dets []FaceDetection, totalDuration float64) ([]CameraTransition, float64) {
	if len(dets) == 0 {
		return nil, 0.5
	}

	baseX := dets[0].NormX
	currentX := baseX
	lastTransitionTime := 0.0

	var transitions []CameraTransition

	for _, d := range dets[1:] {
		dist := math.Abs(d.NormX - currentX)
		timeSinceLast := d.TimeSec - lastTransitionTime

		// Check deadband threshold and minimum hold duration to prevent jittery camera cuts
		if dist >= ft.MinMoveDistance && timeSinceLast >= ft.MinHoldDuration {
			delta := d.NormX - currentX
			transitions = append(transitions, CameraTransition{
				StartTimeSec: d.TimeSec,
				DeltaX:       delta,
				DurationSec:  ft.PanDuration,
			})
			currentX = d.NormX
			lastTransitionTime = d.TimeSec
		}
	}

	return transitions, baseX
}

// BuildDynamicCropFilter generates an FFmpeg filter expression that crops to 9:16 and pans horizontally.
func BuildDynamicCropFilter(baseX float64, transitions []CameraTransition) string {
	if len(transitions) == 0 {
		// Static face crop centered at baseX
		return fmt.Sprintf("crop=w='ih*(9/16)':h='ih':x='clip(%.4f*iw - 0.5*ow, 0, iw-ow)':y=0,scale=1080:1920:flags=lanczos", baseX)
	}

	// Build smooth dynamic formula:
	// x = clip((baseX + delta1*clip((t-t1)/dur1, 0, 1) + delta2*clip((t-t2)/dur2, 0, 1)...)*iw - 0.5*ow, 0, iw-ow)
	var terms []string
	terms = append(terms, fmt.Sprintf("%.4f", baseX))

	for _, tr := range transitions {
		sign := "+"
		delta := tr.DeltaX
		if delta < 0 {
			sign = "-"
			delta = -delta
		}
		dur := tr.DurationSec
		if dur <= 0 {
			dur = 0.8
		}
		term := fmt.Sprintf("%s %.4f*clip((t-%.2f)/%.2f, 0, 1)", sign, delta, tr.StartTimeSec, dur)
		terms = append(terms, term)
	}

	posExpr := strings.Join(terms, " ")
	xExpr := fmt.Sprintf("clip((%s)*iw - 0.5*ow, 0, iw-ow)", posExpr)

	return fmt.Sprintf("crop=w='ih*(9/16)':h='ih':x='%s':y=0,scale=1080:1920:flags=lanczos", xExpr)
}

// DefaultCenterCropFilter returns the default center-crop filter for vertical 9:16 video.
func DefaultCenterCropFilter() string {
	return "crop=w='ih*(9/16)':h='ih':x='(iw-ow)/2':y=0,scale=1080:1920:flags=lanczos"
}
