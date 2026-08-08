'use client';

import { useState, useEffect } from 'react';
import { Loader2, Package, Wrench, Search, ChevronLeft, ChevronRight } from 'lucide-react';
import { usePermissions } from '@/hooks/usePermissions';
import { formatWIB } from '@/lib/timezone';

interface PackageChangeLog {
  id: string;
  userId: string;
  username: string;
  oldProfileName: string | null;
  newProfileName: string | null;
  changedByName: string | null;
  reason: string | null;
  changedAt: string;
}

interface InstallationLog {
  id: string;
  userId: string;
  username: string;
  fullname: string | null;
  phone: string | null;
  address: string | null;
  profileName: string | null;
  territoryName: string | null;
  installerName: string | null;
  installDate: string;
  createdAt: string;
}

export default function ActivityLogsPage() {
  const { loading: permLoading } = usePermissions();
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<'package' | 'installation'>('package');
  const [packageLogs, setPackageLogs] = useState<PackageChangeLog[]>([]);
  const [installLogs, setInstallLogs] = useState<InstallationLog[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(50);
  const [search, setSearch] = useState('');

  useEffect(() => {
    loadData();
  }, [activeTab, page]);

  const loadData = async () => {
    setLoading(true);
    try {
      if (activeTab === 'package') {
        const res = await fetch(`/api/package-change-logs?page=${page}&pageSize=${pageSize}`);
        const data = await res.json();
        setPackageLogs(data.data || []);
        setTotal(data.total || 0);
      } else {
        const res = await fetch(`/api/installation-logs?page=${page}&pageSize=${pageSize}`);
        const data = await res.json();
        setInstallLogs(data.data || []);
        setTotal(data.total || 0);
      }
    } catch (error) {
      console.error('Load logs error:', error);
    } finally {
      setLoading(false);
    }
  };

  const filteredLogs: any[] = activeTab === 'package'
    ? packageLogs.filter(l =>
        l.username.toLowerCase().includes(search.toLowerCase()) ||
        (l.oldProfileName || '').toLowerCase().includes(search.toLowerCase()) ||
        (l.newProfileName || '').toLowerCase().includes(search.toLowerCase())
      )
    : installLogs.filter(l =>
        l.username.toLowerCase().includes(search.toLowerCase()) ||
        (l.fullname || '').toLowerCase().includes(search.toLowerCase()) ||
        (l.installerName || '').toLowerCase().includes(search.toLowerCase())
      );

  const totalPages = Math.ceil(total / pageSize);

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
        <h1 className="text-2xl font-bold text-white">Log Aktivitas</h1>
        <p className="text-sm text-gray-400">Riwayat perubahan paket dan instalasi pelanggan</p>
      </div>

      <div className="flex gap-2 border-b border-gray-800">
        <button
          onClick={() => { setActiveTab('package'); setPage(1); }}
          className={`flex items-center gap-2 px-4 py-2 text-sm font-medium ${activeTab === 'package' ? 'border-b-2 border-cyan-500 text-cyan-400' : 'text-gray-400 hover:text-white'}`}
        >
          <Package className="h-4 w-4" />
          Perubahan Paket
        </button>
        <button
          onClick={() => { setActiveTab('installation'); setPage(1); }}
          className={`flex items-center gap-2 px-4 py-2 text-sm font-medium ${activeTab === 'installation' ? 'border-b-2 border-cyan-500 text-cyan-400' : 'text-gray-400 hover:text-white'}`}
        >
          <Wrench className="h-4 w-4" />
          Instalasi
        </button>
      </div>

      <div className="relative max-w-sm">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-500" />
        <input
          type="text"
          placeholder="Cari..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="w-full rounded-lg border border-gray-700 bg-gray-900 py-2 pl-10 pr-4 text-sm text-white placeholder-gray-500 focus:border-cyan-500 focus:outline-none"
        />
      </div>

      <div className="rounded-lg border border-gray-800 bg-gray-900/50 overflow-hidden">
        {activeTab === 'package' ? (
          <table className="w-full text-sm">
            <thead className="border-b border-gray-800 bg-gray-900">
              <tr>
                <th className="px-4 py-2 text-left text-gray-400">Username</th>
                <th className="px-4 py-2 text-left text-gray-400">Paket Lama</th>
                <th className="px-4 py-2 text-left text-gray-400">Paket Baru</th>
                <th className="px-4 py-2 text-left text-gray-400">Oleh</th>
                <th className="px-4 py-2 text-left text-gray-400">Alasan</th>
                <th className="px-4 py-2 text-left text-gray-400">Tanggal</th>
              </tr>
            </thead>
            <tbody>
              {filteredLogs.map((log) => (
                <tr key={log.id} className="border-b border-gray-800/50">
                  <td className="px-4 py-2 text-white font-mono">{log.username}</td>
                  <td className="px-4 py-2 text-gray-400">{log.oldProfileName || '-'}</td>
                  <td className="px-4 py-2 text-cyan-400">{log.newProfileName || '-'}</td>
                  <td className="px-4 py-2 text-gray-300">{log.changedByName || '-'}</td>
                  <td className="px-4 py-2 text-gray-400">{log.reason || '-'}</td>
                  <td className="px-4 py-2 text-gray-400">{formatWIB(log.changedAt)}</td>
                </tr>
              ))}
              {filteredLogs.length === 0 && (
                <tr><td colSpan={6} className="px-4 py-8 text-center text-gray-500">Tidak ada data</td></tr>
              )}
            </tbody>
          </table>
        ) : (
          <table className="w-full text-sm">
            <thead className="border-b border-gray-800 bg-gray-900">
              <tr>
                <th className="px-4 py-2 text-left text-gray-400">Username</th>
                <th className="px-4 py-2 text-left text-gray-400">Nama</th>
                <th className="px-4 py-2 text-left text-gray-400">Telepon</th>
                <th className="px-4 py-2 text-left text-gray-400">Paket</th>
                <th className="px-4 py-2 text-left text-gray-400">Wilayah</th>
                <th className="px-4 py-2 text-left text-gray-400">Installer</th>
                <th className="px-4 py-2 text-left text-gray-400">Tanggal</th>
              </tr>
            </thead>
            <tbody>
              {filteredLogs.map((log) => (
                <tr key={log.id} className="border-b border-gray-800/50">
                  <td className="px-4 py-2 text-white font-mono">{log.username}</td>
                  <td className="px-4 py-2 text-gray-300">{log.fullname || '-'}</td>
                  <td className="px-4 py-2 text-gray-400">{log.phone || '-'}</td>
                  <td className="px-4 py-2 text-cyan-400">{log.profileName || '-'}</td>
                  <td className="px-4 py-2 text-gray-400">{log.territoryName || '-'}</td>
                  <td className="px-4 py-2 text-gray-300">{log.installerName || '-'}</td>
                  <td className="px-4 py-2 text-gray-400">{formatWIB(log.installDate)}</td>
                </tr>
              ))}
              {filteredLogs.length === 0 && (
                <tr><td colSpan={7} className="px-4 py-8 text-center text-gray-500">Tidak ada data</td></tr>
              )}
            </tbody>
          </table>
        )}
      </div>

      {totalPages > 1 && (
        <div className="flex items-center justify-between">
          <p className="text-sm text-gray-400">Total: {total}</p>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setPage(p => Math.max(1, p - 1))}
              disabled={page === 1}
              className="rounded-lg border border-gray-800 p-1.5 text-gray-400 hover:text-white disabled:opacity-30"
            >
              <ChevronLeft className="h-4 w-4" />
            </button>
            <span className="text-sm text-gray-400">{page} / {totalPages}</span>
            <button
              onClick={() => setPage(p => Math.min(totalPages, p + 1))}
              disabled={page === totalPages}
              className="rounded-lg border border-gray-800 p-1.5 text-gray-400 hover:text-white disabled:opacity-30"
            >
              <ChevronRight className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
