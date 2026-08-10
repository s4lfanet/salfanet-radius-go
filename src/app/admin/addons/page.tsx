'use client';

import { useState, useEffect, useCallback } from 'react';
import { Plus, Pencil, Trash2, RefreshCw, Package, ToggleLeft, ToggleRight, X } from 'lucide-react';
import { showSuccess, showError, showConfirm } from '@/lib/sweetalert';
import { usePermissions } from '@/hooks/usePermissions';
import {
  SimpleModal,
  ModalHeader,
  ModalTitle,
  ModalBody,
  ModalFooter,
  ModalInput,
  ModalLabel,
  ModalButton,
} from '@/components/cyberpunk';

interface AddonType {
  id: number;
  name: string;
  description: string | null;
  price: number;
  isRecurring: boolean;
  isActive: boolean;
  createdAt: string;
}

export default function AddonTypesPage() {
  const { hasPermission, loading: permLoading } = usePermissions();
  const [addons, setAddons] = useState<AddonType[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [editing, setEditing] = useState<AddonType | null>(null);
  const [form, setForm] = useState({ name: '', description: '', price: '', isRecurring: true });
  const [saving, setSaving] = useState(false);

  const fetchAddons = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/addon-types');
      if (res.ok) setAddons(await res.json());
    } catch {
      showError('Gagal memuat data');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchAddons();
  }, [fetchAddons]);

  const openCreate = () => {
    setEditing(null);
    setForm({ name: '', description: '', price: '', isRecurring: true });
    setShowModal(true);
  };

  const openEdit = (addon: AddonType) => {
    setEditing(addon);
    setForm({
      name: addon.name,
      description: addon.description || '',
      price: String(addon.price),
      isRecurring: addon.isRecurring,
    });
    setShowModal(true);
  };

  const handleSave = async () => {
    if (!form.name.trim()) return showError('Nama addon wajib diisi');
    setSaving(true);
    try {
      const body = {
        name: form.name.trim(),
        description: form.description.trim() || null,
        price: parseInt(form.price) || 0,
        isRecurring: form.isRecurring,
      };
      const url = editing ? `/api/addon-types/${editing.id}` : '/api/addon-types';
      const method = editing ? 'PUT' : 'POST';
      const res = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || 'Gagal');
      showSuccess(editing ? 'Addon diperbarui' : 'Addon berhasil dibuat');
      setShowModal(false);
      fetchAddons();
    } catch (err: any) {
      showError(err.message);
    } finally {
      setSaving(false);
    }
  };

  const handleToggleActive = async (addon: AddonType) => {
    try {
      await fetch(`/api/addon-types/${addon.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ isActive: !addon.isActive }),
      });
      fetchAddons();
    } catch {
      showError('Gagal mengubah status');
    }
  };

  const handleDelete = async (addon: AddonType) => {
    const confirmed = await showConfirm(
      'Hapus Addon',
      `Hapus layanan tambahan "${addon.name}"? Jika masih digunakan pelanggan aktif, addon akan dinonaktifkan.`
    );
    if (!confirmed) return;
    try {
      const res = await fetch(`/api/addon-types/${addon.id}`, { method: 'DELETE' });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || 'Gagal');
      showSuccess(data.message);
      fetchAddons();
    } catch (err: any) {
      showError(err.message);
    }
  };

  if (permLoading) return null;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Layanan Tambahan</h1>
          <p className="text-sm text-muted-foreground mt-1">
            Kelola jenis layanan add-on (STB, IPTV, dll.) untuk pelanggan
          </p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={fetchAddons}
            className="flex items-center gap-1.5 px-3 py-2 text-sm rounded-lg border border-border hover:bg-accent transition-colors"
          >
            <RefreshCw className="w-4 h-4" />
          </button>
          {hasPermission('settings.view') && (
            <button
              onClick={openCreate}
              className="flex items-center gap-1.5 px-3 py-2 text-sm font-medium rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
            >
              <Plus className="w-4 h-4" /> Tambah Addon
            </button>
          )}
        </div>
      </div>

      <div className="rounded-lg border border-border bg-card">
        {loading ? (
          <div className="p-8 text-center text-muted-foreground">Memuat...</div>
        ) : addons.length === 0 ? (
          <div className="p-12 text-center text-muted-foreground">
            <Package className="w-10 h-10 mx-auto mb-4 opacity-40" />
            <p>Belum ada layanan tambahan.</p>
            {hasPermission('settings.view') && (
              <button
                onClick={openCreate}
                className="mt-4 flex items-center gap-1.5 px-3 py-2 text-sm font-medium rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors mx-auto"
              >
                <Plus className="w-4 h-4" /> Buat Addon Pertama
              </button>
            )}
          </div>
        ) : (
          <table className="w-full text-sm">
            <thead className="border-b border-border bg-muted/50">
              <tr>
                <th className="text-left p-3 font-medium">Nama Layanan</th>
                <th className="text-left p-3 font-medium">Keterangan</th>
                <th className="text-left p-3 font-medium">Harga / Bulan</th>
                <th className="text-left p-3 font-medium">Tipe</th>
                <th className="text-left p-3 font-medium">Status</th>
                <th className="text-right p-3 font-medium">Aksi</th>
              </tr>
            </thead>
            <tbody>
              {addons.map((a) => (
                <tr key={a.id} className={`border-b border-border last:border-0 ${a.isActive ? '' : 'opacity-50'}`}>
                  <td className="p-3 font-semibold">{a.name}</td>
                  <td className="p-3 text-muted-foreground text-xs">{a.description || '—'}</td>
                  <td className="p-3 font-bold">Rp {Number(a.price).toLocaleString('id-ID')}</td>
                  <td className="p-3">
                    <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                      a.isRecurring ? 'bg-purple-500/10 text-purple-600' : 'bg-green-500/10 text-green-600'
                    }`}>
                      {a.isRecurring ? 'Bulanan' : 'Sekali'}
                    </span>
                  </td>
                  <td className="p-3">
                    <button
                      onClick={() => handleToggleActive(a)}
                      className="flex items-center gap-1 text-xs hover:opacity-80 transition-opacity"
                    >
                      {a.isActive ? (
                        <ToggleRight className="w-5 h-5 text-green-500" />
                      ) : (
                        <ToggleLeft className="w-5 h-5 text-muted-foreground" />
                      )}
                      {a.isActive ? 'Aktif' : 'Nonaktif'}
                    </button>
                  </td>
                  <td className="p-3">
                    <div className="flex gap-1.5 justify-end">
                      <button
                        onClick={() => openEdit(a)}
                        className="p-1.5 rounded hover:bg-accent transition-colors"
                        title="Edit"
                      >
                        <Pencil className="w-3.5 h-3.5 text-blue-500" />
                      </button>
                      <button
                        onClick={() => handleDelete(a)}
                        className="p-1.5 rounded hover:bg-accent transition-colors"
                        title="Hapus"
                      >
                        <Trash2 className="w-3.5 h-3.5 text-red-500" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {showModal && (
        <SimpleModal isOpen={showModal} onClose={() => setShowModal(false)}>
          <ModalHeader>
            <div className="flex items-center gap-2.5">
              <div className="flex items-center justify-center w-9 h-9 rounded-lg bg-primary/10">
                <Package className="w-4 h-4 text-primary" />
              </div>
              <div>
                <ModalTitle>{editing ? 'Edit Layanan Tambahan' : 'Tambah Layanan Tambahan'}</ModalTitle>
              </div>
            </div>
            <button onClick={() => setShowModal(false)} className="p-1 rounded hover:bg-accent">
              <X className="w-5 h-5" />
            </button>
          </ModalHeader>
          <ModalBody>
            <div className="space-y-4">
              <div>
                <ModalLabel>Nama Layanan *</ModalLabel>
                <ModalInput
                  placeholder="Misal: Sewa STB, IPTV Premium"
                  value={form.name}
                  onChange={(e: any) => setForm((f) => ({ ...f, name: e.target.value }))}
                  autoFocus
                />
              </div>
              <div>
                <ModalLabel>Keterangan (opsional)</ModalLabel>
                <ModalInput
                  placeholder="Deskripsi singkat layanan ini"
                  value={form.description}
                  onChange={(e: any) => setForm((f) => ({ ...f, description: e.target.value }))}
                />
              </div>
              <div>
                <ModalLabel>Harga (Rp)</ModalLabel>
                <ModalInput
                  type="number"
                  min="0"
                  placeholder="0"
                  value={form.price}
                  onChange={(e: any) => setForm((f) => ({ ...f, price: e.target.value }))}
                />
              </div>
              <div>
                <ModalLabel>Tipe Biaya</ModalLabel>
                <div className="flex gap-3 mt-1">
                  {[
                    { val: true, label: 'Bulanan (recurring)' },
                    { val: false, label: 'Sekali bayar' },
                  ].map((opt) => (
                    <label
                      key={String(opt.val)}
                      className={`flex items-center gap-2 px-4 py-2 rounded-lg flex-1 justify-center text-sm cursor-pointer border-2 transition-all ${
                        form.isRecurring === opt.val
                          ? 'border-primary bg-primary/5 text-primary font-semibold'
                          : 'border-border text-muted-foreground'
                      }`}
                    >
                      <input
                        type="radio"
                        className="hidden"
                        checked={form.isRecurring === opt.val}
                        onChange={() => setForm((f) => ({ ...f, isRecurring: opt.val }))}
                      />
                      {opt.label}
                    </label>
                  ))}
                </div>
              </div>
            </div>
          </ModalBody>
          <ModalFooter>
            <ModalButton variant="secondary" onClick={() => setShowModal(false)}>
              Batal
            </ModalButton>
            <ModalButton variant="primary" onClick={handleSave} disabled={saving}>
              {saving ? 'Menyimpan...' : editing ? 'Simpan Perubahan' : 'Tambahkan'}
            </ModalButton>
          </ModalFooter>
        </SimpleModal>
      )}
    </div>
  );
}
