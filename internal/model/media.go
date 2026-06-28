package model

import "time"

type MediaType string

const (
	MediaTypeMovie   MediaType = "movie"
	MediaTypeEpisode MediaType = "episode"
	MediaTypeTrack   MediaType = "track"
)

type LibraryType string

const (
	LibraryTypeMovie LibraryType = "movie"
	LibraryTypeShow  LibraryType = "show"
	LibraryTypeMusic LibraryType = "music"
)

type Library struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Type      LibraryType `json:"type"`
	Paths     []string    `json:"paths"`
	CreatedAt time.Time   `json:"created_at"`
	LastScan  *time.Time  `json:"last_scan,omitempty"`
}

type MediaItem struct {
	ID        string    `json:"id"`
	LibraryID string    `json:"library_id"`
	Type      MediaType `json:"type"`
	FilePath  string    `json:"file_path"`
	FileSize  int64     `json:"file_size"`
	FileHash  string    `json:"file_hash,omitempty"`

	// Video properties
	DurationMs   int64  `json:"duration_ms"`
	VideoCodec   string `json:"video_codec,omitempty"`
	VideoWidth   int    `json:"video_width,omitempty"`
	VideoHeight  int    `json:"video_height,omitempty"`
	VideoBitrate int64  `json:"video_bitrate,omitempty"`
	HDRFormat    string `json:"hdr_format,omitempty"`

	// Audio (primary stream)
	AudioCodec    string `json:"audio_codec,omitempty"`
	AudioChannels int    `json:"audio_channels,omitempty"`
	AudioLanguage string `json:"audio_language,omitempty"`

	Container string    `json:"container,omitempty"`
	AddedAt   time.Time `json:"added_at"`

	// Populated by joins
	Metadata *Metadata    `json:"metadata,omitempty"`
	Streams  []MediaStream `json:"streams,omitempty"`
	Artwork  []Artwork    `json:"artwork,omitempty"`
}

type Metadata struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	SortTitle     string    `json:"sort_title,omitempty"`
	OriginalTitle string    `json:"original_title,omitempty"`
	Year          int       `json:"year,omitempty"`
	Description   string    `json:"description,omitempty"`
	Tagline       string    `json:"tagline,omitempty"`
	Genres        []string  `json:"genres,omitempty"`
	Rating        float64   `json:"rating,omitempty"`
	ContentRating string    `json:"content_rating,omitempty"`
	Language      string    `json:"language,omitempty"`

	// External IDs — only set if internet was used
	TMDBID int    `json:"tmdb_id,omitempty"`
	IMDBID string `json:"imdb_id,omitempty"`

	MetadataSource    string     `json:"metadata_source,omitempty"` // tmdb|nfo|manual
	MetadataFetchedAt *time.Time `json:"metadata_fetched_at,omitempty"`
	IsManuallyEdited  bool       `json:"is_manually_edited"`
}

type MediaStream struct {
	ID         string `json:"id"`
	ItemID     string `json:"item_id"`
	StreamType string `json:"stream_type"` // audio|subtitle
	IndexNum   int    `json:"index_num"`
	Codec      string `json:"codec,omitempty"`
	Language   string `json:"language,omitempty"`
	Title      string `json:"title,omitempty"`
	IsDefault  bool   `json:"is_default"`
	IsForced   bool   `json:"is_forced"`
	IsExternal bool   `json:"is_external"`
	FilePath   string `json:"file_path,omitempty"`
}

type Artwork struct {
	ID       string `json:"id"`
	ItemID   string `json:"item_id"`
	ItemType string `json:"item_type"`
	ArtType  string `json:"art_type"` // poster|backdrop|logo|thumb|cover
	FilePath string `json:"file_path"` // relative to data_dir/artwork/
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
}

type Show struct {
	ID          string    `json:"id"`
	LibraryID   string    `json:"library_id"`
	Title       string    `json:"title"`
	SortTitle   string    `json:"sort_title,omitempty"`
	Year        int       `json:"year,omitempty"`
	Description string    `json:"description,omitempty"`
	Genres      []string  `json:"genres,omitempty"`
	Rating      float64   `json:"rating,omitempty"`
	Status      string    `json:"status,omitempty"`
	TMDBID      int       `json:"tmdb_id,omitempty"`
	AddedAt     time.Time `json:"added_at"`
}

type Season struct {
	ID          string `json:"id"`
	ShowID      string `json:"show_id"`
	SeasonNum   int    `json:"season_num"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Year        int    `json:"year,omitempty"`
}

type Episode struct {
	MediaItem
	ShowID     string `json:"show_id"`
	SeasonID   string `json:"season_id"`
	SeasonNum  int    `json:"season_num"`
	EpisodeNum int    `json:"episode_num"`
}
