import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/api/client'
import { Plus, Scan, Trash2, Film, Tv, Music } from 'lucide-react'
import type { Library } from '@/types/api'

const typeIcon = (t: string) =>
  t === 'movie' ? <Film size={16} /> : t === 'show' ? <Tv size={16} /> : <Music size={16} />

export default function AdminLibraries() {
  const qc = useQueryClient()

  const { data: libraries } = useQuery({
    queryKey: ['libraries'],
    queryFn: () => api.get<Library[]>('/libraries'),
  })

  const scanMutation = useMutation({
    mutationFn: (id: string) => api.post(`/libraries/${id}/scan`),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/libraries/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['libraries'] }),
  })

  return (
    <div className="p-8 space-y-6 max-w-3xl">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">Libraries</h1>
        <button className="flex items-center gap-2 bg-accent text-accent-foreground text-sm font-medium px-3 py-2 rounded-lg hover:bg-accent/90 transition-colors">
          <Plus size={15} /> Add Library
        </button>
      </div>

      <div className="space-y-3">
        {libraries?.map((lib) => (
          <div key={lib.id} className="bg-card border border-border rounded-xl p-4 flex items-center gap-4">
            <div className="text-muted-foreground">{typeIcon(lib.type)}</div>
            <div className="flex-1 min-w-0">
              <p className="font-medium">{lib.name}</p>
              <p className="text-xs text-muted-foreground truncate">{lib.paths.join(', ')}</p>
              {lib.last_scan && (
                <p className="text-xs text-muted-foreground mt-0.5">
                  Last scanned: {new Date(lib.last_scan).toLocaleString()}
                </p>
              )}
            </div>
            <div className="flex items-center gap-2">
              <button
                onClick={() => scanMutation.mutate(lib.id)}
                className="text-muted-foreground hover:text-foreground p-1.5 rounded hover:bg-muted transition-colors"
                title="Scan library"
              >
                <Scan size={15} />
              </button>
              <button
                onClick={() => deleteMutation.mutate(lib.id)}
                className="text-muted-foreground hover:text-destructive p-1.5 rounded hover:bg-muted transition-colors"
                title="Remove library"
              >
                <Trash2 size={15} />
              </button>
            </div>
          </div>
        ))}

        {libraries?.length === 0 && (
          <p className="text-sm text-muted-foreground">No libraries yet. Add one to start scanning your media.</p>
        )}
      </div>
    </div>
  )
}
