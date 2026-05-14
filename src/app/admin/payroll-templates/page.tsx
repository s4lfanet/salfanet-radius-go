'use client';

import { useState, useEffect } from 'react';
import { showSuccess, showError, showConfirm } from '@/lib/sweetalert';
import { FileText, Plus, Pencil, Trash2, Search, Star, Copy } from 'lucide-react';
import {
  SimpleModal,
  ModalHeader,
  ModalTitle,
  ModalBody,
  ModalFooter,
  ModalInput,
  ModalTextarea,
  ModalLabel,
  ModalButton,
} from '@/components/cyberpunk';

interface PayrollTemplate {
  id: string;
  name: string;
  baseWage: number;
  allowance: number;
  deduction: number;
  notes?: string;
  isDefault: boolean;
  createdAt: string;
  updatedAt: string;
}

const fmt = (n: number) => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(n);

export default function PayrollTemplatesPage() {
  const [templates, setTemplates] = useState<PayrollTemplate[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [isOpen, setIsOpen] = useState(false);
  const [editing, setEditing] = useState<PayrollTemplate | null>(null);
  const [form, setForm] = useState({ name: '', baseWage: '', allowance: '', deduction: '', notes: '' });

  const load = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/payroll-templates');
      const data = await res.json();
      setTemplates(data.templates || []);
    } catch {
      showError('Gagal', 'Gagal memuat template payroll');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []);

  const openCreate = () => {
    setEditing(null);
    setForm({ name: '', baseWage: '', allowance: '0', deduction: '0', notes: '' });
    setIsOpen(true);
  };

  const openEdit = (t: PayrollTemplate) => {
    setEditing(t);
    setForm({ name: t.name, baseWage: String(t.baseWage), allowance: String(t.allowance), deduction: String(t.deduction), notes: t.notes || '' });
    setIsOpen(true);
  };

  const handleSave = async () => {
    if (!form.name.trim()) return showError('Error', 'Nama template wajib diisi');
    try {
      const url = editing ? `/api/payroll-templates/${editing.id}` : '/api/payroll-templates';
      const method = editing ? 'PUT' : 'POST';
      const res = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: form.name,
          baseWage: Number(form.baseWage) || 0,
          allowance: Number(form.allowance) || 0,
          deduction: Number(form.deduction) || 0,
          notes: form.notes,
        }),
      });
      if (!res.ok) throw new Error();
      showSuccess('Berhasil', editing ? 'Template diperbarui' : 'Template ditambahkan');
      setIsOpen(false);
      load();
    } catch {
      showError('Gagal', 'Gagal menyimpan template');
    }
  };

  const handleDelete = async (id: string) => {
    const ok = await showConfirm('Hapus Template?', 'Data tidak bisa dikembalikan');
    if (!ok) return;
    try {
      await fetch(`/api/payroll-templates/${id}`, { method: 'DELETE' });
      showSuccess('Dihapus', 'Template dihapus');
      load();
    } catch { showError('Gagal', 'Gagal menghapus'); }
  };

  const handleSetDefault = async (id: string) => {
    try {
      await fetch(`/api/payroll-templates/${id}/default`, { method: 'POST' });
      showSuccess('Diatur', 'Template dijadikan default');
      load();
    } catch { showError('Gagal', 'Gagal mengatur default'); }
  };

  const filtered = templates.filter(t => t.name.toLowerCase().includes(search.toLowerCase()));

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="p-2 bg-indigo-500/10 rounded-xl">
            <FileText className="w-5 h-5 text-indigo-500" />
          </div>
          <div>
            <h1 className="text-xl font-bold text-gray-900 dark:text-white">Template Payroll</h1>
            <p className="text-sm text-gray-500 dark:text-gray-400">Kelola template perhitungan gaji karyawan</p>
          </div>
        </div>
        <button onClick={openCreate} className="flex items-center gap-2 px-4 py-2 bg-brand-500 hover:bg-brand-600 text-white rounded-xl text-sm font-semibold transition-colors">
          <Plus className="w-4 h-4" /> Tambah Template
        </button>
      </div>

      <div className="relative max-w-xs">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
        <input value={search} onChange={e => setSearch(e.target.value)} placeholder="Cari template..."
          className="w-full pl-9 pr-3 py-2 text-sm border border-gray-200 dark:border-gray-700 rounded-xl bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-brand-500" />
      </div>

      {loading ? (
        <div className="text-center py-12 text-gray-400">Memuat data...</div>
      ) : filtered.length === 0 ? (
        <div className="text-center py-12 text-gray-400">
          <FileText className="w-10 h-10 mx-auto mb-2 opacity-30" />
          <p>Belum ada template payroll</p>
        </div>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {filtered.map(t => {
            const net = t.baseWage + t.allowance - t.deduction;
            return (
              <div key={t.id} className={`bg-white dark:bg-gray-800/50 rounded-2xl border p-5 ${t.isDefault ? 'border-brand-400 dark:border-brand-500' : 'border-gray-200 dark:border-gray-700'}`}>
                <div className="flex items-start justify-between mb-3">
                  <div>
                    <div className="flex items-center gap-2">
                      <h3 className="font-semibold text-gray-900 dark:text-white">{t.name}</h3>
                      {t.isDefault && <Star className="w-3.5 h-3.5 text-yellow-500 fill-yellow-500" />}
                    </div>
                    {t.notes && <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5 line-clamp-1">{t.notes}</p>}
                  </div>
                  <div className="flex items-center gap-1">
                    {!t.isDefault && (
                      <button onClick={() => handleSetDefault(t.id)} title="Jadikan Default" className="p-1.5 text-gray-400 hover:text-yellow-500 rounded-lg transition-colors">
                        <Star className="w-3.5 h-3.5" />
                      </button>
                    )}
                    <button onClick={() => openEdit(t)} className="p-1.5 text-gray-400 hover:text-brand-500 rounded-lg transition-colors">
                      <Pencil className="w-3.5 h-3.5" />
                    </button>
                    <button onClick={() => handleDelete(t.id)} className="p-1.5 text-gray-400 hover:text-red-500 rounded-lg transition-colors">
                      <Trash2 className="w-3.5 h-3.5" />
                    </button>
                  </div>
                </div>
                <div className="space-y-1.5 text-sm">
                  <div className="flex justify-between">
                    <span className="text-gray-500 dark:text-gray-400">Gaji Pokok</span>
                    <span className="font-medium text-gray-900 dark:text-white">{fmt(t.baseWage)}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-500 dark:text-gray-400">Tunjangan</span>
                    <span className="font-medium text-green-600 dark:text-green-400">+{fmt(t.allowance)}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-500 dark:text-gray-400">Potongan</span>
                    <span className="font-medium text-red-600 dark:text-red-400">-{fmt(t.deduction)}</span>
                  </div>
                  <div className="pt-1.5 border-t border-gray-100 dark:border-gray-700 flex justify-between">
                    <span className="font-semibold text-gray-700 dark:text-gray-300">Nett</span>
                    <span className="font-bold text-brand-600 dark:text-brand-400">{fmt(net)}</span>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Modal */}
      <SimpleModal isOpen={isOpen} onClose={() => setIsOpen(false)}>
        <ModalHeader><ModalTitle>{editing ? 'Edit Template' : 'Tambah Template Payroll'}</ModalTitle></ModalHeader>
        <ModalBody>
          <div className="space-y-4">
            <div>
              <ModalLabel>Nama Template *</ModalLabel>
              <ModalInput value={form.name} onChange={e => setForm(f => ({ ...f, name: e.target.value }))} placeholder="Contoh: Staff Level 1" />
            </div>
            <div className="grid grid-cols-3 gap-3">
              <div>
                <ModalLabel>Gaji Pokok</ModalLabel>
                <ModalInput type="number" value={form.baseWage} onChange={e => setForm(f => ({ ...f, baseWage: e.target.value }))} placeholder="2000000" />
              </div>
              <div>
                <ModalLabel>Tunjangan</ModalLabel>
                <ModalInput type="number" value={form.allowance} onChange={e => setForm(f => ({ ...f, allowance: e.target.value }))} placeholder="500000" />
              </div>
              <div>
                <ModalLabel>Potongan</ModalLabel>
                <ModalInput type="number" value={form.deduction} onChange={e => setForm(f => ({ ...f, deduction: e.target.value }))} placeholder="100000" />
              </div>
            </div>
            <div>
              <ModalLabel>Catatan</ModalLabel>
              <ModalTextarea value={form.notes} onChange={e => setForm(f => ({ ...f, notes: e.target.value }))} rows={2} placeholder="Keterangan template..." />
            </div>
            {/* Preview */}
            <div className="bg-gray-50 dark:bg-gray-900/50 rounded-xl p-3 text-sm">
              <p className="font-semibold text-gray-700 dark:text-gray-300 mb-2">Preview</p>
              <div className="space-y-1">
                <div className="flex justify-between text-gray-600 dark:text-gray-400">
                  <span>Gaji Pokok</span><span>{fmt(Number(form.baseWage) || 0)}</span>
                </div>
                <div className="flex justify-between text-green-600 dark:text-green-400">
                  <span>Tunjangan</span><span>+{fmt(Number(form.allowance) || 0)}</span>
                </div>
                <div className="flex justify-between text-red-600 dark:text-red-400">
                  <span>Potongan</span><span>-{fmt(Number(form.deduction) || 0)}</span>
                </div>
                <div className="flex justify-between font-bold text-brand-600 dark:text-brand-400 border-t border-gray-200 dark:border-gray-700 pt-1">
                  <span>Nett</span><span>{fmt((Number(form.baseWage) || 0) + (Number(form.allowance) || 0) - (Number(form.deduction) || 0))}</span>
                </div>
              </div>
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
