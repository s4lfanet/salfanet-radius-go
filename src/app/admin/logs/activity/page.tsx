'use client';

import { useState, useEffect, useCallback, type ReactNode } from 'react';
import { useTranslation } from '@/hooks/useTranslation';
import { Activity, Search, RefreshCw, Filter, Shield, User, Globe, Zap, Database, MessageCircle, Settings, Package, CreditCard, ChevronLeft, ChevronRight, Clock } from 'lucide-react';
import { formatWIB } from '@/lib/timezone';

interface ActivityLog {
  id: string;
  username: string;
  userRole: string | null;
  action: string;
  description: string;
  module: string;
  status: 'success' | 'warning' | 'error';
  ipAddress: string | null;
  createdAt: string;
}

const MODULE_OPTIONS = [
  { value: 'all', label: 'Semua Modul' },
  { value: 'auth', label: 'Auth' },
  { value: 'pppoe', label: 'PPPoE' },
  { value: 'hotspot', label: 'Hotspot' },
  { value: 'voucher', label: 'Voucher' },
  { value: 'invoice', label: 'Invoice' },
  { value: 'payment', label: 'Pembayaran' },
  { value: 'agent', label: 'Agen' },
  { value: 'session', label: 'Sesi' },
  { value: 'transaction', label: 'Transaksi' },
  { value: 'system', label: 'Sistem' },
  { value: 'network', label: 'Jaringan' },
  { value: 'whatsapp', label: 'WhatsApp' },
  { value: 'genieacs', label: 'GenieACS' },
  { value: 'settings', label: 'Pengaturan' },
  { value: 'user', label: 'Pengguna' },
];

const MODULE_ICONS: Record<string, ReactNode> = {
  auth: <Shield className="w-3.5 h-3.5" />,
  pppoe: <Globe className="w-3.5 h-3.5" />,
  hotspot: <Zap className="w-3.5 h-3.5" />,
  voucher: <CreditCard className="w-3.5 h-3.5" />,
  invoice: <CreditCard className="w-3.5 h-3.5" />,
  payment: <CreditCard className="w-3.5 h-3.5" />,
  agent: <User className="w-3.5 h-3.5" />,
  session: <Clock className="w-3.5 h-3.5" />,
  transaction: <CreditCard className="w-3.5 h-3.5" />,
  system: <Database className="w-3.5 h-3.5" />,
  network: <Globe className="w-3.5 h-3.5" />,
  whatsapp: <MessageCircle className="w-3.5 h-3.5" />,
  genieacs: <Settings className="w-3.5 h-3.5" />,
  settings: <Settings className="w-3.5 h-3.5" />,
  user: <User className="w-3.5 h-3.5" />,
};

const STATUS_STYLES: Record<string, string> = {
  success: 'bg-emerald-500/15 text-emerald-400 border border-emerald-500/30',
  warning: 'bg-amber-500/15 text-amber-400 border border-amber-500/30',
  error: 'bg-red-500/15 text-red-400 border border-red-500/30',
};

const MODULE_STYLES: Record<string, string> = {
  auth: 'bg-purple-500/15 text-purple-400 border border-purple-500/30',
  pppoe: 'bg-blue-500/15 text-blue-400 border border-blue-500/30',
  hotspot: 'bg-yellow-500/15 text-yellow-400 border border-yellow-500/30',
  voucher: 'bg-pink-500/15 text-pink-400 border border-pink-500/30',
  invoice: 'bg-indigo-500/15 text-indigo-400 border border-indigo-500/30',
  payment: 'bg-emerald-500/15 text-emerald-400 border border-emerald-500/30',
  agent: 'bg-orange-500/15 text-orange-400 border border-orange-500/30',
  session: 'bg-cyan-500/15 text-cyan-400 border border-cyan-500/30',
  transaction: 'bg-teal-500/15 text-teal-400 border border-teal-500/30',
  system: 'bg-gray-500/15 text-gray-400 border border-gray-500/30',
  network: 'bg-sky-500/15 text-sky-400 border border-sky-500/30',
  whatsapp: 'bg-green-500/15 text-green-400 border border-green-500/30',
  genieacs: 'bg-violet-500/15 text-violet-400 border border-violet-500/30',
  settings: 'bg-slate-500/15 text-slate-400 border border-slate-500/30',
  user: 'bg-rose-500/15 text-rose-400 border border-rose-500/30',
};

