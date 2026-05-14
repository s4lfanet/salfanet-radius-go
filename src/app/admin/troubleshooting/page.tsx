'use client';

import { useState, useEffect } from 'react';
import { showSuccess, showError, showConfirm } from '@/lib/sweetalert';
import { Plus, Pencil, Trash2, AlertTriangle, Search, CheckCircle2, XCircle } from 'lucide-react';
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

interface Checklist {
  id: string;
  title: string;
  description?: string;
  category: string;
  steps: string;
  isActive: boolean;
  createdAt: string;
}

const CATEGORIES = ['PPPOE', 'HOTSPOT', 'OLT', 'NETWORK', 'HARDWARE', 'SOFTWARE', 'OTHER'];

export default function TroubleshootingChecklistsPage() {
  const [checklists, setChecklists] = useState<Checklist[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [isOpen, setIsOpen] = useState(false);
  const [editing, setEditing] = useState<Checklist | null>(null);
  const [form, setForm] = useState({ title: '', description: '', category: 'NETWORK', steps: '' });

  const load = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/troubleshooting/checklists');
      const data = await res.json();
      setChecklists(data.checklists || []);
    } catch {
      showError('Gagal', 'Gagal memuat data checklist');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []);

  const openCreate = () => {
    setEditing(null);
    setForm({ title: '', description: '', category: 'NETWORK', steps: '' });
    setIsOpen(true);
  };

  const openEdit = (c: Checklist) => {
    setEditing(c);
    setForm({ title: c.title, description: c.description || '', category: c.category, steps: c.steps });
    setIsOpen(true);
  };

  const handleSave = async () => {
    if (!form.title.trim()) return showError('Error', 'Judul wajib diisi');
    try {
      const method = editing ? 'PUT' : 'POST';
      const url = editing ? `/api/troubleshooting/checklists/${editing.id}` : '/api/troubleshooting/checklists';
      const res = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(form),
      });
      if (!res.ok) throw new Error();
      showSuccess('Berhasil', editing ? 'Checklist diperbarui' : 'Checklist ditambahkan');
      setIsOpen(false);
      load();
    } catch {
      showError('Gagal', 'Gagal menyimpan checklist');
    }
  };

  const handleDelete = async (id: string) => {
    const ok = await showConfirm('Hapus Checklist?', 'Data tidak bisa dikembalikan');
    if (!ok) return;
    try {
      await fetch(`/api/troubleshooting/checklists/${id}`, { method: 'DELETE' });
      showSuccess('Dihapus', 'Checklist berhasil dihapus');
      load();
    } catch {
      showError('Gagal', 'Gagal menghapus checklist');
    }
  };

  const filtered = checklists.filter(c =>
    c.title.toLowerCase().includes(search.toLowerCase()) ||
    c.category.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="p-2 bg-yellow-500/10 rounded-xl">
            <AlertTriangle className="w-5 h-5 text-yellow-500" />
          </div>
          <div>
            <h1 className="text-xl font-bold text-gray-900 dark:text-white">Troubleshooting Checklists</h1>
            <p className="text-sm text-gray-500 dark:text-gray-400">Kelola panduan troubleshooting jaringan</p>
          </div>
        </div>
        <button
          onClick={openCreate}
          className="flex items-center gap-2 px-4 py-2 bg-brand-500 hover:bg-brand-600 text-white rounded-xl text-sm font-semibold transition-colors"
        >
          <Plus className="w-4 h-4" />
          Tambah Checklist
        </button>
      </div>

      {/* Search */}
      <div className="relative max-w-xs">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
        <input
          value={search}
          onChange={e => setSearch(e.target.value)}
          placeholder="Cari checklist..."
          className="w-full pl-9 pr-3 py-2 text-sm border border-gray-200 dark:border-gray-700 rounded-xl bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-brand-500"
        />
      </div>

      {/* Table */}
      {loading ? (
        <div className="text-center py-12 text-gray-400">Memuat data...</div>
      ) : filtered.length === 0 ? (
        <div className="text-center py-12 text-gray-400">
          <AlertTriangle className="w-10 h-10 mx-auto mb-2 opacity-30" />
          <p>Belum ada checklist</p>
        </div>
      ) : (
        <div className="bg-white dark:bg-gray-800/50 rounded-2xl border border-gray-200 dark:border-gray-700 overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800">
                <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-400">Judul</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-400">Kategori</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-400">Status</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-400">Aksi</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
              {filtered.map(c => (
                <tr key={c.id} className="hover:bg-gray-50 dark:hover:bg-gray-700/30 transition-colors">
                  <td className="px-4 py-3">
                    <p className="font-medium text-gray-900 dark:text-white">{c.title}</p>
                    {c.description && <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5 line-clamp-1">{c.description}</p>}
                  </td>
                  <td className="px-4 py-3">
                    <span className="px-2 py-0.5 bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-400 rounded-full text-xs font-medium">
                      {c.category}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    {c.isActive ? (
                      <span className="flex items-center gap-1 text-green-600 dark:text-green-400 text-xs font-medium">
                        <CheckCircle2 className="w-3.5 h-3.5" /> Aktif
                      </span>
                    ) : (
                      <span className="flex items-center gap-1 text-red-500 text-xs font-medium">
                        <XCircle className="w-3.5 h-3.5" /> Nonaktif
                      </span>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
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
      <SimpleModal isOpen={isOpen} onClose={() => setIsOpen(false)} size="lg">
        <ModalHeader>
          <ModalTitle>{editing ? 'Edit Checklist' : 'Tambah Checklist'}</ModalTitle>
        </ModalHeader>
        <ModalBody>
          <div className="space-y-4">
            <div>
              <ModalLabel>Judul *</ModalLabel>
              <ModalInput value={form.title} onChange={e => setForm(f => ({ ...f, title: e.target.value }))} placeholder="Judul checklist" />
            </div>
            <div>
              <ModalLabel>Kategori</ModalLabel>
              <ModalSelect value={form.category} onChange={e => setForm(f => ({ ...f, category: e.target.value }))}>
                {CATEGORIES.map(cat => <option key={cat} value={cat}>{cat}</option>)}
              </ModalSelect>
            </div>
            <div>
              <ModalLabel>Deskripsi</ModalLabel>
              <ModalTextarea value={form.description} onChange={e => setForm(f => ({ ...f, description: e.target.value }))} placeholder="Deskripsi singkat" rows={2} />
            </div>
            <div>
              <ModalLabel>Langkah-langkah (steps)</ModalLabel>
              <ModalTextarea value={form.steps} onChange={e => setForm(f => ({ ...f, steps: e.target.value }))} placeholder="1. Cek kabel&#10;2. Ping gateway&#10;3. ..." rows={5} />
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
