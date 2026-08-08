'use client';

import { useState, useEffect } from 'react';
import { Loader2, Signal, AlertTriangle, Activity, TrendingDown, Search } from 'lucide-react';
import { usePermissions } from '@/hooks/usePermissions';
import { formatWIB } from '@/lib/timezone';

interface RxSummary {
  totalOnu: number;
  onlineOnu: number;
  goodSignal: number;
  weakSignal: number;
  badSignal: number;
  noSignal: number;
  avgRxPower: number;
}

interface OltBreakdown {
  oltId: string;
  oltName: string;
  totalOnu: number;
  onlineOnu: number;
  weakSignal: number;
  badSignal: number;
  avgRxPower: number;
}

interface DegradedONU {
  id: string;
  serialNumber: string | null;
  status: string;
  rxPower: number | null;
  txPower: number | null;
  description: string | null;
  frame: number;
  slot: number;
  port: number;
  onuId: number;
  customer: { name: string; username: string } | null;
  olt: { name: string } | null;
  lastSeenAt: string | null;
}

export default function RxPowerPage() {
  const { loading: permLoading } = usePermissions();
  const [loading, setLoading] = useState(true);
  const [summary, setSummary] = useState<RxSummary | null>(null);
  const [breakdown, setBreakdown] = useState<OltBreakdown[]>([]);
  const [degraded, setDegraded] = useState<DegradedONU[]>([]);
  const [degradedTotal, setDegradedTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState('');

  useEffect(() => {
    loadSummary();
    loadDegraded();
  }, []);

  useEffect(() => {
    loadDegraded();
  }, [page]);

  const loadSummary = async () => {
    try {
      const res = await fetch('/api/integration/rx-power/summary');
      const data = await res.json();
      setSummary(data.summary);
      setBreakdown(data.breakdown || []);
    } catch (error) {
      console.error('Load RX summary error:', error);
    }
  };

  const loadDegraded = async () => {
    setLoading(true);
    try {
      const res = await fetch(`/api/integration/rx-power/degraded?page=${page}&pageSize=50`);
      const data = await res.json();
      setDegraded(data.data || []);
      setDegradedTotal(data.total || 0);
    } catch (error) {
      console.error('Load degraded ONUs error:', error);
    } finally {
      setLoading(false);
    }
  };

  const formatDbm = (val: number | null | undefined) => {
    if (val === null || val === undefined || val === 0) return '-';
    return `${val.toFixed(2)} dBm`;
  };

  const signalColor = (rx: number | null) => {
    if (rx === null || rx === 0) return 'text-gray-500';
    if (rx >= -20) return 'text-green-400';
    if (rx >= -27) return 'text-yellow-400';
    return 'text-red-400';
  };

  const filteredDegraded = degraded.filter(o =>
    (o.serialNumber || '').toLowerCase().includes(search.toLowerCase()) ||
    (o.customer?.name || '').toLowerCase().includes(search.toLowerCase()) ||
    (o.customer?.username || '').toLowerCase().includes(search.toLowerCase())
  );

  if (permLoading || loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <Loader2 className="h-8 w-8 animate-spin text-cyan-400" />
      </div>
    );
  }

  return (
    <div className="space-y-6 p-6">
      <div>
        <h1 className="text-2xl font-bold text-white">Monitoring RX Power</h1>
        <p className="text-sm text-gray-400">Monitor kualitas sinyal optik ONU per OLT</p>
      </div>

      {/* Summary Cards */}
      {summary && (
        <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-4">
          <div className="rounded-lg border border-gray-800 bg-gray-900/50 p-4">
            <div className="flex items-center gap-2">
              <Activity className="h-4 w-4 text-cyan-400" />
              <p className="text-xs text-gray-400">Total ONU</p>
            </div>
            <p className="mt-1 text-xl font-bold text-white">{summary.totalOnu}</p>
          </div>
          <div className="rounded-lg border border-gray-800 bg-gray-900/50 p-4">
            <div className="flex items-center gap-2">
              <Signal className="h-4 w-4 text-green-400" />
              <p className="text-xs text-gray-400">Online</p>
            </div>
            <p className="mt-1 text-xl font-bold text-white">{summary.onlineOnu}</p>
          </div>
          <div className="rounded-lg border border-gray-800 bg-gray-900/50 p-4">
            <div className="flex items-center gap-2">
              <Signal className="h-4 w-4 text-green-400" />
              <p className="text-xs text-gray-400">Sinyal Baik</p>
            </div>
            <p className="mt-1 text-xl font-bold text-green-400">{summary.goodSignal}</p>
            <p className="text-[10px] text-gray-500">&ge; -20 dBm</p>
          </div>
          <div className="rounded-lg border border-gray-800 bg-gray-900/50 p-4">
            <div className="flex items-center gap-2">
              <TrendingDown className="h-4 w-4 text-yellow-400" />
              <p className="text-xs text-gray-400">Lemah</p>
            </div>
            <p className="mt-1 text-xl font-bold text-yellow-400">{summary.weakSignal}</p>
            <p className="text-[10px] text-gray-500">-20 s/d -27 dBm</p>
          </div>
          <div className="rounded-lg border border-gray-800 bg-gray-900/50 p-4">
            <div className="flex items-center gap-2">
              <AlertTriangle className="h-4 w-4 text-red-400" />
              <p className="text-xs text-gray-400">Buruk</p>
            </div>
            <p className="mt-1 text-xl font-bold text-red-400">{summary.badSignal}</p>
            <p className="text-[10px] text-gray-500">&lt; -27 dBm</p>
          </div>
          <div className="rounded-lg border border-gray-800 bg-gray-900/50 p-4">
            <div className="flex items-center gap-2">
              <Signal className="h-4 w-4 text-gray-500" />
              <p className="text-xs text-gray-400">Rata-rata</p>
            </div>
            <p className="mt-1 text-xl font-bold text-cyan-400">{summary.avgRxPower.toFixed(2)}</p>
            <p className="text-[10px] text-gray-500">dBm</p>
          </div>
        </div>
      )}

      {/* Per-OLT Breakdown */}
      {breakdown.length > 0 && (
        <div className="rounded-lg border border-gray-800 bg-gray-900/50 overflow-hidden">
          <h3 className="px-4 py-3 text-sm font-medium text-white border-b border-gray-800">Ringkasan per OLT</h3>
          <table className="w-full text-sm">
            <thead className="border-b border-gray-800 bg-gray-900">
              <tr>
                <th className="px-4 py-2 text-left text-gray-400">OLT</th>
                <th className="px-4 py-2 text-right text-gray-400">Total ONU</th>
                <th className="px-4 py-2 text-right text-gray-400">Online</th>
                <th className="px-4 py-2 text-right text-gray-400">Lemah</th>
                <th className="px-4 py-2 text-right text-gray-400">Buruk</th>
                <th className="px-4 py-2 text-right text-gray-400">Avg RX</th>
              </tr>
            </thead>
            <tbody>
              {breakdown.map((b) => (
                <tr key={b.oltId} className="border-b border-gray-800/50">
                  <td className="px-4 py-2 text-white">{b.oltName}</td>
                  <td className="px-4 py-2 text-right text-gray-300">{b.totalOnu}</td>
                  <td className="px-4 py-2 text-right text-green-400">{b.onlineOnu}</td>
                  <td className="px-4 py-2 text-right text-yellow-400">{b.weakSignal}</td>
                  <td className="px-4 py-2 text-right text-red-400">{b.badSignal}</td>
                  <td className={`px-4 py-2 text-right ${signalColor(b.avgRxPower)}`}>
                    {b.avgRxPower !== 0 ? `${b.avgRxPower.toFixed(2)} dBm` : '-'}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Degraded ONUs */}
      <div>
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-medium text-white">
            ONU Sinyal Lemah ({degradedTotal})
          </h3>
          <div className="relative max-w-xs">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-500" />
            <input
              type="text"
              placeholder="Cari serial/customer..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full rounded-lg border border-gray-700 bg-gray-900 py-1.5 pl-10 pr-4 text-sm text-white placeholder-gray-500 focus:border-cyan-500 focus:outline-none"
            />
          </div>
        </div>

        <div className="rounded-lg border border-gray-800 bg-gray-900/50 overflow-hidden">
          <table className="w-full text-sm">
            <thead className="border-b border-gray-800 bg-gray-900">
              <tr>
                <th className="px-4 py-2 text-left text-gray-400">Serial Number</th>
                <th className="px-4 py-2 text-left text-gray-400">OLT</th>
                <th className="px-4 py-2 text-left text-gray-400">Port</th>
                <th className="px-4 py-2 text-left text-gray-400">Pelanggan</th>
                <th className="px-4 py-2 text-right text-gray-400">RX Power</th>
                <th className="px-4 py-2 text-right text-gray-400">TX Power</th>
                <th className="px-4 py-2 text-left text-gray-400">Last Seen</th>
              </tr>
            </thead>
            <tbody>
              {filteredDegraded.map((o) => (
                <tr key={o.id} className="border-b border-gray-800/50">
                  <td className="px-4 py-2 text-white font-mono text-xs">{o.serialNumber || '-'}</td>
                  <td className="px-4 py-2 text-gray-300">{o.olt?.name || '-'}</td>
                  <td className="px-4 py-2 text-gray-400">{o.frame}/{o.slot}/{o.port}:{o.onuId}</td>
                  <td className="px-4 py-2 text-gray-300">
                    {o.customer ? `${o.customer.name} (${o.customer.username})` : '-'}
                  </td>
                  <td className={`px-4 py-2 text-right font-medium ${signalColor(o.rxPower)}`}>
                    {formatDbm(o.rxPower)}
                  </td>
                  <td className="px-4 py-2 text-right text-gray-400">{formatDbm(o.txPower)}</td>
                  <td className="px-4 py-2 text-gray-400 text-xs">
                    {o.lastSeenAt ? formatWIB(o.lastSeenAt) : '-'}
                  </td>
                </tr>
              ))}
              {filteredDegraded.length === 0 && (
                <tr>
                  <td colSpan={7} className="px-4 py-8 text-center text-gray-500">
                    Tidak ada ONU dengan sinyal lemah
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>

        {degradedTotal > 50 && (
          <div className="flex items-center justify-between mt-3">
            <p className="text-sm text-gray-400">Total: {degradedTotal}</p>
            <div className="flex items-center gap-2">
              <button
                onClick={() => setPage(p => Math.max(1, p - 1))}
                disabled={page === 1}
                className="rounded-lg border border-gray-800 px-3 py-1 text-sm text-gray-400 hover:text-white disabled:opacity-30"
              >
                Prev
              </button>
              <span className="text-sm text-gray-400">Page {page}</span>
              <button
                onClick={() => setPage(p => p + 1)}
                disabled={page * 50 >= degradedTotal}
                className="rounded-lg border border-gray-800 px-3 py-1 text-sm text-gray-400 hover:text-white disabled:opacity-30"
              >
                Next
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
