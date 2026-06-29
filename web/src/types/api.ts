export type LibraryType = 'movie' | 'show' | 'music'
export type MediaType = 'movie' | 'episode' | 'track'
export type Role = 'admin' | 'viewer'

export interface Library {
  id: string
  name: string
  type: LibraryType
  paths: string[]
  created_at: string
  last_scan?: string
}

export interface Metadata {
  title: string
  sort_title?: string
  year?: number
  description?: string
  genres?: string[]
  rating?: number
  content_rating?: string
  tmdb_id?: number
  imdb_id?: string
  metadata_source?: string
  is_manually_edited: boolean
}

export interface Artwork {
  id: string
  art_type: 'poster' | 'backdrop' | 'logo' | 'thumb' | 'cover'
  file_path: string
  width?: number
  height?: number
}

export interface MediaStream {
  id: string
  stream_type: 'audio' | 'subtitle'
  index_num: number
  codec?: string
  language?: string
  title?: string
  is_default: boolean
  is_forced: boolean
  is_external: boolean
}

export interface MediaItem {
  id: string
  library_id: string
  type: MediaType
  file_path: string
  file_size: number
  duration_ms: number
  video_codec?: string
  video_width?: number
  video_height?: number
  audio_codec?: string
  container?: string
  added_at: string
  metadata?: Metadata
  artwork?: Artwork[]
  streams?: MediaStream[]
}

export interface PlaybackInfo {
  type: 'hls' | 'direct'
  url: string
  session_id?: string
}

export interface PlaybackProgress {
  item_id: string
  position_ms: number
  played_pct: number
  is_watched: boolean
  last_played_at: string
}

export interface User {
  id: string
  username: string
  role: Role
  is_enabled: boolean
  created_at: string
  last_login_at?: string
}

export interface PrivacySettings {
  internet_enabled: boolean
  features: Record<string, boolean>
}

export interface ActivityEntry {
  timestamp: string
  feature: string
  url: string
  blocked: boolean
  block_reason?: string
  status_code?: number
  duration_ms?: number
}

export interface ServerStats {
  cpu_pct: number
  memory_mb: number
  active_sessions: number
  library_count: number
  item_count: number
  uptime_s: number
}

export interface PagedResponse<T> {
  items: T[]
  cursor?: string
  total: number
}
