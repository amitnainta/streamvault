import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/api/client'
import { Shield, Globe, GlobeLock, Clock, CheckCircle2, XCircle } from 'lucide-react'
import type { PrivacySettings, ActivityEntry } from '@/types/api'

// ── Feature definitions ──────────────────────────────────────────────────────

interface FeatureDef {
  key: string
  label: string
  hosts: string[]
  dataSent: string
  dataReceived: string
  disabledEffect: string
}

const FEATURES: FeatureDef[] = [
  {
    key: 'tmdb_metadata',
    label: 'Movie & TV Metadata (TMDB)',
    hosts: ['api.themoviedb.org'],
    dataSent: 'Movie or show title, release year, preferred language',
    dataReceived: 'Title, synopsis, genres, cast, rating, poster URLs',
    disabledEffect: 'Items appear without metadata. Posters show a placeholder. You can add metadata manually.',
  },
  {
    key: 'tmdb_artwork',
    label: 'Movie & TV Artwork (TMDB)',
    hosts: ['image.tmdb.org'],
    dataSent: 'Image URL paths (from metadata lookup results)',
    dataReceived: 'JPEG/PNG poster, backdrop, and logo images',
    disabledEffect: 'No poster or backdrop images are downloaded. Previously cached images still show.',
  },
  {
    key: 'musicbrainz',
    label: 'Music Metadata (MusicBrainz)',
    hosts: ['musicbrainz.org'],
    dataSent: 'Artist name, album title, track title',
    dataReceived: 'Track metadata, MusicBrainz ID, album information',
    disabledEffect: 'Music items appear with filename-based titles only. No genre or release date.',
  },
  {
    key: 'cover_art',
    label: 'Album Artwork (Cover Art Archive)',
    hosts: ['coverartarchive.org'],
    dataSent: 'MusicBrainz release ID',
    dataReceived: 'Album cover JPEG/PNG image',
    disabledEffect: 'Album covers show a placeholder. Previously cached artwork still shows.',
  },
  {
    key: 'update_check',
    label: 'Software Update Checks',
    hosts: ['api.github.com'],
    dataSent: 'StreamVault version number',
    dataReceived: 'Latest release version number only',
    disabledEffect: 'No update notifications shown. You can still update manually at any time.',
  },
  {
    key: 'lets_encrypt',
    label: "Let's Encrypt TLS Certificate",
    hosts: ['acme-v02.api.letsencrypt.org'],
    dataSent: 'Your domain name, ACME challenge response',
    dataReceived: 'TLS certificate for HTTPS',
    disabledEffect: 'Automatic certificate renewal is disabled. You can still use a manually provided certificate.',
  },
  {
    key: 'crash_reporting',
    label: 'Crash & Error Reporting (opt-in)',
    hosts: ['Your configured endpoint'],
    dataSent: 'Stack trace, OS type, StreamVault version. Never: filenames, titles, or user data.',
    dataReceived: 'Confirmation receipt',
    disabledEffect: 'No crash reports are sent. Errors are still logged locally.',
  },
]

// ── Component ─────────────────────────────────────────────────────────────────

