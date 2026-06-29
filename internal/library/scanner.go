package library

import (
	"context"
	"database/sql"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/amitnainta/streamvault/internal/model"
)

var videoExtensions = map[string]bool{
	".mkv": true, ".mp4": true, ".avi": true, ".mov": true,
	".wmv": true, ".m4v": true, ".ts": true, ".m2ts": true,
	".webm": true, ".flv": true,
}

var audioExtensions = map[string]bool{
	".mp3": true, ".flac": true, ".aac": true, ".ogg": true,
	".opus": true, ".wav": true, ".m4a": true, ".wma": true,
}

// ScanProgress is broadcast via WebSocket during a scan.
type ScanProgress struct {
	LibraryID   string
	Scanned     int
	Total       int
	CurrentFile string
}

// Scanner scans directories and upserts media_items into the database.
type Scanner struct {
	db     *sql.DB
	logger *zap.Logger
	// OnProgress is called during scan — wire to WebSocket hub.
	OnProgress func(ScanProgress)
}

func NewScanner(db *sql.DB, logger *zap.Logger) *Scanner {
	return &Scanner{db: db, logger: logger}
}

// ScanLibrary performs an incremental scan of all paths in a library.
func (s *Scanner) ScanLibrary(ctx context.Context, lib model.Library) error {
	s.logger.Info("scanning library", zap.String("library", lib.Name), zap.String("id", lib.ID))

	// Collect existing file paths so we can detect removals
	existing, err := s.loadExistingPaths(lib.ID)
	if err != nil {
		return err
	}

	seen := make(map[string]bool)
	var count int

	for _, root := range lib.Paths {
		root = filepath.FromSlash(root) // normalise forward-slashes on Windows
		if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}

			ext := strings.ToLower(filepath.Ext(path))
			isMedia := (lib.Type == model.LibraryTypeMusic && audioExtensions[ext]) ||
				(lib.Type != model.LibraryTypeMusic && videoExtensions[ext])
			if !isMedia {
				return nil
			}

			seen[path] = true
			count++

			if s.OnProgress != nil {
				s.OnProgress(ScanProgress{LibraryID: lib.ID, Scanned: count, CurrentFile: filepath.Base(path)})
			}

			return s.upsertFile(ctx, lib, path)
		}); err != nil {
			s.logger.Error("walk error", zap.String("root", root), zap.Error(err))
		}
	}

	// Mark items whose files have been removed
	for path := range existing {
		if !seen[path] {
			s.markRemoved(path)
		}
	}

	// Update last_scan timestamp
	s.db.ExecContext(ctx,
		`UPDATE libraries SET last_scan=? WHERE id=?`,
		time.Now(), lib.ID,
	)

	s.logger.Info("scan complete", zap.String("library", lib.Name), zap.Int("files", count))
	return nil
}

func (s *Scanner) upsertFile(ctx context.Context, lib model.Library, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}

	// Check if already in DB with same size (fast change detection)
	var existingID string
	var existingSize int64
	s.db.QueryRowContext(ctx,
		`SELECT id, file_size FROM media_items WHERE file_path=?`, path,
	).Scan(&existingID, &existingSize)

	if existingID != "" && existingSize == info.Size() {
		return nil // unchanged
	}

	mediaType := classifyType(lib.Type, path)
	id := existingID
	if id == "" {
		id = uuid.New().String()
	}

	now := time.Now()
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO media_items(id,library_id,type,file_path,file_size,added_at,updated_at)
		 VALUES(?,?,?,?,?,?,?)
		 ON CONFLICT(file_path) DO UPDATE SET
		   file_size=excluded.file_size,
		   updated_at=excluded.updated_at`,
		id, lib.ID, mediaType, path, info.Size(), now, now,
	)
	if err != nil {
		return err
	}

	// Populate FTS5 search index with filename-derived title
	title := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	s.db.ExecContext(ctx, `DELETE FROM search_index WHERE item_id=?`, id)
	s.db.ExecContext(ctx, `INSERT INTO search_index(item_id, title) VALUES(?, ?)`, id, title)
	return nil
}

func (s *Scanner) loadExistingPaths(libraryID string) (map[string]bool, error) {
	rows, err := s.db.Query(`SELECT file_path FROM media_items WHERE library_id=?`, libraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[string]bool)
	for rows.Next() {
		var p string
		rows.Scan(&p)
		m[p] = true
	}
	return m, nil
}

func (s *Scanner) markRemoved(path string) {
	// For now: delete. Future: soft-delete with "missing" status.
	s.db.Exec(`DELETE FROM media_items WHERE file_path=?`, path)
}

func classifyType(libType model.LibraryType, path string) model.MediaType {
	switch libType {
	case model.LibraryTypeMusic:
		return model.MediaTypeTrack
	case model.LibraryTypeShow:
		return model.MediaTypeEpisode
	default:
		return model.MediaTypeMovie
	}
}
