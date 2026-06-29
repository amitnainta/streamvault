package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/amitnainta/streamvault/internal/model"
)

type ItemHandler struct {
	db  *sql.DB
	log *zap.Logger
}

func NewItemHandler(db *sql.DB, log *zap.Logger) *ItemHandler {
	return &ItemHandler{db: db, log: log}
}

func (h *ItemHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	libraryID := q.Get("library")
	search := strings.TrimSpace(q.Get("search"))
	mediaType := q.Get("type")
	limit := 100
	if l, err := strconv.Atoi(q.Get("limit")); err == nil && l > 0 && l <= 500 {
		limit = l
	}

	orderBy := "m.added_at DESC"
	switch q.Get("sort") {
	case "title_asc":
		orderBy = "mt.title ASC"
	case "title_desc":
		orderBy = "mt.title DESC"
	case "year_desc":
		orderBy = "mt.year DESC, m.added_at DESC"
	case "year_asc":
		orderBy = "mt.year ASC, m.added_at DESC"
	}

	const selectCols = `
		SELECT m.id, m.library_id, m.type, m.file_path, m.file_size, m.duration_ms,
		       m.video_codec, m.video_width, m.video_height, m.audio_codec, m.container, m.added_at,
		       mt.title, mt.year, mt.description, mt.genres, mt.rating, mt.content_rating, mt.metadata_source
		FROM media_items m
		LEFT JOIN metadata mt ON mt.id = m.id`

	var rows *sql.Rows
	var err error

	if search != "" {
		// Use FTS5 full-text search
		fts := fts5Query(search)
		var where []string
		var args []any
		args = append(args, fts)
		if libraryID != "" {
			where = append(where, "m.library_id=?")
			args = append(args, libraryID)
		}
		if mediaType != "" {
			where = append(where, "m.type=?")
			args = append(args, mediaType)
		}
		clause := ""
		if len(where) > 0 {
			clause = "AND " + strings.Join(where, " AND ")
		}
		args = append(args, limit)
		rows, err = h.db.QueryContext(r.Context(),
			selectCols+`
			JOIN search_index si ON si.item_id = m.id
			WHERE si MATCH ? `+clause+`
			ORDER BY `+orderBy+`
			LIMIT ?`, args...)
	} else {
		var where []string
		var args []any
		if libraryID != "" {
			where = append(where, "m.library_id=?")
			args = append(args, libraryID)
		}
		if mediaType != "" {
			where = append(where, "m.type=?")
			args = append(args, mediaType)
		}
		clause := ""
		if len(where) > 0 {
			clause = "WHERE " + strings.Join(where, " AND ")
		}
		args = append(args, limit)
		rows, err = h.db.QueryContext(r.Context(),
			selectCols+`
			`+clause+`
			ORDER BY `+orderBy+`
			LIMIT ?`, args...)
	}

	if err != nil {
		h.log.Error("items list query failed", zap.Error(err))
		writeError(w, 500, "db error")
		return
	}
	defer rows.Close()

	var items []model.MediaItem
	for rows.Next() {
		item := scanItem(rows)
		items = append(items, item)
	}
	if items == nil {
		items = []model.MediaItem{}
	}
	writeJSON(w, 200, items)
}

// fts5Query converts a user search string into an FTS5 MATCH expression.
// Each word becomes a prefix match; special FTS5 chars are stripped.
func fts5Query(s string) string {
	replacer := strings.NewReplacer(`"`, ``, `'`, ``, `*`, ``, `(`, ``, `)`, ``, `:`, ``, `^`, ``, `-`, ` `)
	s = strings.TrimSpace(replacer.Replace(s))
	if s == "" {
		return `""`
	}
	words := strings.Fields(s)
	parts := make([]string, len(words))
	for i, w := range words {
		parts[i] = `"` + w + `"*`
	}
	return strings.Join(parts, " ")
}

