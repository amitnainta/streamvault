import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/api/client'
import { Plus, Scan, Trash2, Film, Tv, Music, X, FolderOpen } from 'lucide-react'
import type { Library } from '@/types/api'

const typeIcon = (t: string) =>
  t === 'movie' ? <Film size={16} /> : t === 'show' ? <Tv size={16} /> : <Music size={16} />

export default function AdminLibraries() {
  const qc = useQueryClient()
  const [showAdd, setShowAdd] = useState(false)
  const [scanning, setScanning] = useState<string | null>(null)

  const { data: libraries } = useQuery({
    queryKey: ['libraries'],
    queryFn: () => api.get<Library[]>('/libraries'),
  })

  const scanMutation = useMutation({
    mutationFn: async (id: string) => {
      setScanning(id)
      await api.post(`/libraries/${id}/scan`)
    },
    onSuccess: () => {
      setTimeout(() => {
        setScanning(null)
        qc.invalidateQueries({ queryKey: ['libraries'] })
        qc.invalidateQueries({ queryKey: ['items'] })
      }, 3000)
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/libraries/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['libraries'] })
      qc.invalidateQueries({ queryKey: ['items'] })
    },
  })

  return (
    <div className="p-8 space-y-6 max-w-3xl">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">Libraries</h1>
        <button
          onClick={() => setShowAdd(true)}
          className="flex items-center gap-2 bg-accent text-accent-foreground text-sm font-medium px-3 py-2 rounded-lg hover:bg-accent/90 transition-colors"
        >
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
                disabled={scanning === lib.id}
                className="text-muted-foreground hover:text-foreground p-1.5 rounded hover:bg-muted transition-colors disabled:opacity-50"
                title="Scan library"
              >
                <Scan size={15} className={scanning === lib.id ? 'animate-spin' : ''} />
              </button>
              <button
                onClick={() => deleteMutation.mutate(lib.id)}
                className="text-muted-foreground hover:text-red-400 p-1.5 rounded hover:bg-muted transition-colors"
                title="Remove library"
              >
                <Trash2 size={15} />
              </button>
            </div>
          </div>
        ))}

        {libraries?.length === 0 && (
          <div className="text-center py-16 text-muted-foreground">
            <FolderOpen size={40} className="mx-auto mb-3 opacity-30" />
            <p className="text-sm">No libraries yet. Add one to start scanning your media.</p>
          </div>
        )}
      </div>

      {showAdd && <AddLibraryModal onClose={() => setShowAdd(false)} onCreated={(id) => {
        setShowAdd(false)
        qc.invalidateQueries({ queryKey: ['libraries'] })
        scanMutation.mutate(id)
      }} />}
    </div>
  )
}

function AddLibraryModal({ onClose, onCreated }: { onClose: () => void; onCreated: (id: string) => void }) {
  const [name, setName] = useState('')
  const [type, setType] = useState<'movie' | 'show' | 'music'>('movie')
  const [path, setPath] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    if (!name.trim() || !path.trim()) { setError('Name and path are required'); return }
    setLoading(true)
    setError('')
    try {
      const lib = await api.post<Library>('/libraries', { name: name.trim(), type, paths: [path.trim()] })
      onCreated(lib.id)
    } catch (err: any) {
      setError(err.message || 'Failed to create library')
      setLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div className="bg-card border border-border rounded-2xl p-6 w-full max-w-md shadow-2xl">
        <div className="flex items-center justify-between mb-5">
          <h2 className="text-lg font-semibold">Add Library</h2>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground p-1 rounded">
            <X size={18} />
          </button>
        </div>

        <form onSubmit={submit} className="space-y-4">
          <div>
            <label className="text-sm text-muted-foreground mb-1 block">Library name</label>
            <input
              value={name}
              onChange={e => setName(e.target.value)}
              placeholder="e.g. Movies"
              className="w-full bg-muted border border-border rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-accent"
            />
          </div>

          <div>
            <label className="text-sm text-muted-foreground mb-1 block">Type</label>
            <div className="grid grid-cols-3 gap-2">
              {(['movie', 'show', 'music'] as const).map(t => (
                <button
                  key={t}
                  type="button"
                  onClick={() => setType(t)}
                  className={`flex items-center justify-center gap-1.5 py-2 rounded-lg border text-sm font-medium transition-colors
                    ${type === t ? 'border-accent bg-accent/10 text-accent' : 'border-border text-muted-foreground hover:border-accent/50'}`}
                >
                  {t === 'movie' ? <Film size={14} /> : t === 'show' ? <Tv size={14} /> : <Music size={14} />}
                  {t === 'movie' ? 'Movies' : t === 'show' ? 'TV Shows' : 'Music'}
                </button>
              ))}
            </div>
          </div>

          <div>
            <label className="text-sm text-muted-foreground mb-1 block">Folder path</label>
            <input
              value={path}
              onChange={e => setPath(e.target.value)}
              placeholder="e.g. D:\Movies or /mnt/media/movies"
              className="w-full bg-muted border border-border rounded-lg px-3 py-2 text-sm font-mono focus:outline-none focus:ring-1 focus:ring-accent"
            />
            <p className="text-xs text-muted-foreground mt-1">Full path to the folder on this server</p>
          </div>

          {error && <p className="text-sm text-red-400">{error}</p>}

          <div className="flex gap-2 pt-1">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 py-2 rounded-lg border border-border text-sm hover:bg-muted transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading}
              className="flex-1 py-2 rounded-lg bg-accent text-accent-foreground text-sm font-medium hover:bg-accent/90 transition-colors disabled:opacity-50"
            >
              {loading ? 'Creating...' : 'Add & Scan'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
