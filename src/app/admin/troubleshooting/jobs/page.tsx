'use client';

import { useState, useEffect } from 'react';
import { showSuccess, showError } from '@/lib/sweetalert';
import { Plus, Wrench, Search, Clock, AlertCircle, CheckCircle2, ArrowUpRight } from 'lucide-react';
import Link from 'next/link';
import {
  SimpleModal,
  ModalHeader,
  ModalTitle,
  ModalBody,
  ModalFooter,
  ModalInput,
  ModalSelect,
  ModalTextarea,
  ModalLabel,
  ModalButton,
} from '@/components/cyberpunk';

interface Job {
  id: string;
  title: string;
  description?: string;
  checklistId?: string;
  assignedToId?: string;
  status: string;
  priority: string;
  createdAt: string;
  updatedAt: string;
}

const STATUS_COLORS: Record<string, string> = {
  OPEN: 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-400',
  IN_PROGRESS: 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-400',
  RESOLVED: 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400',
  CLOSED: 'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400',
};

const PRIORITY_COLORS: Record<string, string> = {
  HIGH: 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400',
  MEDIUM: 'bg-orange-100 dark:bg-orange-900/30 text-orange-700 dark:text-orange-400',
  LOW: 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400',
};

export default function TroubleshootingJobsPage() {
  const [jobs, setJobs] = useState<Job[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [filterStatus, setFilterStatus] = useState('');
  const [isOpen, setIsOpen] = useState(false);
  const [form, setForm] = useState({ title: '', description: '', status: 'OPEN', priority: 'MEDIUM' });

  const load = async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams({ page: '1', limit: '50' });
      if (filterStatus) params.set('status', filterStatus);
      const res = await fetch(`/api/troubleshooting/jobs?${params}`);
      const data = await res.json();
      setJobs(data.jobs || []);
    } catch {
      showError('Gagal', 'Gagal memuat data jobs');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, [filterStatus]);

  const handleCreate = async () => {
    if (!form.title.trim()) return showError('Error', 'Judul wajib diisi');
    try {
      const res = await fetch('/api/troubleshooting/jobs', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(form),
      });
      if (!res.ok) throw new Error();
      showSuccess('Berhasil', 'Job ditambahkan');
      setIsOpen(false);
      load();
    } catch {
      showError('Gagal', 'Gagal membuat job');
    }
  };

  const filtered = jobs.filter(j =>
    j.title.toLowerCase().includes(search.toLowerCase())
  );

  const stats = {
    open: jobs.filter(j => j.status === 'OPEN').length,
    inProgress: jobs.filter(j => j.status === 'IN_PROGRESS').length,
    resolved: jobs.filter(j => j.status === 'RESOLVED').length,
  };

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="p-2 bg-brand-500/10 rounded-xl">
            <Wrench className="w-5 h-5 text-brand-500" />
          </div>
          <div>
            <h1 className="text-xl font-bold text-gray-900 dark:text-white">Troubleshooting Jobs</h1>
            <p className="text-sm text-gray-500 dark:text-gray-400">Lacak pekerjaan troubleshooting aktif</p>
          </div>
        </div>
        <button
          onClick={() => { setForm({ title: '', description: '', status: 'OPEN', priority: 'MEDIUM' }); setIsOpen(true); }}
          className="flex items-center gap-2 px-4 py-2 bg-brand-500 hover:bg-brand-600 text-white rounded-xl text-sm font-semibold transition-colors"
        >
          <Plus className="w-4 h-4" />
          Buat Job
        </button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-3 gap-4">
        {[
          { label: 'Open', count: stats.open, icon: AlertCircle, color: 'text-yellow-500', bg: 'bg-yellow-500/10' },
          { label: 'In Progress', count: stats.inProgress, icon: Clock, color: 'text-blue-500', bg: 'bg-blue-500/10' },
          { label: 'Resolved', count: stats.resolved, icon: CheckCircle2, color: 'text-green-500', bg: 'bg-green-500/10' },
        ].map(s => (
          <div key={s.label} className="bg-white dark:bg-gray-800/50 rounded-2xl border border-gray-200 dark:border-gray-700 p-4 flex items-center gap-3">
            <div className={`p-2 ${s.bg} rounded-xl`}>
              <s.icon className={`w-5 h-5 ${s.color}`} />
            </div>
            <div>
              <p className="text-2xl font-bold text-gray-900 dark:text-white">{s.count}</p>
              <p className="text-xs text-gray-500 dark:text-gray-400">{s.label}</p>
            </div>
          </div>
        ))}
      </div>

      {/* Filters */}
      <div className="flex gap-3 flex-wrap">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <input
            value={search}
            onChange={e => setSearch(e.target.value)}
            placeholder="Cari job..."
            className="pl-9 pr-3 py-2 text-sm border border-gray-200 dark:border-gray-700 rounded-xl bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-brand-500"
          />
        </div>
        <select
          value={filterStatus}
          onChange={e => setFilterStatus(e.target.value)}
          className="px-3 py-2 text-sm border border-gray-200 dark:border-gray-700 rounded-xl bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-brand-500"
        >
          <option value="">Semua Status</option>
          <option value="OPEN">Open</option>
          <option value="IN_PROGRESS">In Progress</option>
          <option value="RESOLVED">Resolved</option>
          <option value="CLOSED">Closed</option>
        </select>
      </div>

      {/* Table */}
      {loading ? (
        <div className="text-center py-12 text-gray-400">Memuat data...</div>
      ) : filtered.length === 0 ? (
        <div className="text-center py-12 text-gray-400">
          <Wrench className="w-10 h-10 mx-auto mb-2 opacity-30" />
          <p>Belum ada troubleshooting job</p>
        </div>
      ) : (
        <div className="bg-white dark:bg-gray-800/50 rounded-2xl border border-gray-200 dark:border-gray-700 overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800">
                <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-400">Judul</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-400">Status</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-400">Prioritas</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-400">Dibuat</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-400">Detail</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
              {filtered.map(j => (
                <tr key={j.id} className="hover:bg-gray-50 dark:hover:bg-gray-700/30 transition-colors">
                  <td className="px-4 py-3">
                    <p className="font-medium text-gray-900 dark:text-white">{j.title}</p>
                    {j.description && <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5 line-clamp-1">{j.description}</p>}
                  </td>
                  <td className="px-4 py-3">
                    <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${STATUS_COLORS[j.status] || 'bg-gray-100 text-gray-600'}`}>
                      {j.status.replace('_', ' ')}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${PRIORITY_COLORS[j.priority] || 'bg-gray-100 text-gray-600'}`}>
                      {j.priority}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-gray-500 dark:text-gray-400">
                    {new Date(j.createdAt).toLocaleDateString('id-ID')}
                  </td>
                  <td className="px-4 py-3">
                    <Link href={`/admin/troubleshooting/jobs/${j.id}`} className="p-1.5 text-gray-400 hover:text-brand-500 hover:bg-brand-50 dark:hover:bg-brand-500/10 rounded-lg transition-colors inline-flex">
                      <ArrowUpRight className="w-3.5 h-3.5" />
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Create Modal */}
      <SimpleModal isOpen={isOpen} onClose={() => setIsOpen(false)}>
        <ModalHeader>
          <ModalTitle>Buat Troubleshooting Job</ModalTitle>
        </ModalHeader>
        <ModalBody>
          <div className="space-y-4">
            <div>
              <ModalLabel>Judul *</ModalLabel>
              <ModalInput value={form.title} onChange={e => setForm(f => ({ ...f, title: e.target.value }))} placeholder="Judul job" />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <ModalLabel>Status</ModalLabel>
                <ModalSelect value={form.status} onChange={e => setForm(f => ({ ...f, status: e.target.value }))}>
                  <option value="OPEN">Open</option>
                  <option value="IN_PROGRESS">In Progress</option>
                  <option value="RESOLVED">Resolved</option>
                  <option value="CLOSED">Closed</option>
                </ModalSelect>
              </div>
              <div>
                <ModalLabel>Prioritas</ModalLabel>
                <ModalSelect value={form.priority} onChange={e => setForm(f => ({ ...f, priority: e.target.value }))}>
                  <option value="HIGH">High</option>
                  <option value="MEDIUM">Medium</option>
                  <option value="LOW">Low</option>
                </ModalSelect>
              </div>
            </div>
            <div>
              <ModalLabel>Deskripsi</ModalLabel>
              <ModalTextarea value={form.description} onChange={e => setForm(f => ({ ...f, description: e.target.value }))} placeholder="Detail masalah..." rows={3} />
            </div>
          </div>
        </ModalBody>
        <ModalFooter>
          <ModalButton variant="secondary" onClick={() => setIsOpen(false)}>Batal</ModalButton>
          <ModalButton variant="primary" onClick={handleCreate}>Buat Job</ModalButton>
        </ModalFooter>
      </SimpleModal>
    </div>
  );
}
