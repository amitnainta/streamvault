import { useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'
import { useNavigate } from 'react-router-dom'
import type { Library, MediaItem } from '@/types/api'

export default function LibraryPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const { data: library } = useQuery({
    queryKey: ['library', id],
    queryFn: () => api.get<Library>(`/libraries/${id}`),
    enabled: !!id,
  })

  const { data: items, isLoading } = useQuery({
    queryKey: ['items', id],
    queryFn: () => api.get<MediaItem[]>(`/items?library=${id}&limit=200`),
    enabled: !!id,
  })

  return (
    <div className="p-8">
      <h1 className="text-xl font-semibold mb-6">{library?.name ?? '…'}</h1>
      {isLoading && <p className="text-muted-foreground">Loading…</p>}
      {items && (
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 xl:grid-cols-8 gap-4">
          {items.map((item) => (
            <button
              key={item.id}
              onClick={() => navigate(`/item/${item.id}`)}
              className="group relative rounded-lg overflow-hidden bg-muted aspect-[2/3] text-left"
            >
              <img
                src={`/artwork/${item.id}/poster`}
                alt={item.metadata?.title}
                className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
                loading="lazy"
                onError={(e) => { (e.target as HTMLImageElement).style.display = 'none' }}
              />
              <div className="absolute inset-0 bg-black/0 group-hover:bg-black/30 transition-colors" />
              <div className="absolute bottom-0 left-0 right-0 p-2 bg-gradient-to-t from-black/80 to-transparent opacity-0 group-hover:opacity-100 transition-opacity">
                <p className="text-xs text-white font-medium truncate">{item.metadata?.title ?? item.file_path.split('/').pop()}</p>
                {item.metadata?.year && <p className="text-xs text-white/70">{item.metadata.year}</p>}
              </div>
            </button>
          ))}
        </div>
      )}
      {items?.length === 0 && (
        <p className="text-muted-foreground text-sm">No items found. Run a library scan to add media.</p>
      )}
    </div>
  )
}
