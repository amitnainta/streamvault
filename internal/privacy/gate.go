package privacy

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Feature identifies a specific internet-connected capability.
// Each maps to a toggle in the Privacy & Internet settings UI.
type Feature string

const (
	FeatureTMDBMetadata   Feature = "tmdb_metadata"
	FeatureTMDBArtwork    Feature = "tmdb_artwork"
	FeatureMusicBrainz    Feature = "musicbrainz"
	FeatureCoverArt       Feature = "cover_art"
	FeatureUpdateCheck    Feature = "update_check"
	FeatureLetsEncrypt    Feature = "lets_encrypt"
	FeatureCrashReporting Feature = "crash_reporting"
	FeatureUsageStats     Feature = "usage_stats"
)

// ErrInternetDisabled is returned when the master internet toggle is OFF.
var ErrInternetDisabled = fmt.Errorf("internet access is disabled (master toggle is OFF)")

// ErrFeatureDisabled is returned when a specific feature toggle is OFF.
type ErrFeatureDisabled struct{ Feature Feature }

func (e ErrFeatureDisabled) Error() string {
	return fmt.Sprintf("internet feature %q is disabled", e.Feature)
}

// ActivityEntry records a single outbound request attempt.
type ActivityEntry struct {
	Timestamp   time.Time
	Feature     Feature
	URL         string
	Blocked     bool
	BlockReason string
	StatusCode  int
	DurationMs  int
}

// OutboundClient is the ONLY HTTP client permitted in the codebase.
// All external HTTP calls must go through this. The linter enforces this.
type OutboundClient struct {
	settings    SettingsReader
	activityLog ActivityLogger
	http        *http.Client
	logger      *zap.Logger
}

func NewOutboundClient(s SettingsReader, l *zap.Logger) *OutboundClient {
	return &OutboundClient{
		settings:    s,
		activityLog: s, // Settings also implements ActivityLogger
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: l,
	}
}

func (c *OutboundClient) Do(ctx context.Context, feature Feature, req *http.Request) (*http.Response, error) {
	// 1. Check master toggle
	if !c.settings.InternetEnabled() {
		entry := ActivityEntry{
			Timestamp:   time.Now(),
			Feature:     feature,
			URL:         req.URL.String(),
			Blocked:     true,
			BlockReason: "master internet toggle is OFF",
		}
		c.activityLog.RecordActivity(entry)
		c.logger.Debug("outbound request blocked", zap.String("feature", string(feature)), zap.String("url", req.URL.String()), zap.String("reason", entry.BlockReason))
		return nil, ErrInternetDisabled
	}

	// 2. Check feature-specific toggle
	if !c.settings.FeatureEnabled(feature) {
		entry := ActivityEntry{
			Timestamp:   time.Now(),
			Feature:     feature,
			URL:         req.URL.String(),
			Blocked:     true,
			BlockReason: fmt.Sprintf("feature %q toggle is OFF", feature),
		}
		c.activityLog.RecordActivity(entry)
		c.logger.Debug("outbound request blocked", zap.String("feature", string(feature)), zap.String("url", req.URL.String()), zap.String("reason", entry.BlockReason))
		return nil, ErrFeatureDisabled{Feature: feature}
	}

	// 3. Log the attempt
	start := time.Now()
	c.logger.Info("outbound request", zap.String("feature", string(feature)), zap.String("url", req.URL.String()))

	// 4. Execute
	resp, err := c.http.Do(req.WithContext(ctx))

	// 5. Record result
	entry := ActivityEntry{
		Timestamp:  start,
		Feature:    feature,
		URL:        req.URL.String(),
		Blocked:    false,
		DurationMs: int(time.Since(start).Milliseconds()),
	}
	if resp != nil {
		entry.StatusCode = resp.StatusCode
	}
	c.activityLog.RecordActivity(entry)

	return resp, err
}
