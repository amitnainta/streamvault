package transcode

import (
	"os/exec"
	"strings"
)

// HWAccelType identifies the hardware acceleration method.
type HWAccelType string

const (
	HWAccelNone  HWAccelType = "none"
	HWAccelNVENC HWAccelType = "nvenc"
	HWAccelQSV   HWAccelType = "qsv"
	HWAccelAMF   HWAccelType = "amf"
	HWAccelVAAPI HWAccelType = "vaapi"
)

// DetectBestHWAccel probes FFmpeg for available hardware encoders.
// Returns the best available option in priority order.
func DetectBestHWAccel() HWAccelType {
	out, err := exec.Command("ffmpeg", "-hide_banner", "-encoders").Output()
	if err != nil {
		return HWAccelNone
	}
	encoders := string(out)

	switch {
	case strings.Contains(encoders, "h264_nvenc"):
		return HWAccelNVENC
	case strings.Contains(encoders, "h264_qsv"):
		return HWAccelQSV
	case strings.Contains(encoders, "h264_amf"):
		return HWAccelAMF
	case strings.Contains(encoders, "h264_vaapi"):
		return HWAccelVAAPI
	default:
		return HWAccelNone
	}
}

// VideoEncoder returns the FFmpeg encoder name for the given HW accel type.
func VideoEncoder(hw HWAccelType, codec string) string {
	if codec == "copy" {
		return "copy"
	}
	switch hw {
	case HWAccelNVENC:
		return codec + "_nvenc"
	case HWAccelQSV:
		return codec + "_qsv"
	case HWAccelAMF:
		return codec + "_amf"
	case HWAccelVAAPI:
		return codec + "_vaapi"
	default:
		// Software fallback
		if codec == "h264" {
			return "libx264"
		}
		if codec == "hevc" {
			return "libx265"
		}
		return "libx264"
	}
}
