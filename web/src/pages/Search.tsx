import { useSearchParams, Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'
import type { MediaItem } from '@/types/api'

const TYPE_FILTERS = [
  { label: 'All', value: '' },
  { label: 'Movies', value: 'movie' },
  { label: 'TV Episodes', value: 'episode' },
  { label: 'Music', value: 'track' },
]

export default function SearchPage() {
  const [params, setParams] = useSearchParams()
  const q = params.get('q') ?? ''
  const type = params.get('type') ?? ''

  const { data: items = [], isFetching } = useQuery({
    queryKey: ['search', q, type],
    queryFn: () => {
      const qs = new URLSearchParams({ search: q, limit: '100' })
      if (type) qs.set('type', type)
      return api.get<MediaItem[]>(`/items?${qs}`)
    },
    enabled: q.length > 0,
  })

  const setType = (t: string) => {
    const next = new URLSearchParams(params)
    if (t) next.set('type', t)
    else next.delete('type')
    setParams(next)
  }

  return (
    <div className="p-6 space-y-5">
      <div className="flex items-center justify-between flex-wrap gap-3">
        <div>
          <h1 className="text-xl font-semibold text-foreground">
            {q ? `Results for "${q}"` : 'Search'}
          </h1>
          {q && !isFetching && (
            <p className="text-sm text-muted-foreground mt-0.5">
              {items.length} {items.length === 1 ? 'result' : 'results'}
            </p>
          )}
        </div>

        {/* Type filter chips */}
        <div className="flex gap-2 flex-wrap">
          {TYPE_FILTERS.map(f => (
            <button
              key={f.value}
              onClick={() => setType(f.value)}
              className={`px-3 py-1 rounded-full text-sm transition-colors ${
                type === f.value
                  ? 'bg-accent text-white'
                  : 'bg-muted text-muted-foreground hover:text-foreground'
              }`}
            >
              {f.label}
            </button>
          ))}
        </div>
      </div>

      {!q && (
        <p className="text-muted-foreground text-sm">Type in the search box to find movies, shows, and music.</p>
      )}

      {q && isFetching && (
        <div className="grid grid-cols-3 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-6 gap-4">
          {Array.from({ length: 12 }).map((_, i) => (
            <div key={i} className="aspect-[2/3] rounded-lg bg-muted animate-pulse" />
          ))}
        </div>
      )}

      {q && !isFetching && items.length === 0 && (
        <div className="text-center py-20 text-muted-foreground">
          <p className="text-lg">No results found for "{q}"</p>
          <p className="text-sm mt-1">Try a different search term or filter.</p>
        </div>
      )}

      {items.length > 0 && (
        <div className="grid grid-cols-3 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-6 gap-4">
          {items.map(item => (
            <Link key={item.id} to={`/item/${item.id}`} className="group block">
              <div className="relative aspect-[2/3] rounded-lg overflow-hidden bg-muted">
                {item.artwork?.find(a => a.art_type === 'poster') ? (
                  <img
                    src={`/artwork/${item.id}/poster`}
                    alt={item.metadata?.title}
                    className="w-full h-full object-cover"
                    loading="lazy"
                  />
                ) : (
                  <div className="w-full h-full flex items-center justify-center p-3 text-center">
                    <span className="text-muted-foreground text-xs leading-tight">{item.metadata?.title}</span>
                  </div>
                )}
                <div className="absolute inset-0 bg-black/60 opacity-0 group-hover:opacity-100 transition-opacity flex items-end p-2">
                  <div>
                    <p className="text-white text-xs font-medium leading-tight line-clamp-2">{item.metadata?.title}</p>
                    {item.metadata?.year ? <p className="text-white/60 text-xs">{item.metadata.year}</p> : null}
                  </div>
                </div>
              </div>
              <p className="mt-1.5 text-xs text-muted-foreground truncate group-hover:text-foreground transition-colors">
                {item.metadata?.title}
              </p>
            </Link>
          ))}
        </div>
      )}
    </div>
  )
}
