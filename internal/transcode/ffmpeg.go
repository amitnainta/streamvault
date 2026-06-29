package transcode

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

// TranscodeJob describes a single FFmpeg invocation.
type TranscodeJob struct {
	InputPath    string
	OutputDir    string
	VideoCodec   string
	AudioCodec   string
	Width        int
	Height       int
	VideoBitrate string
	AudioBitrate string
	StartTimeSec float64
	HWAccel      HWAccelType
	SubtitlePath string // burn-in subtitle path; empty = no burn-in
}

// FFmpegRunner wraps the FFmpeg subprocess.
type FFmpegRunner struct {
	logger *zap.Logger
}

// Start launches FFmpeg and returns immediately. The done channel is closed
// when either the session is killed or FFmpeg exits naturally.
func (f *FFmpegRunner) Start(job TranscodeJob, done <-chan struct{}) error {
	args := f.buildArgs(job)
	cmd := exec.Command("ffmpeg", args...)
	stderr := &strings.Builder{}
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("ffmpeg start: %w", err)
	}

	// Watch for kill signal or natural exit
	go func() {
		exitCh := make(chan error, 1)
		go func() { exitCh <- cmd.Wait() }()
		select {
		case <-done:
			cmd.Process.Kill()
		case err := <-exitCh:
			if err != nil {
				f.logger.Warn("ffmpeg exited with error",
					zap.Error(err),
					zap.String("stderr", stderr.String()),
				)
			}
		}
	}()

	return nil
}

func (f *FFmpegRunner) buildArgs(job TranscodeJob) []string {
	args := []string{
		"-hide_banner", "-loglevel", "warning",
		"-ss", strconv.FormatFloat(job.StartTimeSec, 'f', 3, 64),
		"-i", job.InputPath,
	}

	// VideoCodec is already the resolved encoder name (e.g. "h264_nvenc", "libx264")
	videoEnc := job.VideoCodec
	args = append(args, "-c:v", videoEnc)
	if job.VideoBitrate != "" && videoEnc != "copy" {
		args = append(args, "-b:v", job.VideoBitrate)
	}
	if job.Width > 0 && job.Height > 0 && videoEnc != "copy" {
		args = append(args, "-vf", fmt.Sprintf("scale=%d:%d", job.Width, job.Height))
	}

	// Audio codec
	audioEnc := job.AudioCodec
	if audioEnc != "copy" {
		audioEnc = "aac"
	}
	args = append(args, "-c:a", audioEnc)
	if job.AudioBitrate != "" && audioEnc != "copy" {
		args = append(args, "-b:a", job.AudioBitrate)
	}

	// HLS output
	args = append(args,
		"-f", "hls",
		"-hls_time", "6",
		"-hls_list_size", "0",
		"-hls_flags", "independent_segments",
		"-hls_segment_filename", filepath.Join(job.OutputDir, "seg%05d.ts"),
		filepath.Join(job.OutputDir, "index.m3u8"),
	)

	return args
}