func (h *ItemHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	row := h.db.QueryRowContext(r.Context(), `
		SELECT m.id, m.library_id, m.type, m.file_path, m.file_size, m.duration_ms,
		       m.video_codec, m.video_width, m.video_height, m.audio_codec, m.container, m.added_at,
		       mt.title, mt.year, mt.description, mt.genres, mt.rating, mt.content_rating, mt.metadata_source
		FROM media_items m
		LEFT JOIN metadata mt ON mt.id = m.id
		WHERE m.id=?`, id)

	item := scanItem(row)
	if item.ID == "" {
		writeError(w, 404, "item not found")
		return
	}

	// Load artwork
	artRows, _ := h.db.QueryContext(r.Context(),
		`SELECT id, art_type, file_path, width, height FROM artwork WHERE item_id=? AND item_type='media'`, id)
	defer artRows.Close()
	for artRows.Next() {
		var a model.Artwork
		artRows.Scan(&a.ID, &a.ArtType, &a.FilePath, &a.Width, &a.Height)
		a.ItemID = id
		item.Artwork = append(item.Artwork, a)
	}

	// Load streams
	streamRows, _ := h.db.QueryContext(r.Context(),
		`SELECT id, stream_type, index_num, codec, language, title, is_default, is_forced, is_external
		 FROM media_streams WHERE item_id=?`, id)
	defer streamRows.Close()
	for streamRows.Next() {
		var s model.MediaStream
		streamRows.Scan(&s.ID, &s.StreamType, &s.IndexNum, &s.Codec, &s.Language, &s.Title, &s.IsDefault, &s.IsForced, &s.IsExternal)
		s.ItemID = id
		item.Streams = append(item.Streams, s)
	}

	writeJSON(w, 200, item)
}

func (h *ItemHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var patch struct {
		Title         *string  `json:"title"`
		Year          *int     `json:"year"`
		Description   *string  `json:"description"`
		Genres        []string `json:"genres"`
		ContentRating *string  `json:"content_rating"`
	}
	if err := readJSON(r, &patch); err != nil {
		writeError(w, 400, "invalid body")
		return
	}

	// Upsert metadata with manual edits
	var fields []string
	var args []any
	if patch.Title != nil {
		fields = append(fields, "title=?")
		args = append(args, *patch.Title)
	}
	if patch.Year != nil {
		fields = append(fields, "year=?")
		args = append(args, *patch.Year)
	}
	if patch.Description != nil {
		fields = append(fields, "description=?")
		args = append(args, *patch.Description)
	}
	if patch.Genres != nil {
		b, _ := json.Marshal(patch.Genres)
		fields = append(fields, "genres=?")
		args = append(args, string(b))
	}
	if patch.ContentRating != nil {
		fields = append(fields, "content_rating=?")
		args = append(args, *patch.ContentRating)
	}

	if len(fields) == 0 {
		writeError(w, 400, "no fields to update")
		return
	}

	fields = append(fields, "is_manually_edited=1", "metadata_source='manual'")
	args = append(args, id)

	h.db.ExecContext(r.Context(),
		`INSERT INTO metadata(id, title) VALUES(?, COALESCE((SELECT title FROM metadata WHERE id=?), ''))
		 ON CONFLICT(id) DO UPDATE SET `+strings.Join(fields, ", "),
		append([]any{id, id}, args...)...,
	)
	w.WriteHeader(204)
}

func (h *ItemHandler) UploadArtwork(w http.ResponseWriter, r *http.Request) {
	// TODO: Phase 2 — handle multipart file upload, store in artwork dir
	writeError(w, 501, "not implemented yet")
}

// scanItem scans a row from the joined media_items + metadata query.
type scanner interface {
	Scan(dest ...any) error
}

func scanItem(row scanner) model.MediaItem {
	var item model.MediaItem
	var meta model.Metadata
	var genresJSON, metaSource sql.NullString
	var title, description, contentRating sql.NullString
	var year sql.NullInt64
	var rating sql.NullFloat64

	row.Scan(
		&item.ID, &item.LibraryID, &item.Type, &item.FilePath, &item.FileSize, &item.DurationMs,
		&item.VideoCodec, &item.VideoWidth, &item.VideoHeight, &item.AudioCodec, &item.Container, &item.AddedAt,
		&title, &year, &description, &genresJSON, &rating, &contentRating, &metaSource,
	)

	if title.Valid {
		meta.Title = title.String
	} else {
		// Fallback: use filename without extension
		meta.Title = strings.TrimSuffix(filepath.Base(item.FilePath), filepath.Ext(item.FilePath))
	}
	if year.Valid {
		meta.Year = int(year.Int64)
	}
	if description.Valid {
		meta.Description = description.String
	}
	if rating.Valid {
		meta.Rating = rating.Float64
	}
	if contentRating.Valid {
		meta.ContentRating = contentRating.String
	}
	if genresJSON.Valid {
		json.Unmarshal([]byte(genresJSON.String), &meta.Genres)
	}
	if metaSource.Valid {
		meta.MetadataSource = metaSource.String
	}

	item.Metadata = &meta
	return item
}