export default function AdminPrivacy() {
  const qc = useQueryClient()

  const { data: settings } = useQuery({
    queryKey: ['settings', 'privacy'],
    queryFn: () => api.get<PrivacySettings>('/settings/privacy'),
  })

  const { data: activity } = useQuery({
    queryKey: ['activity-log'],
    queryFn: () => api.get<ActivityEntry[]>('/settings/privacy/activity?limit=50'),
  })

  const updateMutation = useMutation({
    mutationFn: (patch: Partial<PrivacySettings>) =>
      api.patch<PrivacySettings>('/settings/privacy', patch),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['settings', 'privacy'] }),
  })

  const masterOn = settings?.internet_enabled ?? false

  function toggleMaster() {
    if (!masterOn) {
      const confirmed = window.confirm(
        'Enabling internet access will allow StreamVault to contact external services for the features you enable below. No data will be sent without your explicit permission per feature.\n\nContinue?'
      )
      if (!confirmed) return
    }
    updateMutation.mutate({ internet_enabled: !masterOn })
  }

  function toggleFeature(key: string, current: boolean) {
    if (!current) {
      const feature = FEATURES.find((f) => f.key === key)
      const confirmed = window.confirm(
        `This will allow StreamVault to contact:\n${feature?.hosts.join(', ')}\n\nData sent: ${feature?.dataSent}\n\nContinue?`
      )
      if (!confirmed) return
    }
    updateMutation.mutate({ features: { ...settings?.features, [key]: !current } })
  }

  return (
    <div className="p-8 max-w-3xl space-y-8">
      {/* Header */}
      <div>
        <div className="flex items-center gap-3 mb-1">
          <Shield size={22} className="text-accent" />
          <h1 className="text-xl font-semibold">Privacy & Internet</h1>
        </div>
        <p className="text-sm text-muted-foreground">
          StreamVault works completely offline. Every feature that contacts the internet is listed here.
          Nothing leaves your device without your explicit consent.
        </p>
      </div>

      {/* Master toggle */}
      <div className={`rounded-xl border-2 p-5 transition-colors ${masterOn ? 'border-accent/40 bg-accent/5' : 'border-border bg-card'}`}>
        <div className="flex items-start justify-between gap-4">
          <div className="flex items-start gap-3">
            {masterOn
              ? <Globe size={20} className="text-accent mt-0.5 shrink-0" />
              : <GlobeLock size={20} className="text-muted-foreground mt-0.5 shrink-0" />
            }
            <div>
              <p className="font-medium">Allow Internet Access</p>
              <p className="text-sm text-muted-foreground mt-0.5">
                {masterOn
                  ? 'Internet is enabled. Only features you turn on below will make outbound requests.'
                  : 'Internet is blocked. StreamVault makes zero outbound network requests. All features work offline.'}
              </p>
            </div>
          </div>
          <Toggle on={masterOn} onChange={toggleMaster} />
        </div>
      </div>

      {/* Per-feature toggles */}
      <section className="space-y-3">
        <h2 className="text-sm font-medium text-muted-foreground uppercase tracking-wider">
          Internet Features
        </h2>

        {FEATURES.map((f) => {
          const enabled = (settings?.features?.[f.key] ?? false) && masterOn
          const available = masterOn

          return (
            <div
              key={f.key}
              className={`rounded-xl border p-4 space-y-3 transition-opacity ${!available ? 'opacity-40' : ''} bg-card border-border`}
            >
              <div className="flex items-start justify-between gap-4">
                <div className="flex-1 min-w-0">
                  <p className="font-medium text-sm">{f.label}</p>
                  <div className="flex flex-wrap gap-1 mt-1">
                    {f.hosts.map((h) => (
                      <span key={h} className="text-xs bg-muted text-muted-foreground px-2 py-0.5 rounded font-mono">
                        {h}
                      </span>
                    ))}
                  </div>
                </div>
                <Toggle on={enabled} onChange={() => toggleFeature(f.key, enabled)} disabled={!available} />
              </div>

              <div className="grid grid-cols-2 gap-3 text-xs">
                <div className="space-y-0.5">
                  <p className="text-muted-foreground font-medium">Data sent</p>
                  <p className="text-foreground">{f.dataSent}</p>
                </div>
                <div className="space-y-0.5">
                  <p className="text-muted-foreground font-medium">Data received</p>
                  <p className="text-foreground">{f.dataReceived}</p>
                </div>
              </div>

              <div className="text-xs text-muted-foreground border-t border-border pt-2">
                <span className="font-medium">If disabled: </span>{f.disabledEffect}
              </div>
            </div>
          )
        })}
      </section>

      {/* What is NEVER sent */}
      <section className="rounded-xl border border-border bg-card p-4 space-y-2">
        <p className="text-sm font-medium">What is never sent — under any circumstance</p>
        <ul className="text-sm text-muted-foreground space-y-1">
          {[
            'Media filenames, paths, or file hashes',
            'Your watch history or playback data',
            'User names, emails, or passwords',
            'Library structure or item counts',
            'Device identifiers or IP addresses',
            'Any data to StreamVault\'s own servers (StreamVault has no cloud servers)',
          ].map((item) => (
            <li key={item} className="flex items-start gap-2">
              <XCircle size={14} className="text-destructive mt-0.5 shrink-0" />
              {item}
            </li>
          ))}
        </ul>
      </section>

      {/* Activity log */}
      <section className="space-y-3">
        <h2 className="text-sm font-medium text-muted-foreground uppercase tracking-wider flex items-center gap-2">
          <Clock size={14} /> Network Activity Log
        </h2>
        <div className="rounded-xl border border-border bg-card overflow-hidden">
          {!activity || activity.length === 0 ? (
            <p className="text-sm text-muted-foreground p-4">No network activity recorded.</p>
          ) : (
            <table className="w-full text-xs">
              <thead className="border-b border-border">
                <tr className="text-muted-foreground">
                  <th className="text-left p-3 font-medium">Time</th>
                  <th className="text-left p-3 font-medium">Feature</th>
                  <th className="text-left p-3 font-medium">URL</th>
                  <th className="text-left p-3 font-medium">Result</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {activity.map((e, i) => (
                  <tr key={i} className="hover:bg-muted/30 transition-colors">
                    <td className="p-3 text-muted-foreground whitespace-nowrap">
                      {new Date(e.timestamp).toLocaleTimeString()}
                    </td>
                    <td className="p-3 font-mono">{e.feature}</td>
                    <td className="p-3 text-muted-foreground max-w-xs truncate">{e.url}</td>
                    <td className="p-3">
                      {e.blocked ? (
                        <span className="flex items-center gap-1 text-muted-foreground">
                          <XCircle size={12} /> Blocked
                        </span>
                      ) : (
                        <span className="flex items-center gap-1 text-green-500">
                          <CheckCircle2 size={12} /> {e.status_code}
                        </span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </section>
    </div>
  )
}

function Toggle({ on, onChange, disabled }: { on: boolean; onChange: () => void; disabled?: boolean }) {
  return (
    <button
      onClick={onChange}
      disabled={disabled}
      aria-pressed={on}
      className={`relative w-10 h-6 rounded-full transition-colors shrink-0 focus:outline-none focus:ring-2 focus:ring-accent focus:ring-offset-2 focus:ring-offset-background disabled:cursor-not-allowed ${on ? 'bg-accent' : 'bg-muted'}`}
    >
      <span
        className={`absolute top-1 left-1 w-4 h-4 rounded-full bg-white transition-transform ${on ? 'translate-x-4' : 'translate-x-0'}`}
      />
    </button>
  )
}