const LIMIT = 25;

export default function ActivityLogsPage() {
  const { t } = useTranslation();
  const [logs, setLogs] = useState<ActivityLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(0);
  const [module, setModule] = useState('all');
  const [search, setSearch] = useState('');
  const [searchInput, setSearchInput] = useState('');

  const loadLogs = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams({
        module,
        limit: String(LIMIT),
        offset: String(page * LIMIT),
      });
      if (search) params.set('search', search);

      const res = await fetch(`/api/admin/activity-logs?${params}`);
      const data = await res.json();
      if (data.success) {
        setLogs(data.activities);
        setTotal(data.total);
      }
    } catch (err) {
      console.error('Load activity logs error:', err);
    } finally {
      setLoading(false);
    }
  }, [module, search, page]);

  useEffect(() => {
    loadLogs();
  }, [loadLogs]);

  const handleSearch = () => {
    setSearch(searchInput);
    setPage(0);
  };

  const handleModuleChange = (v: string) => {
    setModule(v);
    setPage(0);
  };

  const totalPages = Math.ceil(total / LIMIT);

  return (
    <div className="space-y-6 p-6">
      {/* Background decoration */}
      <div className="fixed inset-0 overflow-hidden pointer-events-none -z-10">
        <div className="absolute top-1/4 left-1/4 w-96 h-96 bg-[#bc13fe]/5 rounded-full blur-3xl" />
        <div className="absolute bottom-1/4 right-1/4 w-96 h-96 bg-[#00f7ff]/5 rounded-full blur-3xl" />
      </div>

      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="p-2.5 rounded-xl bg-[#bc13fe]/20 border border-[#bc13fe]/30">
            <Activity className="w-6 h-6 text-[#bc13fe]" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-foreground">Log Aktivitas</h1>
            <p className="text-sm text-muted-foreground">Riwayat aktivitas pengguna dan sistem</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-sm text-muted-foreground">{total.toLocaleString('id-ID')} entri</span>
          <button
            onClick={loadLogs}
            disabled={loading}
            className="p-2 rounded-lg bg-muted border border-border hover:bg-accent transition-colors disabled:opacity-50"
          >
            <RefreshCw className={`w-4 h-4 text-muted-foreground ${loading ? 'animate-spin' : ''}`} />
          </button>
        </div>
      </div>

      {/* Filters */}
      <div className="flex flex-col sm:flex-row gap-3">
        {/* Search */}
        <div className="flex flex-1 items-center gap-2 px-3 py-2 bg-muted border border-border rounded-xl focus-within:border-[#00f7ff]/50 transition-colors">
          <Search className="w-4 h-4 text-muted-foreground shrink-0" />
          <input
            className="flex-1 bg-transparent text-sm text-foreground placeholder:text-muted-foreground outline-none"
            placeholder="Cari username, aksi, IP..."
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
          />
          {searchInput && (
            <button
              onClick={() => { setSearchInput(''); setSearch(''); setPage(0); }}
              className="text-muted-foreground hover:text-foreground text-xs"
            >
              ✕
            </button>
          )}
        </div>
        <button
          onClick={handleSearch}
          className="px-4 py-2 bg-[#00f7ff] text-[#1a0f35] font-bold text-sm rounded-xl hover:shadow-[0_0_15px_rgba(0,247,255,0.3)] transition-all"
        >
          Cari
        </button>

        {/* Module filter */}
        <div className="flex items-center gap-2 px-3 py-2 bg-muted border border-border rounded-xl">
          <Filter className="w-4 h-4 text-muted-foreground shrink-0" />
          <select
            value={module}
            onChange={(e) => handleModuleChange(e.target.value)}
            className="bg-transparent text-sm text-foreground outline-none cursor-pointer"
          >
            {MODULE_OPTIONS.map((opt) => (
              <option key={opt.value} value={opt.value} className="bg-[#1e1b2e]">
                {opt.label}
              </option>
            ))}
          </select>
        </div>
      </div>

      {/* Table */}
      <div className="bg-card border border-border rounded-xl overflow-hidden">
        {loading ? (
          <div className="flex items-center justify-center py-20">
            <div className="flex flex-col items-center gap-3">
              <div className="w-10 h-10 border-4 border-[#00f7ff] border-t-transparent rounded-full animate-spin" />
              <p className="text-sm text-muted-foreground">Memuat log aktivitas...</p>
            </div>
          </div>
        ) : logs.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-20 gap-3">
            <Activity className="w-12 h-12 text-muted-foreground/30" />
            <p className="text-muted-foreground">Tidak ada log aktivitas ditemukan</p>
          </div>
        ) : (
          <>
            {/* Desktop Table */}
            <div className="hidden md:block overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-muted/50">
                    <th className="px-4 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">Waktu</th>
                    <th className="px-4 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">Pengguna</th>
                    <th className="px-4 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">Modul</th>
                    <th className="px-4 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">Aksi</th>
                    <th className="px-4 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">Deskripsi</th>
                    <th className="px-4 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">Status</th>
                    <th className="px-4 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">IP</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {logs.map((log) => (
                    <tr key={log.id} className="hover:bg-muted/30 transition-colors">
                      <td className="px-4 py-3 whitespace-nowrap">
                        <span className="font-mono text-xs text-muted-foreground">
                          {formatWIB(log.createdAt, 'dd MMM yyyy HH:mm:ss')}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <div>
                          <p className="font-medium text-foreground">{log.username}</p>
                          {log.userRole && (
                            <p className="text-xs text-muted-foreground capitalize">{log.userRole}</p>
                          )}
                        </div>
                      </td>
                      <td className="px-4 py-3">
                        <span className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium ${MODULE_STYLES[log.module] || 'bg-gray-500/15 text-gray-400 border border-gray-500/30'}`}>
                          {MODULE_ICONS[log.module]}
                          {log.module}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <span className="font-mono text-xs text-[#00f7ff]">{log.action}</span>
                      </td>
                      <td className="px-4 py-3 max-w-xs">
                        <p className="text-sm text-foreground truncate" title={log.description}>
                          {log.description}
                        </p>
                      </td>
                      <td className="px-4 py-3">
                        <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${STATUS_STYLES[log.status] || STATUS_STYLES.success}`}>
                          {log.status === 'success' ? '✓' : log.status === 'warning' ? '⚠' : '✗'} {log.status}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <span className="font-mono text-xs text-muted-foreground">{log.ipAddress || '-'}</span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {/* Mobile Cards */}
            <div className="md:hidden divide-y divide-border">
              {logs.map((log) => (
                <div key={log.id} className="p-4 space-y-2">
                  <div className="flex items-start justify-between gap-2">
                    <div>
                      <p className="font-medium text-foreground text-sm">{log.username}</p>
                      {log.userRole && <p className="text-xs text-muted-foreground capitalize">{log.userRole}</p>}
                    </div>
                    <div className="flex items-center gap-2 shrink-0">
                      <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${STATUS_STYLES[log.status] || STATUS_STYLES.success}`}>
                        {log.status}
                      </span>
                    </div>
                  </div>
                  <p className="text-sm text-foreground">{log.description}</p>
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium ${MODULE_STYLES[log.module] || 'bg-gray-500/15 text-gray-400 border border-gray-500/30'}`}>
                      {MODULE_ICONS[log.module]}
                      {log.module}
                    </span>
                    <span className="font-mono text-xs text-[#00f7ff]">{log.action}</span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="font-mono text-xs text-muted-foreground">{log.ipAddress || '-'}</span>
                    <span className="font-mono text-xs text-muted-foreground">
                      {formatWIB(log.createdAt, 'dd MMM HH:mm:ss')}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </>
        )}
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between">
          <p className="text-sm text-muted-foreground">
            Menampilkan {page * LIMIT + 1}–{Math.min((page + 1) * LIMIT, total)} dari {total.toLocaleString('id-ID')} entri
          </p>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setPage((p) => Math.max(0, p - 1))}
              disabled={page === 0 || loading}
              className="p-2 rounded-lg bg-muted border border-border hover:bg-accent transition-colors disabled:opacity-40"
            >
              <ChevronLeft className="w-4 h-4" />
            </button>
            <span className="text-sm text-muted-foreground px-2">
              {page + 1} / {totalPages}
            </span>
            <button
              onClick={() => setPage((p) => Math.min(totalPages - 1, p + 1))}
              disabled={page >= totalPages - 1 || loading}
              className="p-2 rounded-lg bg-muted border border-border hover:bg-accent transition-colors disabled:opacity-40"
            >
              <ChevronRight className="w-4 h-4" />
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
