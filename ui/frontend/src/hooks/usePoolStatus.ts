import { useSyncExternalStore } from 'react';
import { api } from '@/lib/api';
import type { PoolStatus } from '@/lib/types';

// One shared poller for GET /cluster/pool-status, however many components
// subscribe (header badge + Cluster Manager). Reading pool-status is not
// "use", so polling it never wakes the cluster or keeps it alive.

const POLL_MS = 5000;

let current: PoolStatus | null = null;
let timer: ReturnType<typeof setInterval> | null = null;
const listeners = new Set<() => void>();

function emit(s: PoolStatus | null) {
  current = s;
  listeners.forEach((l) => l());
}

export async function refreshPoolStatus() {
  try {
    emit(await api.getPoolStatus());
  } catch {
    emit(null); // backend unreachable: show "unknown", never a stale green
  }
}

// setPoolStatus publishes a status the caller already has (e.g. a PUT response).
export function setPoolStatus(s: PoolStatus) {
  emit(s);
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  if (!timer) {
    refreshPoolStatus();
    timer = setInterval(refreshPoolStatus, POLL_MS);
  }
  return () => {
    listeners.delete(listener);
    if (listeners.size === 0 && timer) {
      clearInterval(timer);
      timer = null;
    }
  };
}

export function usePoolStatus(): PoolStatus | null {
  return useSyncExternalStore(subscribe, () => current);
}

export type TargetTone = 'up' | 'waking' | 'deleting' | 'stuck' | 'asleep' | 'stopped' | 'unknown';

// A teardown normally takes ~15s; past this it is stuck, not slow.
const STUCK_DELETE_SECONDS = 180;

export interface TargetDescription {
  label: string;
  detail: string;
  tone: TargetTone;
}

export function formatTTL(seconds: number): string {
  if (seconds <= 0) return 'never';
  if (seconds % 3600 === 0) return `${seconds / 3600} h`;
  return `${Math.round(seconds / 60)} min`;
}

// describeTarget maps pool-status to what the UI says about the target cluster.
export function describeTarget(p: PoolStatus | null): TargetDescription {
  if (!p) return { label: 'Unknown', detail: 'Backend not reachable', tone: 'unknown' };
  const err = p.lastError ? ` · last error: ${p.lastError}` : '';
  const ondemand = p.mode === 'ondemand';

  if (p.deleting) {
    const mins = Math.floor(p.deletingSeconds / 60);
    if (p.deletingSeconds >= STUCK_DELETE_SECONDS) {
      return {
        label: 'Deleting (stuck)',
        detail: `Teardown running for ${mins} min. Check the CAPK controller: kubectl -n capk-system get pods` + err,
        tone: 'stuck',
      };
    }
    return { label: 'Deleting…', detail: 'Tearing the cluster down' + err, tone: 'deleting' };
  }
  if (p.stopped && p.state === 'none') {
    return { label: 'Stopped', detail: 'Deleted from the UI; stays down until you Deploy again' + err, tone: 'stopped' };
  }
  if (p.state === 'building' || ((p.state === 'warm' || p.state === 'claimed') && !p.clusterReady)) {
    return {
      label: ondemand ? 'Waking…' : 'Building…',
      detail: (ondemand ? 'Accessed while asleep, building the cluster' : 'Building the standby cluster') + err,
      tone: 'waking',
    };
  }
  if (p.state === 'warm' || p.state === 'claimed') {
    if (!ondemand) {
      return {
        label: p.state === 'warm' ? 'Standby ready' : 'Up',
        detail: (p.state === 'warm' ? 'Deploy claims it instantly' : 'Standby pool mode') + err,
        tone: 'up',
      };
    }
    if (p.idleTTLSeconds <= 0) {
      return { label: 'Up', detail: 'On-demand · auto-delete off' + err, tone: 'up' };
    }
    const left = Math.max(0, Math.ceil((p.idleTTLSeconds - p.idleSeconds) / 60));
    return {
      label: 'Up',
      detail: `On-demand · auto-deletes after ${formatTTL(p.idleTTLSeconds)} idle · ${left} min left` + err,
      tone: 'up',
    };
  }
  if (ondemand) {
    return {
      label: 'Asleep',
      detail: 'Wakes on first access: Deploy, kubectl via 172.18.255.223, or the target agent' + err,
      tone: 'asleep',
    };
  }
  return { label: 'Down', detail: 'No target cluster' + err, tone: 'stopped' };
}

export const toneDot: Record<TargetTone, string> = {
  up: 'bg-green-500',
  waking: 'bg-amber-500 animate-pulse',
  deleting: 'bg-orange-500 animate-pulse',
  stuck: 'bg-red-500 animate-pulse',
  asleep: 'bg-sky-500',
  stopped: 'bg-red-500',
  unknown: 'bg-muted-foreground',
};
