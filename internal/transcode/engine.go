package transcode

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Session represents an active FFmpeg transcoding session.
type Session struct {
	ID          string
	UserID      string
	ItemID      string
	StartedAt   time.Time
	LastPingAt  time.Time
	OutputDir   string
	VideoCodec  string
	AudioCodec  string
	done        chan struct{}
}

// Engine manages all active transcode sessions.
type Engine struct {
	dataDir  string
	logger   *zap.Logger
	mu       sync.Mutex
	sessions map[string]*Session
	hwAccel  HWAccelType
}

func NewEngine(dataDir string, logger *zap.Logger) *Engine {
	e := &Engine{
		dataDir:  dataDir,
		logger:   logger,
		sessions: make(map[string]*Session),
		hwAccel:  DetectBestHWAccel(),
	}
	go e.cleanupLoop()
	return e
}

// StartSession creates a new HLS transcode session and returns its ID.
func (e *Engine) StartSession(userID, itemID, filePath string, opts Options) (*Session, error) {
	sessionID := uuid.New().String()
	outDir := filepath.Join(e.dataDir, "transcode", sessionID)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("create transcode dir: %w", err)
	}

	sess := &Session{
		ID:         sessionID,
		UserID:     userID,
		ItemID:     itemID,
		StartedAt:  time.Now(),
		LastPingAt: time.Now(),
		OutputDir:  outDir,
		VideoCodec: opts.VideoCodec,
		AudioCodec: opts.AudioCodec,
		done:       make(chan struct{}),
	}

	job := TranscodeJob{
		InputPath:    filePath,
		OutputDir:    outDir,
		VideoCodec:   opts.VideoCodec,
		AudioCodec:   opts.AudioCodec,
		Width:        opts.Width,
		Height:       opts.Height,
		VideoBitrate: opts.VideoBitrate,
		AudioBitrate: opts.AudioBitrate,
		StartTimeSec: opts.StartTimeSec,
		HWAccel:      e.hwAccel,
	}

	runner := &FFmpegRunner{logger: e.logger}
	if err := runner.Start(job, sess.done); err != nil {
		os.RemoveAll(outDir)
		return nil, fmt.Errorf("start ffmpeg: %w", err)
	}

	e.mu.Lock()
	e.sessions[sessionID] = sess
	e.mu.Unlock()

	e.logger.Info("transcode session started",
		zap.String("session", sessionID),
		zap.String("item", itemID),
		zap.String("video_codec", opts.VideoCodec),
	)
	return sess, nil
}

// Ping marks a session as recently active (called on each segment request).
func (e *Engine) Ping(sessionID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if s, ok := e.sessions[sessionID]; ok {
		s.LastPingAt = time.Now()
	}
}

// GetSession returns a session by ID.
func (e *Engine) GetSession(sessionID string) (*Session, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	s, ok := e.sessions[sessionID]
	return s, ok
}

// KillSession terminates a session and removes temp files.
func (e *Engine) KillSession(sessionID string) {
	e.mu.Lock()
	sess, ok := e.sessions[sessionID]
	if ok {
		delete(e.sessions, sessionID)
	}
	e.mu.Unlock()
	if ok {
		close(sess.done)
		os.RemoveAll(sess.OutputDir)
		e.logger.Info("transcode session killed", zap.String("session", sessionID))
	}
}

// cleanupLoop kills sessions idle for more than 5 minutes.
func (e *Engine) cleanupLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		e.mu.Lock()
		for id, sess := range e.sessions {
			if time.Since(sess.LastPingAt) > 5*time.Minute {
				delete(e.sessions, id)
				close(sess.done)
				os.RemoveAll(sess.OutputDir)
				e.logger.Info("transcode session expired", zap.String("session", id))
			}
		}
		e.mu.Unlock()
	}
}

// Options controls what FFmpeg produces.
type Options struct {
	VideoCodec   string  // "copy" | "h264" | "hevc"
	AudioCodec   string  // "copy" | "aac"
	Width        int
	Height       int
	VideoBitrate string
	AudioBitrate string
	StartTimeSec float64
}
