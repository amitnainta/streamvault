import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'
import { Clock, Plus } from 'lucide-react'
import type { MediaItem, PlaybackProgress } from '@/types/api'
import { useAuthStore } from '@/store/authStore'
import { useNavigate } from 'react-router-dom'

export default function HomePage() {
  const { user } = useAuthStore()
  const navigate = useNavigate()

  const { data: recent } = useQuery({
    queryKey: ['items', 'recent'],
    queryFn: () => api.get<MediaItem[]>('/items?limit=24&sort=added_at_desc'),
  })

  const { data: progress } = useQuery({
    queryKey: ['progress', user?.id],
    queryFn: () => api.get<PlaybackProgress[]>(`/users/${user?.id}/progress?in_progress=true`),
    enabled: !!user?.id,
  })

  return (
    <div className="p-8 space-y-10">
      {/* Continue watching */}
      {progress && progress.length > 0 && (
        <section>
          <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
            <Clock size={18} className="text-accent" /> Continue Watching
          </h2>
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-4">
            {progress.slice(0, 6).map((p) => (
              <MediaCard key={p.item_id} itemId={p.item_id} progress={p.played_pct} />
            ))}
          </div>
        </section>
      )}

      {/* Recently added */}
      <section>
        <h2 className="text-lg font-semibold mb-4">Recently Added</h2>
        {recent && recent.length > 0 ? (
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-4">
            {recent.map((item) => (
              <MediaCard key={item.id} itemId={item.id} item={item} />
            ))}
          </div>
        ) : (
          <EmptyState onAdd={() => navigate('/admin/libraries')} />
        )}
      </section>
    </div>
  )
}

function MediaCard({ itemId, item, progress }: { itemId: string; item?: MediaItem; progress?: number }) {
  const navigate = useNavigate()
  const poster = item?.artwork?.find((a) => a.art_type === 'poster')

  return (
    <button
      onClick={() => navigate(`/item/${itemId}`)}
      className="group relative rounded-lg overflow-hidden bg-muted aspect-[2/3] text-left"
    >
      {poster ? (
        <img
          src={`/artwork/${itemId}/poster`}
          alt={item?.metadata?.title}
          className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
          loading="lazy"
        />
      ) : (
        <div className="w-full h-full flex items-center justify-center text-muted-foreground text-xs p-2 text-center">
          {item?.metadata?.title ?? itemId}
        </div>
      )}

      {/* Progress bar */}
      {progress !== undefined && progress > 0 && (
        <div className="absolute bottom-0 left-0 right-0 h-1 bg-muted/60">
          <div className="h-full bg-accent" style={{ width: `${progress * 100}%` }} />
        </div>
      )}

      {/* Hover overlay */}
      <div className="absolute inset-0 bg-black/0 group-hover:bg-black/30 transition-colors" />
    </button>
  )
}

function EmptyState({ onAdd }: { onAdd: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center py-20 text-center space-y-4">
      <div className="w-16 h-16 rounded-full bg-muted flex items-center justify-center">
        <Plus size={28} className="text-muted-foreground" />
      </div>
      <div>
        <p className="font-medium">No media yet</p>
        <p className="text-sm text-muted-foreground mt-1">Add a library to get started</p>
      </div>
      <button
        onClick={onAdd}
        className="mt-2 px-4 py-2 bg-accent text-accent-foreground rounded-lg text-sm font-medium hover:bg-accent/90 transition-colors"
      >
        Add Library
      </button>
    </div>
  )
}
