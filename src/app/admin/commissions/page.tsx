'use client';

import { useState, useEffect } from 'react';
import { showSuccess, showError, showConfirm } from '@/lib/sweetalert';
import { Plus, Pencil, Trash2, Gift, Search, CheckCircle2, XCircle } from 'lucide-react';
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

interface Commission {
  id: string;
  employeeId: string;
  type: string;
  amount: number;
  description: string;
  status: string;
  approvedBy?: string;
  approvedAt?: string;
  paidAt?: string;
  createdAt: string;
}

const TYPES = ['INSTALLATION', 'SALES', 'REFERRAL', 'BONUS', 'OTHER'];
const STATUS_COLORS: Record<string, string> = {
  PENDING: 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-400',
  APPROVED: 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-400',
  REJECTED: 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400',
  PAID: 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400',
};

const fmt = (n: number) => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(n);

export default function CommissionsPage() {
  const [commissions, setCommissions] = useState<Commission[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [filterStatus, setFilterStatus] = useState('');
  const [isOpen, setIsOpen] = useState(false);
  const [editing, setEditing] = useState<Commission | null>(null);
  const [form, setForm] = useState({ employeeId: '', type: 'INSTALLATION', amount: '', description: '' });

  const load = async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams({ page: '1', limit: '100' });
      if (filterStatus) params.set('status', filterStatus);
      const res = await fetch(`/api/admin/commissions?${params}`);
      const data = await res.json();
      setCommissions(data.commissions || []);
    } catch {
      showError('Gagal', 'Gagal memuat data komisi');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, [filterStatus]);

  const openCreate = () => {
    setEditing(null);
    setForm({ employeeId: '', type: 'INSTALLATION', amount: '', description: '' });
    setIsOpen(true);
  };

  const openEdit = (c: Commission) => {
    setEditing(c);
    setForm({ employeeId: c.employeeId, type: c.type, amount: String(c.amount), description: c.description });
    setIsOpen(true);
  };

  const handleSave = async () => {
    if (!form.employeeId.trim()) return showError('Error', 'ID Karyawan wajib diisi');
    if (!form.amount || isNaN(Number(form.amount))) return showError('Error', 'Jumlah tidak valid');
    try {
      const url = editing ? `/api/admin/commissions/${editing.id}` : '/api/admin/commissions';
      const method = editing ? 'PUT' : 'POST';
      const res = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...form, amount: Number(form.amount) }),
      });
      if (!res.ok) throw new Error();
      showSuccess('Berhasil', editing ? 'Komisi diperbarui' : 'Komisi ditambahkan');
      setIsOpen(false);
      load();
    } catch {
      showError('Gagal', 'Gagal menyimpan komisi');
    }
  };

  const handleApprove = async (id: string) => {
    const ok = await showConfirm('Setujui Komisi?', 'Status akan berubah menjadi APPROVED');
    if (!ok) return;
    try {
      await fetch(`/api/admin/commissions/${id}/approve`, { method: 'POST' });
      showSuccess('Disetujui', 'Komisi disetujui');
      load();
    } catch { showError('Gagal', 'Gagal menyetujui'); }
  };

  const handleReject = async (id: string) => {
    const ok = await showConfirm('Tolak Komisi?', 'Status akan berubah menjadi REJECTED');
    if (!ok) return;
    try {
      await fetch(`/api/admin/commissions/${id}/reject`, { method: 'POST' });
      showSuccess('Ditolak', 'Komisi ditolak');
      load();
    } catch { showError('Gagal', 'Gagal menolak'); }
  };

  const handleDelete = async (id: string) => {
    const ok = await showConfirm('Hapus Komisi?', 'Data tidak bisa dikembalikan');
    if (!ok) return;
    try {
      await fetch(`/api/admin/commissions/${id}`, { method: 'DELETE' });
      showSuccess('Dihapus', 'Komisi dihapus');
      load();
    } catch { showError('Gagal', 'Gagal menghapus'); }
  };

  const filtered = commissions.filter(c =>
    c.employeeId.toLowerCase().includes(search.toLowerCase()) ||
    c.type.toLowerCase().includes(search.toLowerCase()) ||
    c.description.toLowerCase().includes(search.toLowerCase())
  );

  const totalPending = commissions.filter(c => c.status === 'PENDING').reduce((s, c) => s + c.amount, 0);
  const totalApproved = commissions.filter(c => c.status === 'APPROVED').reduce((s, c) => s + c.amount, 0);

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="p-2 bg-purple-500/10 rounded-xl">
            <Gift className="w-5 h-5 text-purple-500" />
          </div>
          <div>
            <h1 className="text-xl font-bold text-gray-900 dark:text-white">Komisi Karyawan</h1>
            <p className="text-sm text-gray-500 dark:text-gray-400">Kelola komisi instalasi, penjualan, dan referral</p>
          </div>
        </div>
        <button onClick={openCreate} className="flex items-center gap-2 px-4 py-2 bg-brand-500 hover:bg-brand-600 text-white rounded-xl text-sm font-semibold transition-colors">
          <Plus className="w-4 h-4" /> Tambah Komisi
        </button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-2 gap-4">
        <div className="bg-white dark:bg-gray-800/50 rounded-xl border border-gray-200 dark:border-gray-700 p-4">
          <p className="text-xs text-gray-500 dark:text-gray-400 mb-1">Total Pending</p>
          <p className="text-xl font-bold text-yellow-600 dark:text-yellow-400">{fmt(totalPending)}</p>
        </div>
        <div className="bg-white dark:bg-gray-800/50 rounded-xl border border-gray-200 dark:border-gray-700 p-4">
          <p className="text-xs text-gray-500 dark:text-gray-400 mb-1">Total Disetujui</p>
          <p className="text-xl font-bold text-blue-600 dark:text-blue-400">{fmt(totalApproved)}</p>
        </div>
      </div>

      {/* Filters */}
      <div className="flex gap-3 flex-wrap">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <input value={search} onChange={e => setSearch(e.target.value)} placeholder="Cari karyawan / tipe..."
            className="pl-9 pr-3 py-2 text-sm border border-gray-200 dark:border-gray-700 rounded-xl bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-brand-500" />
        </div>
        <select value={filterStatus} onChange={e => setFilterStatus(e.target.value)}
          className="px-3 py-2 text-sm border border-gray-200 dark:border-gray-700 rounded-xl bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-brand-500">
          <option value="">Semua Status</option>
          <option value="PENDING">Pending</option>
          <option value="APPROVED">Approved</option>
          <option value="REJECTED">Rejected</option>
          <option value="PAID">Paid</option>
        </select>
      </div>

      {/* Table */}
      {loading ? (
        <div className="text-center py-12 text-gray-400">Memuat data...</div>
      ) : filtered.length === 0 ? (
        <div className="text-center py-12 text-gray-400">
          <Gift className="w-10 h-10 mx-auto mb-2 opacity-30" />
          <p>Belum ada data komisi</p>
        </div>
      ) : (
        <div className="bg-white dark:bg-gray-800/50 rounded-xl border border-gray-200 dark:border-gray-700 overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800">
                <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-400">Karyawan</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-400">Tipe</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-400">Jumlah</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-400">Status</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-400">Tanggal</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-400">Aksi</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
              {filtered.map(c => (
                <tr key={c.id} className="hover:bg-gray-50 dark:hover:bg-gray-700/30 transition-colors">
                  <td className="px-4 py-3 font-mono text-xs text-gray-700 dark:text-gray-300">{c.employeeId}</td>
                  <td className="px-4 py-3">
                    <span className="px-2 py-0.5 bg-purple-100 dark:bg-purple-900/30 text-purple-700 dark:text-purple-400 rounded-full text-xs font-medium">
                      {c.type}
                    </span>
                  </td>
                  <td className="px-4 py-3 font-semibold text-gray-900 dark:text-white">{fmt(c.amount)}</td>
                  <td className="px-4 py-3">
                    <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${STATUS_COLORS[c.status] || ''}`}>{c.status}</span>
                  </td>
                  <td className="px-4 py-3 text-gray-500 dark:text-gray-400 text-xs">{new Date(c.createdAt).toLocaleDateString('id-ID')}</td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-1">
                      {c.status === 'PENDING' && (
                        <>
                          <button onClick={() => handleApprove(c.id)} title="Setujui" className="p-1.5 text-gray-400 hover:text-green-500 hover:bg-green-50 dark:hover:bg-green-500/10 rounded-lg transition-colors">
                            <CheckCircle2 className="w-3.5 h-3.5" />
                          </button>
                          <button onClick={() => handleReject(c.id)} title="Tolak" className="p-1.5 text-gray-400 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-500/10 rounded-lg transition-colors">
                            <XCircle className="w-3.5 h-3.5" />
                          </button>
                        </>
                      )}
                      <button onClick={() => openEdit(c)} className="p-1.5 text-gray-400 hover:text-brand-500 hover:bg-brand-50 dark:hover:bg-brand-500/10 rounded-lg transition-colors">
                        <Pencil className="w-3.5 h-3.5" />
                      </button>
                      <button onClick={() => handleDelete(c.id)} className="p-1.5 text-gray-400 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-500/10 rounded-lg transition-colors">
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Modal */}
      <SimpleModal isOpen={isOpen} onClose={() => setIsOpen(false)}>
        <ModalHeader><ModalTitle>{editing ? 'Edit Komisi' : 'Tambah Komisi'}</ModalTitle></ModalHeader>
        <ModalBody>
          <div className="space-y-4">
            <div>
              <ModalLabel>ID Karyawan *</ModalLabel>
              <ModalInput value={form.employeeId} onChange={e => setForm(f => ({ ...f, employeeId: e.target.value }))} placeholder="ID Karyawan" />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <ModalLabel>Tipe</ModalLabel>
                <ModalSelect value={form.type} onChange={e => setForm(f => ({ ...f, type: e.target.value }))}>
                  {TYPES.map(t => <option key={t} value={t}>{t}</option>)}
                </ModalSelect>
              </div>
              <div>
                <ModalLabel>Jumlah (Rp) *</ModalLabel>
                <ModalInput type="number" value={form.amount} onChange={e => setForm(f => ({ ...f, amount: e.target.value }))} placeholder="100000" />
              </div>
            </div>
            <div>
              <ModalLabel>Deskripsi</ModalLabel>
              <ModalTextarea value={form.description} onChange={e => setForm(f => ({ ...f, description: e.target.value }))} rows={2} placeholder="Keterangan komisi..." />
            </div>
          </div>
        </ModalBody>
        <ModalFooter>
          <ModalButton variant="secondary" onClick={() => setIsOpen(false)}>Batal</ModalButton>
          <ModalButton variant="primary" onClick={handleSave}>Simpan</ModalButton>
        </ModalFooter>
      </SimpleModal>
    </div>
  );
}
