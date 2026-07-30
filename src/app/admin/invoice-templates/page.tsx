'use client';

import { useState, useEffect } from 'react';
import { showSuccess, showError, showConfirm } from '@/lib/sweetalert';
import { Plus, Pencil, Trash2, FileText, Star, Eye } from 'lucide-react';
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

interface InvoiceTemplate {
  id: string;
  name: string;
  subject: string;
  htmlBody: string;
  isDefault: boolean;
  templateType: string;
  createdAt: string;
}

export default function InvoiceTemplatesPage() {
  const [templates, setTemplates] = useState<InvoiceTemplate[]>([]);
  const [loading, setLoading] = useState(true);
  const [isOpen, setIsOpen] = useState(false);
  const [isPreviewOpen, setIsPreviewOpen] = useState(false);
  const [previewHtml, setPreviewHtml] = useState('');
  const [editing, setEditing] = useState<InvoiceTemplate | null>(null);
  const [form, setForm] = useState({ name: '', subject: '', htmlBody: '' });

  const load = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/invoice-templates');
      const data = await res.json();
      setTemplates(data.templates || []);
    } catch {
      showError('Gagal', 'Gagal memuat template invoice');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []);

  const openCreate = () => {
    setEditing(null);
    setForm({ name: '', subject: '', htmlBody: '' });
    setIsOpen(true);
  };

  const openEdit = (t: InvoiceTemplate) => {
    setEditing(t);
    setForm({ name: t.name, subject: t.subject, htmlBody: t.htmlBody });
    setIsOpen(true);
  };

  const openPreview = (t: InvoiceTemplate) => {
    setPreviewHtml(t.htmlBody);
    setIsPreviewOpen(true);
  };

  const handleSave = async () => {
    if (!form.name.trim()) return showError('Error', 'Nama template wajib diisi');
    try {
      const method = editing ? 'PUT' : 'POST';
      const url = editing ? `/api/invoice-templates/${editing.id}` : '/api/invoice-templates';
      const res = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(form),
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
      await fetch(`/api/invoice-templates/${id}`, { method: 'DELETE' });
      showSuccess('Dihapus', 'Template berhasil dihapus');
      load();
    } catch {
      showError('Gagal', 'Gagal menghapus template');
    }
  };

  const handleSetDefault = async (id: string) => {
    try {
      const res = await fetch(`/api/invoice-templates/${id}/default`, { method: 'POST' });
      if (!res.ok) throw new Error();
      showSuccess('Berhasil', 'Template default diperbarui');
      load();
    } catch {
      showError('Gagal', 'Gagal mengatur template default');
    }
  };

  return (
    <div className="space-y-6 p-6">
      {/* Background */}
      <div className="fixed inset-0 overflow-hidden pointer-events-none -z-10">
      </div>

      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="p-2.5 rounded-xl bg-brand-600/20 border border-brand-600/30">
            <FileText className="w-6 h-6 text-brand-600" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-foreground">Template Invoice</h1>
            <p className="text-sm text-muted-foreground">Kelola template HTML untuk invoice</p>
          </div>
        </div>
        <button
          onClick={openCreate}
          className="flex items-center gap-2 px-4 py-2 rounded-xl bg-brand-600/20 hover:bg-brand-600/30 border border-brand-600/40 text-brand-600 text-sm font-medium transition-all"
        >
          <Plus className="w-4 h-4" />
          Tambah Template
        </button>
      </div>

      {/* Table */}
      <div className="rounded-xl border border-border/50 bg-card/50 backdrop-blur-sm overflow-hidden">
        {loading ? (
          <div className="flex items-center justify-center py-20 text-muted-foreground">Memuat...</div>
        ) : templates.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-20 text-muted-foreground gap-2">
            <FileText className="w-10 h-10 opacity-30" />
            <p>Belum ada template invoice</p>
          </div>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border/50 text-muted-foreground text-xs uppercase">
                <th className="text-left px-4 py-3">Nama</th>
                <th className="text-left px-4 py-3">Subject</th>
                <th className="text-left px-4 py-3">Tipe</th>
                <th className="text-center px-4 py-3">Default</th>
                <th className="text-right px-4 py-3">Aksi</th>
              </tr>
            </thead>
            <tbody>
              {templates.map((t) => (
                <tr key={t.id} className="border-b border-border/30 hover:bg-muted/20 transition-colors">
                  <td className="px-4 py-3 font-medium text-foreground">{t.name}</td>
                  <td className="px-4 py-3 text-muted-foreground">{t.subject || '-'}</td>
                  <td className="px-4 py-3">
                    <span className="px-2 py-0.5 rounded-full text-xs bg-blue-500/15 text-blue-400 border border-blue-500/30">
                      {t.templateType}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-center">
                    {t.isDefault ? (
                      <span className="px-2 py-0.5 rounded-full text-xs bg-emerald-500/15 text-emerald-400 border border-emerald-500/30">
                        Default
                      </span>
                    ) : (
                      <button
                        onClick={() => handleSetDefault(t.id)}
                        className="text-xs text-muted-foreground hover:text-brand-600 transition-colors"
                        title="Jadikan default"
                      >
                        <Star className="w-4 h-4 mx-auto" />
                      </button>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center justify-end gap-2">
                      <button
                        onClick={() => openPreview(t)}
                        className="p-1.5 rounded-lg hover:bg-blue-500/20 text-muted-foreground hover:text-blue-400 transition-all"
                        title="Preview"
                      >
                        <Eye className="w-4 h-4" />
                      </button>
                      <button
                        onClick={() => openEdit(t)}
                        className="p-1.5 rounded-lg hover:bg-brand-600/20 text-muted-foreground hover:text-brand-600 transition-all"
                        title="Edit"
                      >
                        <Pencil className="w-4 h-4" />
                      </button>
                      <button
                        onClick={() => handleDelete(t.id)}
                        className="p-1.5 rounded-lg hover:bg-red-500/20 text-muted-foreground hover:text-red-400 transition-all"
                        title="Hapus"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* Form Modal */}
      <SimpleModal isOpen={isOpen} onClose={() => setIsOpen(false)}>
        <ModalHeader>
          <ModalTitle>{editing ? 'Edit Template' : 'Tambah Template Invoice'}</ModalTitle>
        </ModalHeader>
        <ModalBody className="space-y-4">
          <div>
            <ModalLabel>Nama Template</ModalLabel>
            <ModalInput
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              placeholder="contoh: Template Invoice Standar"
            />
          </div>
          <div>
            <ModalLabel>Subject Email</ModalLabel>
            <ModalInput
              value={form.subject}
              onChange={(e) => setForm({ ...form, subject: e.target.value })}
              placeholder="contoh: Invoice #{invoiceNumber}"
            />
          </div>
          <div>
            <ModalLabel>HTML Body</ModalLabel>
            <textarea
              value={form.htmlBody}
              onChange={(e) => setForm({ ...form, htmlBody: e.target.value })}
              rows={12}
              placeholder="<html>...</html>"
              className="w-full px-3 py-2 rounded-xl border border-border bg-background text-sm font-mono text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-[brand-600]/50 resize-y"
            />
          </div>
        </ModalBody>
        <ModalFooter>
          <ModalButton variant="secondary" onClick={() => setIsOpen(false)}>Batal</ModalButton>
          <ModalButton onClick={handleSave}>Simpan</ModalButton>
        </ModalFooter>
      </SimpleModal>

      {/* Preview Modal */}
      <SimpleModal isOpen={isPreviewOpen} onClose={() => setIsPreviewOpen(false)}>
        <ModalHeader>
          <ModalTitle>Preview Template</ModalTitle>
        </ModalHeader>
        <ModalBody>
          <div
            className="border border-border rounded-lg overflow-auto max-h-[60vh] bg-white text-black p-4"
            dangerouslySetInnerHTML={{ __html: previewHtml }}
          />
        </ModalBody>
        <ModalFooter>
          <ModalButton variant="secondary" onClick={() => setIsPreviewOpen(false)}>Tutup</ModalButton>
        </ModalFooter>
      </SimpleModal>
    </div>
  );
}
