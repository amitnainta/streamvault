package transcode

import "os/exec"

// HWAccelType identifies the hardware acceleration method.
type HWAccelType string

const (
	HWAccelNone  HWAccelType = "none"
	HWAccelNVENC HWAccelType = "nvenc"
	HWAccelQSV   HWAccelType = "qsv"
	HWAccelAMF   HWAccelType = "amf"
	HWAccelVAAPI HWAccelType = "vaapi"
)

// DetectBestHWAccel probes FFmpeg by doing a real test encode for each
// hardware encoder in priority order. Listing encoders is not enough —
// an encoder can be compiled in but fail at runtime with no GPU present.
func DetectBestHWAccel() HWAccelType {
	candidates := []struct {
		hw      HWAccelType
		encoder string
	}{
		{HWAccelNVENC, "h264_nvenc"},
		{HWAccelQSV, "h264_qsv"},
		{HWAccelAMF, "h264_amf"},
		{HWAccelVAAPI, "h264_vaapi"},
	}

	for _, c := range candidates {
		if testEncoder(c.encoder) {
			return c.hw
		}
	}
	return HWAccelNone
}

// testEncoder runs a 1-second null encode to verify the encoder works at runtime.
func testEncoder(encoder string) bool {
	cmd := exec.Command("ffmpeg",
		"-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "nullsrc=s=128x128:d=1",
		"-c:v", encoder,
		"-f", "null", "-",
	)
	err := cmd.Run()
	return err == nil
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
