import { useParams, useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'
import { Play, Star, Clock } from 'lucide-react'
import type { MediaItem } from '@/types/api'

function fmtDuration(ms: number) {
  const h = Math.floor(ms / 3_600_000)
  const m = Math.floor((ms % 3_600_000) / 60_000)
  return h > 0 ? `${h}h ${m}m` : `${m}m`
}

export default function ItemPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const { data: item, isLoading } = useQuery({
    queryKey: ['item', id],
    queryFn: () => api.get<MediaItem>(`/items/${id}`),
    enabled: !!id,
  })

  if (isLoading) return <div className="p-8 text-muted-foreground">Loading…</div>
  if (!item) return <div className="p-8 text-muted-foreground">Item not found.</div>

  const backdrop = item.artwork?.find((a) => a.art_type === 'backdrop')
  const meta = item.metadata

  return (
    <div className="relative min-h-screen">
      {/* Backdrop */}
      {backdrop && (
        <div className="absolute inset-0 z-0">
          <img src={`/artwork/${id}/backdrop`} alt="" className="w-full h-64 object-cover" />
          <div className="absolute inset-0 bg-gradient-to-b from-black/60 via-background/80 to-background" />
        </div>
      )}

      <div className="relative z-10 p-8 pt-48 space-y-6 max-w-4xl">
        <div className="flex gap-6">
          {/* Poster */}
          <div className="w-40 shrink-0 rounded-xl overflow-hidden bg-muted aspect-[2/3]">
            <img src={`/artwork/${id}/poster`} alt={meta?.title} className="w-full h-full object-cover" />
          </div>

          {/* Info */}
          <div className="space-y-3 pt-2">
            <h1 className="text-2xl font-bold">{meta?.title ?? item.file_path.split('/').pop()}</h1>

            <div className="flex items-center gap-3 text-sm text-muted-foreground flex-wrap">
              {meta?.year && <span>{meta.year}</span>}
              {item.duration_ms > 0 && (
                <span className="flex items-center gap-1"><Clock size={13} />{fmtDuration(item.duration_ms)}</span>
              )}
              {meta?.rating && (
                <span className="flex items-center gap-1"><Star size={13} className="text-yellow-400" />{meta.rating.toFixed(1)}</span>
              )}
              {meta?.content_rating && (
                <span className="border border-border px-1.5 py-0.5 rounded text-xs">{meta.content_rating}</span>
              )}
            </div>

            {meta?.genres && meta.genres.length > 0 && (
              <div className="flex gap-2 flex-wrap">
                {meta.genres.map((g) => (
                  <span key={g} className="bg-muted text-muted-foreground text-xs px-2 py-0.5 rounded-full">{g}</span>
                ))}
              </div>
            )}

            {meta?.description && (
              <p className="text-sm text-muted-foreground max-w-xl leading-relaxed">{meta.description}</p>
            )}

            {/* Play button */}
            <button
              onClick={() => navigate(`/player/${id}`)}
              className="flex items-center gap-2 bg-accent hover:bg-accent/90 text-accent-foreground font-medium rounded-lg px-5 py-2.5 text-sm transition-colors mt-2"
            >
              <Play size={16} fill="currentColor" /> Play
            </button>
          </div>
        </div>

        {/* Technical info */}
        <div className="border-t border-border pt-4 grid grid-cols-2 sm:grid-cols-4 gap-4 text-sm">
          {item.video_codec && <Stat label="Video" value={item.video_codec.toUpperCase()} />}
          {item.video_width && <Stat label="Resolution" value={`${item.video_width}×${item.video_height}`} />}
          {item.audio_codec && <Stat label="Audio" value={item.audio_codec.toUpperCase()} />}
          {item.container && <Stat label="Container" value={item.container.toUpperCase()} />}
        </div>
      </div>
    </div>
  )
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="font-medium">{value}</p>
    </div>
  )
}
