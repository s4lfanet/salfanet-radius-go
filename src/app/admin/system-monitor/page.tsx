'use client';

import { useState, useEffect } from 'react';
import { Loader2, Database, Zap, Gauge, Trash2, RefreshCw, ShieldCheck } from 'lucide-react';
import { usePermissions } from '@/hooks/usePermissions';

interface CacheStats {
  enabled: boolean;
  stats?: {
    total: number;
    active: number;
    expired: number;
  };
}

interface QueueStats {
  enabled: boolean;
  stats?: {
    enqueued: number;
    completed: number;
    failed: number;
    retried: number;
    pending: number;
  };
}

export default function SystemMonitorPage() {
  const { loading: permLoading } = usePermissions();
  const [loading, setLoading] = useState(true);
  const [cacheStats, setCacheStats] = useState<CacheStats | null>(null);
  const [queueStats, setQueueStats] = useState<QueueStats | null>(null);
  const [rateLimit, setRateLimit] = useState<any>(null);

  useEffect(() => {
    loadAll();
    const interval = setInterval(loadAll, 15000);
    return () => clearInterval(interval);
  }, []);

  const loadAll = async () => {
    try {
      await Promise.all([loadCache(), loadQueue(), loadRateLimit()]);
    } finally {
      setLoading(false);
    }
  };

  const loadCache = async () => {
    const res = await fetch('/api/scaling/cache/stats');
    const data = await res.json();
    setCacheStats(data);
  };

  const loadQueue = async () => {
    const res = await fetch('/api/scaling/queue/stats');
    const data = await res.json();
    setQueueStats(data);
  };

  const loadRateLimit = async () => {
    const res = await fetch('/api/scaling/rate-limit/status');
    const data = await res.json();
    setRateLimit(data);
  };

  const handleFlushCache = async () => {
    if (!confirm('Flush semua cache? Ini akan menghapus semua cached data.')) return;
    await fetch('/api/scaling/cache/flush', { method: 'POST' });
    await loadCache();
  };

  if (permLoading || loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <Loader2 className="h-8 w-8 animate-spin text-cyan-400" />
      </div>
    );
  }

  return (
    <div className="space-y-6 p-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">System Monitor</h1>
          <p className="text-sm text-gray-400">Cache, Job Queue, Rate Limiting — auto-refresh 15s</p>
        </div>
        <button
          onClick={loadAll}
          className="flex items-center gap-2 rounded-lg border border-gray-700 px-3 py-1.5 text-sm text-gray-300 hover:text-white hover:border-cyan-500"
        >
          <RefreshCw className="h-4 w-4" /> Refresh
        </button>
      </div>

      {/* Cache Stats */}
      <div className="rounded-lg border border-gray-800 bg-gray-900/50 p-5">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <Database className="h-5 w-5 text-cyan-400" />
            <h3 className="text-sm font-medium text-white">In-Memory Cache</h3>
            <span className={`text-xs px-2 py-0.5 rounded ${cacheStats?.enabled ? 'bg-green-900/50 text-green-400' : 'bg-gray-800 text-gray-500'}`}>
              {cacheStats?.enabled ? 'ACTIVE' : 'DISABLED'}
            </span>
          </div>
          <button
            onClick={handleFlushCache}
            className="flex items-center gap-1 text-xs text-red-400 hover:text-red-300"
          >
            <Trash2 className="h-3 w-3" /> Flush
          </button>
        </div>
        {cacheStats?.stats && (
          <div className="grid grid-cols-3 gap-4">
            <div>
              <p className="text-xs text-gray-400">Total Entries</p>
              <p className="text-xl font-bold text-white">{cacheStats.stats.total}</p>
            </div>
            <div>
              <p className="text-xs text-gray-400">Active</p>
              <p className="text-xl font-bold text-green-400">{cacheStats.stats.active}</p>
            </div>
            <div>
              <p className="text-xs text-gray-400">Expired</p>
              <p className="text-xl font-bold text-yellow-400">{cacheStats.stats.expired}</p>
            </div>
          </div>
        )}
      </div>

      {/* Job Queue Stats */}
      <div className="rounded-lg border border-gray-800 bg-gray-900/50 p-5">
        <div className="flex items-center gap-2 mb-4">
          <Zap className="h-5 w-5 text-yellow-400" />
          <h3 className="text-sm font-medium text-white">Job Queue</h3>
          <span className={`text-xs px-2 py-0.5 rounded ${queueStats?.enabled ? 'bg-green-900/50 text-green-400' : 'bg-gray-800 text-gray-500'}`}>
            {queueStats?.enabled ? 'ACTIVE' : 'DISABLED'}
          </span>
        </div>
        {queueStats?.stats && (
          <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
            <div>
              <p className="text-xs text-gray-400">Enqueued</p>
              <p className="text-xl font-bold text-white">{queueStats.stats.enqueued}</p>
            </div>
            <div>
              <p className="text-xs text-gray-400">Completed</p>
              <p className="text-xl font-bold text-green-400">{queueStats.stats.completed}</p>
            </div>
            <div>
              <p className="text-xs text-gray-400">Failed</p>
              <p className="text-xl font-bold text-red-400">{queueStats.stats.failed}</p>
            </div>
            <div>
              <p className="text-xs text-gray-400">Retried</p>
              <p className="text-xl font-bold text-yellow-400">{queueStats.stats.retried}</p>
            </div>
            <div>
              <p className="text-xs text-gray-400">Pending</p>
              <p className="text-xl font-bold text-cyan-400">{queueStats.stats.pending}</p>
            </div>
          </div>
        )}
      </div>

      {/* Rate Limiting */}
      <div className="rounded-lg border border-gray-800 bg-gray-900/50 p-5">
        <div className="flex items-center gap-2 mb-4">
          <ShieldCheck className="h-5 w-5 text-green-400" />
          <h3 className="text-sm font-medium text-white">Rate Limiting</h3>
          <span className={`text-xs px-2 py-0.5 rounded ${rateLimit?.enabled ? 'bg-green-900/50 text-green-400' : 'bg-gray-800 text-gray-500'}`}>
            {rateLimit?.enabled ? 'ACTIVE' : 'DISABLED'}
          </span>
        </div>
        {rateLimit && (
          <div className="grid grid-cols-3 gap-4">
            <div>
              <p className="text-xs text-gray-400">Global API</p>
              <p className="text-sm font-medium text-white">{rateLimit.global?.max} req/{rateLimit.global?.window}</p>
            </div>
            <div>
              <p className="text-xs text-gray-400">Auth Endpoints</p>
              <p className="text-sm font-medium text-white">{rateLimit.auth?.max} req/{rateLimit.auth?.window}</p>
            </div>
            <div>
              <p className="text-xs text-gray-400">Portal Login</p>
              <p className="text-sm font-medium text-white">{rateLimit.portal?.max} req/{rateLimit.portal?.window}</p>
            </div>
          </div>
        )}
      </div>

      {/* Captive Portal Info */}
      <div className="rounded-lg border border-gray-800 bg-gray-900/50 p-5">
        <div className="flex items-center gap-2 mb-3">
          <Gauge className="h-5 w-5 text-purple-400" />
          <h3 className="text-sm font-medium text-white">Captive Portal</h3>
        </div>
        <div className="space-y-2 text-sm">
          <div className="flex items-center justify-between">
            <span className="text-gray-400">Endpoint</span>
            <code className="text-cyan-400 text-xs">/api/captive/identify</code>
          </div>
          <div className="flex items-center justify-between">
            <span className="text-gray-400">Portal Page</span>
            <code className="text-cyan-400 text-xs">/captive</code>
          </div>
          <p className="text-xs text-gray-500 mt-2">
            Captive portal untuk pelanggan isolir — identifikasi via IP, tampilkan tagihan, redirect ke pembayaran
          </p>
        </div>
      </div>
    </div>
  );
}
