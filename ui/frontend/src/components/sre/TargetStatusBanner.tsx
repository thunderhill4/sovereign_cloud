import { useState } from 'react';
import { api } from '@/lib/api';
import { cn } from '@/lib/utils';
import { describeTarget, formatTTL, setPoolStatus, toneDot, usePoolStatus } from '@/hooks/usePoolStatus';

// Presets for the on-demand idle timer, in seconds. 0 = never auto-delete.
const PRESETS = [15 * 60, 30 * 60, 3600, 2 * 3600, 4 * 3600, 0];
const MIN_MINUTES = 1;
const MAX_MINUTES = 24 * 60;

// TargetStatusBanner: live target-cluster state (up / waking / asleep / stopped)
// and, in on-demand mode, the idle timer after which it is deleted.
export function TargetStatusBanner() {
  const pool = usePoolStatus();
  const target = describeTarget(pool);

  return (
    <div className="space-y-2 rounded-md border border-border bg-card px-3 py-2">
      <div className="flex flex-wrap items-center gap-2 text-xs">
        <span className={cn('h-2 w-2 rounded-full', toneDot[target.tone])} />
        <span className="font-medium">{target.label}</span>
        {pool && (
          <span className="rounded border border-border px-1 text-[10px] uppercase tracking-wide text-muted-foreground">
            {pool.mode === 'ondemand' ? 'on-demand' : 'standby pool'}
          </span>
        )}
        <span className="text-muted-foreground">{target.detail}</span>
      </div>
      {pool?.mode === 'ondemand' && (
        <IdleTimer key={pool.idleTTLSeconds} ttl={pool.idleTTLSeconds} idle={pool.idleSeconds} source={pool.idleTTLSource} />
      )}
    </div>
  );
}

function IdleTimer({ ttl, idle, source }: { ttl: number; idle: number; source: string }) {
  const [custom, setCustom] = useState('');
  const [selected, setSelected] = useState<number>(ttl);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const customSeconds = custom === '' ? null : Number(custom) * 60;
  const customValid =
    customSeconds !== null && Number.isInteger(Number(custom)) && Number(custom) >= MIN_MINUTES && Number(custom) <= MAX_MINUTES;
  const next = custom !== '' ? (customValid ? customSeconds! : null) : selected;
  const changed = next !== null && next !== ttl;
  const deletesNow = changed && next! > 0 && next! <= idle;

  async function save() {
    if (next === null) return;
    setSaving(true);
    setError('');
    try {
      setPoolStatus(await api.setPoolSettings(next));
      setCustom('');
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="flex flex-wrap items-center gap-2 text-xs">
      <span className="text-muted-foreground">Auto-delete after idle:</span>
      {PRESETS.map((s) => (
        <button
          key={s}
          onClick={() => { setSelected(s); setCustom(''); }}
          className={cn(
            'rounded-full border px-2.5 py-0.5 transition-colors',
            custom === '' && selected === s
              ? 'border-primary bg-primary text-primary-foreground'
              : 'border-border bg-secondary text-secondary-foreground hover:bg-accent hover:text-accent-foreground',
          )}
        >
          {s === 0 ? 'Never' : formatTTL(s)}
        </button>
      ))}
      <input
        type="number"
        min={MIN_MINUTES}
        max={MAX_MINUTES}
        placeholder="custom min"
        value={custom}
        onChange={(e) => setCustom(e.target.value)}
        className={cn(
          'w-24 rounded-md border bg-background px-2 py-0.5',
          custom !== '' && !customValid ? 'border-red-500' : 'border-border',
        )}
      />
      <button
        onClick={save}
        disabled={!changed || saving}
        className="rounded-md border border-primary bg-primary px-2.5 py-0.5 text-primary-foreground disabled:opacity-40"
      >
        {saving ? 'Saving…' : 'Save'}
      </button>
      <span className="text-muted-foreground">
        current: {formatTTL(ttl)}
        {source === 'ui' ? ' (set here)' : source === 'env' ? ' (POOL_IDLE_TTL)' : ' (default)'}
      </span>
      {custom !== '' && !customValid && (
        <span className="text-red-500">{MIN_MINUTES}–{MAX_MINUTES} minutes</span>
      )}
      {deletesNow && (
        <span className="text-amber-500">Idle {Math.floor(idle / 60)} min already: saving deletes the cluster within ~5 s</span>
      )}
      {error && <span className="text-red-500">{error}</span>}
    </div>
  );
}
