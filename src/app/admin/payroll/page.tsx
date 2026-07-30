'use client';

import { useState, useEffect } from 'react';
import { showSuccess, showError, showConfirm } from '@/lib/sweetalert';
import { DollarSign, Plus, Pencil, Trash2, Search, CheckCircle2, RefreshCcw, Play } from 'lucide-react';
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

interface PayrollRecord {
  id: string;
  employeeId: string;
  month: string;
  baseWage: number;
  allowance: number;
  deduction: number;
  bonus: number;
  overtime: number;
  netAmount: number;
  status: string;
  paidAt?: string;
  notes?: string;
  createdAt: string;
}

const STATUS_COLORS: Record<string, string> = {
  DRAFT: 'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400',
  PAID: 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400',
  CANCELLED: 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400',
};

const fmt = (n: number) => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(n);

function currentMonth() {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
}

export default function PayrollPage() {
  const [records, setRecords] = useState<PayrollRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [filterMonth, setFilterMonth] = useState(currentMonth());
  const [filterStatus, setFilterStatus] = useState('');
  const [search, setSearch] = useState('');
  const [isOpen, setIsOpen] = useState(false);
  const [isGenerateOpen, setIsGenerateOpen] = useState(false);
  const [editing, setEditing] = useState<PayrollRecord | null>(null);
  const [generateMonth, setGenerateMonth] = useState(currentMonth());
  const [form, setForm] = useState({
    employeeId: '', month: currentMonth(),
    baseWage: '', allowance: '0', deduction: '0', bonus: '0', overtime: '0', notes: '',
  });

  const load = async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams({ page: '1', limit: '100' });
      if (filterMonth) params.set('month', filterMonth);
      if (filterStatus) params.set('status', filterStatus);
      const res = await fetch(`/api/admin/payroll?${params}`);
      const data = await res.json();
      setRecords(data.payroll || []);
    } catch {
      showError('Gagal', 'Gagal memuat data payroll');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, [filterMonth, filterStatus]);

  const openEdit = (r: PayrollRecord) => {
    setEditing(r);
    setForm({
      employeeId: r.employeeId, month: r.month,
      baseWage: String(r.baseWage), allowance: String(r.allowance),
      deduction: String(r.deduction), bonus: String(r.bonus),
      overtime: String(r.overtime), notes: r.notes || '',
    });
    setIsOpen(true);
  };

  const handleSave = async () => {
    if (!editing) return;
    try {
      const net = (Number(form.baseWage) || 0) + (Number(form.allowance) || 0) + (Number(form.bonus) || 0) + (Number(form.overtime) || 0) - (Number(form.deduction) || 0);
      const res = await fetch(`/api/admin/payroll/${editing.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          baseWage: Number(form.baseWage) || 0,
          allowance: Number(form.allowance) || 0,
          deduction: Number(form.deduction) || 0,
          bonus: Number(form.bonus) || 0,
          overtime: Number(form.overtime) || 0,
          netAmount: net,
          notes: form.notes,
        }),
      });
      if (!res.ok) throw new Error();
      showSuccess('Berhasil', 'Payroll diperbarui');
      setIsOpen(false);
      load();
    } catch {
      showError('Gagal', 'Gagal menyimpan payroll');
    }
  };

  const handleDelete = async (id: string) => {
    const ok = await showConfirm('Hapus Payroll?', 'Data tidak bisa dikembalikan');
    if (!ok) return;
    try {
      await fetch(`/api/admin/payroll/${id}`, { method: 'DELETE' });
      showSuccess('Dihapus', 'Data payroll dihapus');
      load();
    } catch { showError('Gagal', 'Gagal menghapus'); }
  };

  const handlePay = async (id: string) => {
    const ok = await showConfirm('Tandai Lunas?', 'Status payroll akan diubah menjadi PAID');
    if (!ok) return;
    try {
      await fetch(`/api/admin/payroll/pay/${id}`, { method: 'POST' });
      showSuccess('Lunas', 'Payroll ditandai lunas');
      load();
    } catch { showError('Gagal', 'Gagal mengupdate status'); }
  };

  const handleGenerate = async () => {
    try {
      const res = await fetch('/api/admin/payroll/generate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ month: generateMonth }),
      });
      if (!res.ok) throw new Error();
      showSuccess('Berhasil', `Payroll bulan ${generateMonth} digenerate`);
      setIsGenerateOpen(false);
      setFilterMonth(generateMonth);
      load();
    } catch {
      showError('Gagal', 'Gagal generate payroll');
    }
  };

  const filtered = records.filter(r => r.employeeId.toLowerCase().includes(search.toLowerCase()));

  const totalNet = filtered.reduce((s, r) => s + r.netAmount, 0);
  const paidCount = filtered.filter(r => r.status === 'PAID').length;

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="p-2 bg-emerald-500/10 rounded-xl">
            <DollarSign className="w-5 h-5 text-emerald-500" />
          </div>
          <div>
            <h1 className="text-xl font-bold text-gray-900 dark:text-white">Payroll</h1>
            <p className="text-sm text-gray-500 dark:text-gray-400">Manajemen gaji karyawan bulanan</p>
          </div>
        </div>
        <button onClick={() => { setGenerateMonth(currentMonth()); setIsGenerateOpen(true); }}
          className="flex items-center gap-2 px-4 py-2 bg-brand-500 hover:bg-brand-600 text-white rounded-xl text-sm font-semibold transition-colors">
          <Play className="w-4 h-4" /> Generate Payroll
        </button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-3 gap-4">
        <div className="bg-white dark:bg-gray-800/50 rounded-xl border border-gray-200 dark:border-gray-700 p-4">
          <p className="text-xs text-gray-500 dark:text-gray-400 mb-1">Total Karyawan</p>
          <p className="text-2xl font-bold text-gray-900 dark:text-white">{filtered.length}</p>
        </div>
        <div className="bg-white dark:bg-gray-800/50 rounded-xl border border-gray-200 dark:border-gray-700 p-4">
          <p className="text-xs text-gray-500 dark:text-gray-400 mb-1">Sudah Dibayar</p>
          <p className="text-2xl font-bold text-green-600 dark:text-green-400">{paidCount}</p>
        </div>
        <div className="bg-white dark:bg-gray-800/50 rounded-xl border border-gray-200 dark:border-gray-700 p-4">
          <p className="text-xs text-gray-500 dark:text-gray-400 mb-1">Total Penggajian</p>
          <p className="text-lg font-bold text-brand-600 dark:text-brand-400">{fmt(totalNet)}</p>
        </div>
      </div>

      {/* Filters */}
      <div className="flex gap-3 flex-wrap">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <input value={search} onChange={e => setSearch(e.target.value)} placeholder="Cari ID karyawan..."
            className="pl-9 pr-3 py-2 text-sm border border-gray-200 dark:border-gray-700 rounded-xl bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-brand-500" />
        </div>
        <input type="month" value={filterMonth} onChange={e => setFilterMonth(e.target.value)}
          className="px-3 py-2 text-sm border border-gray-200 dark:border-gray-700 rounded-xl bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-brand-500" />
        <select value={filterStatus} onChange={e => setFilterStatus(e.target.value)}
          className="px-3 py-2 text-sm border border-gray-200 dark:border-gray-700 rounded-xl bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-brand-500">
          <option value="">Semua Status</option>
          <option value="DRAFT">Draft</option>
          <option value="PAID">Paid</option>
          <option value="CANCELLED">Cancelled</option>
        </select>
      </div>

      {/* Table */}
      {loading ? (
        <div className="text-center py-12 text-gray-400">Memuat data...</div>
      ) : filtered.length === 0 ? (
        <div className="text-center py-12 text-gray-400">
          <DollarSign className="w-10 h-10 mx-auto mb-2 opacity-30" />
          <p>Belum ada data payroll bulan ini</p>
          <button onClick={() => { setGenerateMonth(filterMonth); setIsGenerateOpen(true); }}
            className="mt-3 px-4 py-2 bg-brand-500 hover:bg-brand-600 text-white rounded-xl text-sm font-semibold transition-colors inline-flex items-center gap-2">
            <Play className="w-4 h-4" /> Generate Payroll
          </button>
        </div>
      ) : (
        <div className="bg-white dark:bg-gray-800/50 rounded-xl border border-gray-200 dark:border-gray-700 overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800">
                <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-400">Karyawan</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-400">Gaji Pokok</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-400">Bonus</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-400">Potongan</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-400">Nett</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-400">Status</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-400">Aksi</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
              {filtered.map(r => (
                <tr key={r.id} className="hover:bg-gray-50 dark:hover:bg-gray-700/30 transition-colors">
                  <td className="px-4 py-3 font-mono text-xs text-gray-700 dark:text-gray-300">{r.employeeId}</td>
                  <td className="px-4 py-3 text-gray-600 dark:text-gray-400">{fmt(r.baseWage)}</td>
                  <td className="px-4 py-3 text-green-600 dark:text-green-400">+{fmt(r.bonus + r.overtime + r.allowance)}</td>
                  <td className="px-4 py-3 text-red-600 dark:text-red-400">-{fmt(r.deduction)}</td>
                  <td className="px-4 py-3 font-bold text-gray-900 dark:text-white">{fmt(r.netAmount)}</td>
                  <td className="px-4 py-3">
                    <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${STATUS_COLORS[r.status] || ''}`}>{r.status}</span>
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-1">
                      {r.status === 'DRAFT' && (
                        <button onClick={() => handlePay(r.id)} title="Tandai Lunas" className="p-1.5 text-gray-400 hover:text-green-500 hover:bg-green-50 dark:hover:bg-green-500/10 rounded-lg transition-colors">
                          <CheckCircle2 className="w-3.5 h-3.5" />
                        </button>
                      )}
                      <button onClick={() => openEdit(r)} className="p-1.5 text-gray-400 hover:text-brand-500 hover:bg-brand-50 dark:hover:bg-brand-500/10 rounded-lg transition-colors">
                        <Pencil className="w-3.5 h-3.5" />
                      </button>
                      <button onClick={() => handleDelete(r.id)} className="p-1.5 text-gray-400 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-500/10 rounded-lg transition-colors">
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

      {/* Edit Modal */}
      <SimpleModal isOpen={isOpen} onClose={() => setIsOpen(false)} size="lg">
        <ModalHeader><ModalTitle>Edit Payroll</ModalTitle></ModalHeader>
        <ModalBody>
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-3">
              <div>
                <ModalLabel>Gaji Pokok</ModalLabel>
                <ModalInput type="number" value={form.baseWage} onChange={e => setForm(f => ({ ...f, baseWage: e.target.value }))} />
              </div>
              <div>
                <ModalLabel>Tunjangan</ModalLabel>
                <ModalInput type="number" value={form.allowance} onChange={e => setForm(f => ({ ...f, allowance: e.target.value }))} />
              </div>
              <div>
                <ModalLabel>Bonus</ModalLabel>
                <ModalInput type="number" value={form.bonus} onChange={e => setForm(f => ({ ...f, bonus: e.target.value }))} />
              </div>
              <div>
                <ModalLabel>Lembur</ModalLabel>
                <ModalInput type="number" value={form.overtime} onChange={e => setForm(f => ({ ...f, overtime: e.target.value }))} />
              </div>
              <div>
                <ModalLabel>Potongan</ModalLabel>
                <ModalInput type="number" value={form.deduction} onChange={e => setForm(f => ({ ...f, deduction: e.target.value }))} />
              </div>
              <div className="flex flex-col justify-end">
                <div className="bg-gray-50 dark:bg-gray-900/50 rounded-xl p-3 text-sm">
                  <span className="text-gray-500 dark:text-gray-400">Nett: </span>
                  <span className="font-bold text-brand-600 dark:text-brand-400">
                    {fmt((Number(form.baseWage)||0)+(Number(form.allowance)||0)+(Number(form.bonus)||0)+(Number(form.overtime)||0)-(Number(form.deduction)||0))}
                  </span>
                </div>
              </div>
            </div>
            <div>
              <ModalLabel>Catatan</ModalLabel>
              <ModalTextarea value={form.notes} onChange={e => setForm(f => ({ ...f, notes: e.target.value }))} rows={2} />
            </div>
          </div>
        </ModalBody>
        <ModalFooter>
          <ModalButton variant="secondary" onClick={() => setIsOpen(false)}>Batal</ModalButton>
          <ModalButton variant="primary" onClick={handleSave}>Simpan</ModalButton>
        </ModalFooter>
      </SimpleModal>

      {/* Generate Modal */}
      <SimpleModal isOpen={isGenerateOpen} onClose={() => setIsGenerateOpen(false)}>
        <ModalHeader><ModalTitle>Generate Payroll</ModalTitle></ModalHeader>
        <ModalBody>
          <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
            Generate slip gaji untuk semua karyawan aktif pada bulan yang dipilih.
          </p>
          <div>
            <ModalLabel>Bulan</ModalLabel>
            <ModalInput type="month" value={generateMonth} onChange={e => setGenerateMonth(e.target.value)} />
          </div>
        </ModalBody>
        <ModalFooter>
          <ModalButton variant="secondary" onClick={() => setIsGenerateOpen(false)}>Batal</ModalButton>
          <ModalButton variant="primary" onClick={handleGenerate}>Generate</ModalButton>
        </ModalFooter>
      </SimpleModal>
    </div>
  );
}
