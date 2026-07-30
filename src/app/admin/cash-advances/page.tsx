'use client';

import { useState, useEffect } from 'react';
import { showSuccess, showError, showConfirm } from '@/lib/sweetalert';
import { Plus, Pencil, Trash2, Wallet, Search, CheckCircle2, XCircle, DollarSign } from 'lucide-react';
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

interface CashAdvance {
  id: string;
  employeeId: string;
  amount: number;
  reason: string;
  status: string;
  approvedBy?: string;
  approvedAt?: string;
  paidAt?: string;
  installments: number;
  notes?: string;
  createdAt: string;
}

const STATUS_COLORS: Record<string, string> = {
  PENDING: 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-400',
  APPROVED: 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-400',
  REJECTED: 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400',
  PAID: 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400',
};

const fmt = (n: number) => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(n);

export default function CashAdvancesPage() {
  const [advances, setAdvances] = useState<CashAdvance[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [filterStatus, setFilterStatus] = useState('');
  const [isOpen, setIsOpen] = useState(false);
  const [editing, setEditing] = useState<CashAdvance | null>(null);
  const [form, setForm] = useState({ employeeId: '', amount: '', reason: '', installments: '1', notes: '' });

  const load = async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams({ page: '1', limit: '100' });
      if (filterStatus) params.set('status', filterStatus);
      const res = await fetch(`/api/admin/cash-advances?${params}`);
      const data = await res.json();
      setAdvances(data.advances || []);
    } catch {
      showError('Gagal', 'Gagal memuat data kasbon');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, [filterStatus]);

  const openCreate = () => {
    setEditing(null);
    setForm({ employeeId: '', amount: '', reason: '', installments: '1', notes: '' });
    setIsOpen(true);
  };

  const openEdit = (a: CashAdvance) => {
    setEditing(a);
    setForm({ employeeId: a.employeeId, amount: String(a.amount), reason: a.reason, installments: String(a.installments), notes: a.notes || '' });
    setIsOpen(true);
  };

  const handleSave = async () => {
    if (!form.employeeId.trim()) return showError('Error', 'ID Karyawan wajib diisi');
    if (!form.amount || isNaN(Number(form.amount))) return showError('Error', 'Jumlah tidak valid');
    try {
      const url = editing ? `/api/admin/cash-advances/${editing.id}` : '/api/admin/cash-advances';
      const method = editing ? 'PUT' : 'POST';
      const res = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...form, amount: Number(form.amount), installments: Number(form.installments) }),
      });
      if (!res.ok) throw new Error();
      showSuccess('Berhasil', editing ? 'Kasbon diperbarui' : 'Kasbon ditambahkan');
      setIsOpen(false);
      load();
    } catch {
      showError('Gagal', 'Gagal menyimpan kasbon');
    }
  };

  const handleDelete = async (id: string) => {
    const ok = await showConfirm('Hapus Kasbon?', 'Data tidak bisa dikembalikan');
    if (!ok) return;
    try {
      await fetch(`/api/admin/cash-advances/${id}`, { method: 'DELETE' });
      showSuccess('Dihapus', 'Kasbon dihapus');
      load();
    } catch {
      showError('Gagal', 'Gagal menghapus');
    }
  };

  const handlePay = async (id: string) => {
    const ok = await showConfirm('Tandai Lunas?', 'Status akan diubah menjadi PAID');
    if (!ok) return;
    try {
      await fetch(`/api/admin/cash-advances/pay/${id}`, { method: 'POST' });
      showSuccess('Lunas', 'Kasbon ditandai lunas');
      load();
    } catch {
      showError('Gagal', 'Gagal mengupdate status');
    }
  };

  const filtered = advances.filter(a =>
    a.employeeId.toLowerCase().includes(search.toLowerCase()) ||
    a.reason.toLowerCase().includes(search.toLowerCase())
  );

  const totalPending = advances.filter(a => a.status === 'PENDING').reduce((s, a) => s + a.amount, 0);
  const totalApproved = advances.filter(a => a.status === 'APPROVED').reduce((s, a) => s + a.amount, 0);

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="p-2 bg-orange-500/10 rounded-xl">
            <Wallet className="w-5 h-5 text-orange-500" />
          </div>
          <div>
            <h1 className="text-xl font-bold text-gray-900 dark:text-white">Kasbon Karyawan</h1>
            <p className="text-sm text-gray-500 dark:text-gray-400">Kelola pengajuan kasbon / uang muka</p>
          </div>
        </div>
        <button onClick={openCreate} className="flex items-center gap-2 px-4 py-2 bg-brand-500 hover:bg-brand-600 text-white rounded-xl text-sm font-semibold transition-colors">
          <Plus className="w-4 h-4" /> Tambah Kasbon
        </button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-2 gap-4">
        <div className="bg-white dark:bg-gray-800/50 rounded-xl border border-gray-200 dark:border-gray-700 p-4">
          <p className="text-xs text-gray-500 dark:text-gray-400 mb-1">Total Pending</p>
          <p className="text-xl font-bold text-yellow-600 dark:text-yellow-400">{fmt(totalPending)}</p>
        </div>
        <div className="bg-white dark:bg-gray-800/50 rounded-xl border border-gray-200 dark:border-gray-700 p-4">
          <p className="text-xs text-gray-500 dark:text-gray-400 mb-1">Total Disetujui (belum lunas)</p>
          <p className="text-xl font-bold text-blue-600 dark:text-blue-400">{fmt(totalApproved)}</p>
        </div>
      </div>

      {/* Filters */}
      <div className="flex gap-3 flex-wrap">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <input value={search} onChange={e => setSearch(e.target.value)} placeholder="Cari karyawan..."
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
          <Wallet className="w-10 h-10 mx-auto mb-2 opacity-30" />
          <p>Belum ada data kasbon</p>
        </div>
      ) : (
        <div className="bg-white dark:bg-gray-800/50 rounded-xl border border-gray-200 dark:border-gray-700 overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800">
                <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-400">Karyawan</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-400">Jumlah</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-400">Alasan</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-400">Status</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-400">Cicilan</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-400">Aksi</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
              {filtered.map(a => (
                <tr key={a.id} className="hover:bg-gray-50 dark:hover:bg-gray-700/30 transition-colors">
                  <td className="px-4 py-3 font-mono text-xs text-gray-700 dark:text-gray-300">{a.employeeId}</td>
                  <td className="px-4 py-3 font-semibold text-gray-900 dark:text-white">{fmt(a.amount)}</td>
                  <td className="px-4 py-3 text-gray-600 dark:text-gray-400 max-w-[200px] truncate">{a.reason}</td>
                  <td className="px-4 py-3">
                    <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${STATUS_COLORS[a.status] || ''}`}>{a.status}</span>
                  </td>
                  <td className="px-4 py-3 text-gray-600 dark:text-gray-400">{a.installments}x</td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-1">
                      {a.status === 'APPROVED' && (
                        <button onClick={() => handlePay(a.id)} title="Tandai Lunas" className="p-1.5 text-gray-400 hover:text-green-500 hover:bg-green-50 dark:hover:bg-green-500/10 rounded-lg transition-colors">
                          <CheckCircle2 className="w-3.5 h-3.5" />
                        </button>
                      )}
                      <button onClick={() => openEdit(a)} className="p-1.5 text-gray-400 hover:text-brand-500 hover:bg-brand-50 dark:hover:bg-brand-500/10 rounded-lg transition-colors">
                        <Pencil className="w-3.5 h-3.5" />
                      </button>
                      <button onClick={() => handleDelete(a.id)} className="p-1.5 text-gray-400 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-500/10 rounded-lg transition-colors">
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
        <ModalHeader><ModalTitle>{editing ? 'Edit Kasbon' : 'Tambah Kasbon'}</ModalTitle></ModalHeader>
        <ModalBody>
          <div className="space-y-4">
            <div>
              <ModalLabel>ID Karyawan *</ModalLabel>
              <ModalInput value={form.employeeId} onChange={e => setForm(f => ({ ...f, employeeId: e.target.value }))} placeholder="ID Karyawan" />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <ModalLabel>Jumlah (Rp) *</ModalLabel>
                <ModalInput type="number" value={form.amount} onChange={e => setForm(f => ({ ...f, amount: e.target.value }))} placeholder="500000" />
              </div>
              <div>
                <ModalLabel>Jumlah Cicilan</ModalLabel>
                <ModalInput type="number" min="1" value={form.installments} onChange={e => setForm(f => ({ ...f, installments: e.target.value }))} />
              </div>
            </div>
            <div>
              <ModalLabel>Alasan Kasbon *</ModalLabel>
              <ModalTextarea value={form.reason} onChange={e => setForm(f => ({ ...f, reason: e.target.value }))} rows={2} placeholder="Alasan pengajuan kasbon..." />
            </div>
            <div>
              <ModalLabel>Catatan</ModalLabel>
              <ModalTextarea value={form.notes} onChange={e => setForm(f => ({ ...f, notes: e.target.value }))} rows={2} placeholder="Catatan tambahan..." />
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
