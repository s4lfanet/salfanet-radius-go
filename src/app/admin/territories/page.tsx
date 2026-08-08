'use client';

import { useState, useEffect } from 'react';
import { Plus, Pencil, Trash2, MapPin, Users, Search, X, Loader2, User } from 'lucide-react';
import { showSuccess, showError, showConfirm } from '@/lib/sweetalert';
import { usePermissions } from '@/hooks/usePermissions';
import { useTranslation } from '@/hooks/useTranslation';
import {
  SimpleModal,
  ModalHeader,
  ModalTitle,
  ModalDescription,
  ModalBody,
  ModalFooter,
  ModalInput,
  ModalTextarea,
  ModalLabel,
  ModalButton,
} from '@/components/cyberpunk';

interface Collector {
  id: string;
  name: string;
  email: string;
}

interface TerritoryArea {
  id: string;
  territoryId: string;
  kelurahanKode: string | null;
  kelurahanNama: string | null;
  kecamatanNama: string | null;
  kabupatenNama: string | null;
  provinsiNama: string | null;
  dusunNama: string | null;
  collectorId: string | null;
}

interface Territory {
  id: string;
  name: string;
  description: string | null;
  collectorId: string | null;
  isActive: boolean;
  userCount: number;
  collector?: Collector | null;
  areas?: TerritoryArea[];
}

export default function TerritoriesPage() {
  const { hasPermission, loading: permLoading } = usePermissions();
  const { t } = useTranslation();
  const [loading, setLoading] = useState(true);
  const [territories, setTerritories] = useState<Territory[]>([]);
  const [collectors, setCollectors] = useState<Collector[]>([]);
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [editingTerritory, setEditingTerritory] = useState<Territory | null>(null);
  const [selectedTerritory, setSelectedTerritory] = useState<Territory | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    collectorId: '',
    isActive: true,
  });
  const [areaForm, setAreaForm] = useState({
    kelurahanNama: '',
    kecamatanNama: '',
    kabupatenNama: '',
    provinsiNama: '',
    dusunNama: '',
  });

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    try {
      const [terrRes, collRes] = await Promise.all([
        fetch('/api/territories'),
        fetch('/api/territories/collectors'),
      ]);
      const terrData = await terrRes.json();
      const collData = await collRes.json();
      setTerritories(terrData.data || []);
      setCollectors(collData.data || []);
    } catch (error) {
      console.error('Load territories error:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleOpenDialog = (territory: Territory | null) => {
    if (territory) {
      setEditingTerritory(territory);
      setFormData({
        name: territory.name,
        description: territory.description || '',
        collectorId: territory.collectorId || '',
        isActive: territory.isActive,
      });
    } else {
      setEditingTerritory(null);
      setFormData({ name: '', description: '', collectorId: '', isActive: true });
    }
    setIsDialogOpen(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const payload = {
        name: formData.name,
        description: formData.description || null,
        collectorId: formData.collectorId || null,
        isActive: formData.isActive,
      };

      if (editingTerritory) {
        const res = await fetch(`/api/territories/${editingTerritory.id}`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload),
        });
        if (!res.ok) throw new Error('Failed to update territory');
        showSuccess('Wilayah berhasil diperbarui');
      } else {
        const res = await fetch('/api/territories', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload),
        });
        if (!res.ok) throw new Error('Failed to create territory');
        showSuccess('Wilayah berhasil dibuat');
      }
      setIsDialogOpen(false);
      loadData();
    } catch (error) {
      showError('Gagal menyimpan wilayah');
    }
  };

  const handleDelete = async (id: string) => {
    const confirmed = await showConfirm('Nonaktifkan wilayah ini?');
    if (!confirmed) return;
    try {
      const res = await fetch(`/api/territories/${id}`, { method: 'DELETE' });
      if (!res.ok) throw new Error('Failed to delete');
      showSuccess('Wilayah dinonaktifkan');
      loadData();
    } catch (error) {
      showError('Gagal menonaktifkan wilayah');
    }
  };

  const handleAddArea = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedTerritory) return;
    try {
      const res = await fetch(`/api/territories/${selectedTerritory.id}/areas`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          kelurahanNama: areaForm.kelurahanNama || null,
          kecamatanNama: areaForm.kecamatanNama || null,
          kabupatenNama: areaForm.kabupatenNama || null,
          provinsiNama: areaForm.provinsiNama || null,
          dusunNama: areaForm.dusunNama || null,
        }),
      });
      if (!res.ok) throw new Error('Failed to add area');
      showSuccess('Area berhasil ditambahkan');
      setAreaForm({ kelurahanNama: '', kecamatanNama: '', kabupatenNama: '', provinsiNama: '', dusunNama: '' });
      loadTerritoryDetail(selectedTerritory.id);
    } catch (error) {
      showError('Gagal menambahkan area');
    }
  };

  const handleRemoveArea = async (areaId: string) => {
    if (!selectedTerritory) return;
    const confirmed = await showConfirm('Hapus area ini?');
    if (!confirmed) return;
    try {
      const res = await fetch(`/api/territories/${selectedTerritory.id}/areas/${areaId}`, { method: 'DELETE' });
      if (!res.ok) throw new Error('Failed to remove area');
      showSuccess('Area dihapus');
      loadTerritoryDetail(selectedTerritory.id);
    } catch (error) {
      showError('Gagal menghapus area');
    }
  };

  const loadTerritoryDetail = async (id: string) => {
    try {
      const res = await fetch(`/api/territories/${id}`);
      const data = await res.json();
      setSelectedTerritory(data);
    } catch (error) {
      console.error('Load territory detail error:', error);
    }
  };

  const filteredTerritories = territories.filter((t) =>
    t.name.toLowerCase().includes(searchQuery.toLowerCase())
  );

  if (loading || permLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <Loader2 className="h-8 w-8 animate-spin text-cyan-400" />
      </div>
    );
  }

  return (
    <div className="space-y-6 p-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Manajemen Wilayah</h1>
          <p className="text-sm text-gray-400">Kelola wilayah dan kolektor ISP</p>
        </div>
        <button
          onClick={() => handleOpenDialog(null)}
          className="flex items-center gap-2 rounded-lg bg-cyan-500 px-4 py-2 text-sm font-medium text-black hover:bg-cyan-400"
        >
          <Plus className="h-4 w-4" />
          Tambah Wilayah
        </button>
      </div>

      <div className="flex items-center gap-2">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-500" />
          <input
            type="text"
            placeholder="Cari wilayah..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full rounded-lg border border-gray-700 bg-gray-900 py-2 pl-10 pr-4 text-sm text-white placeholder-gray-500 focus:border-cyan-500 focus:outline-none"
          />
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {filteredTerritories.map((territory) => (
          <div
            key={territory.id}
            className="rounded-lg border border-gray-800 bg-gray-900/50 p-4 hover:border-cyan-800 transition-colors"
          >
            <div className="flex items-start justify-between">
              <div className="flex-1">
                <div className="flex items-center gap-2">
                  <MapPin className="h-4 w-4 text-cyan-400" />
                  <h3 className="font-semibold text-white">{territory.name}</h3>
                </div>
                {territory.description && (
                  <p className="mt-1 text-sm text-gray-400">{territory.description}</p>
                )}
              </div>
              <div className="flex gap-1">
                <button
                  onClick={() => handleOpenDialog(territory)}
                  className="rounded p-1 text-gray-400 hover:text-cyan-400"
                >
                  <Pencil className="h-4 w-4" />
                </button>
                <button
                  onClick={() => handleDelete(territory.id)}
                  className="rounded p-1 text-gray-400 hover:text-red-400"
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>
            </div>

            <div className="mt-3 flex items-center gap-4 text-sm">
              {territory.collector && (
                <div className="flex items-center gap-1 text-gray-400">
                  <User className="h-3 w-3" />
                  {territory.collector.name}
                </div>
              )}
              <div className="flex items-center gap-1 text-gray-400">
                <Users className="h-3 w-3" />
                {territory.userCount} pelanggan
              </div>
            </div>

            {territory.areas && territory.areas.length > 0 && (
              <div className="mt-3 border-t border-gray-800 pt-2">
                <p className="text-xs text-gray-500 mb-1">Area ({territory.areas.length})</p>
                <div className="flex flex-wrap gap-1">
                  {territory.areas.slice(0, 3).map((area) => (
                    <span
                      key={area.id}
                      className="rounded bg-gray-800 px-2 py-0.5 text-xs text-gray-300"
                    >
                      {area.dusunNama || area.kelurahanNama || 'Area'}
                    </span>
                  ))}
                  {territory.areas.length > 3 && (
                    <button
                      onClick={() => loadTerritoryDetail(territory.id)}
                      className="text-xs text-cyan-400 hover:underline"
                    >
                      +{territory.areas.length - 3} lainnya
                    </button>
                  )}
                </div>
              </div>
            )}

            <button
              onClick={() => loadTerritoryDetail(territory.id)}
              className="mt-3 w-full rounded-lg border border-gray-800 py-1.5 text-xs text-gray-400 hover:border-cyan-800 hover:text-cyan-400"
            >
              Kelola Area
            </button>
          </div>
        ))}
      </div>

      {filteredTerritories.length === 0 && !loading && (
        <div className="text-center py-12 text-gray-500">
          <MapPin className="mx-auto h-12 w-12 mb-2 opacity-50" />
          <p>Belum ada wilayah. Klik "Tambah Wilayah" untuk membuat.</p>
        </div>
      )}

      {/* Create/Edit Territory Modal */}
      <SimpleModal isOpen={isDialogOpen} onClose={() => setIsDialogOpen(false)}>
        <form onSubmit={handleSubmit}>
          <ModalHeader>
            <ModalTitle>{editingTerritory ? 'Edit Wilayah' : 'Tambah Wilayah'}</ModalTitle>
            <ModalDescription>
              {editingTerritory ? 'Perbarui informasi wilayah' : 'Buat wilayah baru untuk manajemen kolektor'}
            </ModalDescription>
          </ModalHeader>
          <ModalBody>
            <div className="space-y-4">
              <div>
                <ModalLabel>Nama Wilayah</ModalLabel>
                <ModalInput
                  value={formData.name}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                    setFormData({ ...formData, name: e.target.value })
                  }
                  placeholder="contoh: Wilayah Utara"
                  required
                />
              </div>
              <div>
                <ModalLabel>Deskripsi</ModalLabel>
                <ModalTextarea
                  value={formData.description}
                  onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) =>
                    setFormData({ ...formData, description: e.target.value })
                  }
                  placeholder="Deskripsi wilayah (opsional)"
                  rows={2}
                />
              </div>
              <div>
                <ModalLabel>Kolektor</ModalLabel>
                <select
                  value={formData.collectorId}
                  onChange={(e) => setFormData({ ...formData, collectorId: e.target.value })}
                  className="w-full rounded-lg border border-gray-700 bg-gray-900 px-3 py-2 text-sm text-white focus:border-cyan-500 focus:outline-none"
                >
                  <option value="">-- Pilih Kolektor --</option>
                  {collectors.map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.name} ({c.email})
                    </option>
                  ))}
                </select>
              </div>
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={formData.isActive}
                  onChange={(e) => setFormData({ ...formData, isActive: e.target.checked })}
                  className="rounded border-gray-700"
                />
                <span className="text-sm text-gray-300">Aktif</span>
              </label>
            </div>
          </ModalBody>
          <ModalFooter>
            <ModalButton type="button" variant="secondary" onClick={() => setIsDialogOpen(false)}>
              Batal
            </ModalButton>
            <ModalButton type="submit">{editingTerritory ? 'Simpan' : 'Buat'}</ModalButton>
          </ModalFooter>
        </form>
      </SimpleModal>

      {/* Territory Areas Detail Modal */}
      <SimpleModal
        isOpen={!!selectedTerritory}
        onClose={() => setSelectedTerritory(null)}
      >
        <ModalHeader>
          <ModalTitle>Area Wilayah: {selectedTerritory?.name}</ModalTitle>
          <ModalDescription>Tambah atau hapus area (kelurahan/dusun) dalam wilayah ini</ModalDescription>
        </ModalHeader>
        <ModalBody>
          <form onSubmit={handleAddArea} className="space-y-3 mb-4">
            <div className="grid grid-cols-2 gap-2">
              <div>
                <ModalLabel>Provinsi</ModalLabel>
                <ModalInput
                  value={areaForm.provinsiNama}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                    setAreaForm({ ...areaForm, provinsiNama: e.target.value })
                  }
                  placeholder="Provinsi"
                />
              </div>
              <div>
                <ModalLabel>Kabupaten</ModalLabel>
                <ModalInput
                  value={areaForm.kabupatenNama}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                    setAreaForm({ ...areaForm, kabupatenNama: e.target.value })
                  }
                  placeholder="Kabupaten"
                />
              </div>
              <div>
                <ModalLabel>Kecamatan</ModalLabel>
                <ModalInput
                  value={areaForm.kecamatanNama}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                    setAreaForm({ ...areaForm, kecamatanNama: e.target.value })
                  }
                  placeholder="Kecamatan"
                />
              </div>
              <div>
                <ModalLabel>Kelurahan</ModalLabel>
                <ModalInput
                  value={areaForm.kelurahanNama}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                    setAreaForm({ ...areaForm, kelurahanNama: e.target.value })
                  }
                  placeholder="Kelurahan/Desa"
                />
              </div>
              <div className="col-span-2">
                <ModalLabel>Dusun (opsional)</ModalLabel>
                <ModalInput
                  value={areaForm.dusunNama}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                    setAreaForm({ ...areaForm, dusunNama: e.target.value })
                  }
                  placeholder="Nama dusun/RT/RW"
                />
              </div>
            </div>
            <button
              type="submit"
              className="flex items-center gap-2 rounded-lg bg-cyan-500 px-3 py-1.5 text-sm font-medium text-black hover:bg-cyan-400"
            >
              <Plus className="h-4 w-4" />
              Tambah Area
            </button>
          </form>

          <div className="space-y-2 max-h-60 overflow-y-auto">
            {selectedTerritory?.areas?.map((area) => (
              <div
                key={area.id}
                className="flex items-center justify-between rounded-lg border border-gray-800 bg-gray-900/50 px-3 py-2"
              >
                <div className="text-sm text-gray-300">
                  {area.dusunNama && <span className="font-medium">{area.dusunNama}, </span>}
                  {area.kelurahanNama}
                  {area.kecamatanNama && <span className="text-gray-500"> - {area.kecamatanNama}</span>}
                </div>
                <button
                  onClick={() => handleRemoveArea(area.id)}
                  className="rounded p-1 text-gray-400 hover:text-red-400"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              </div>
            ))}
            {(!selectedTerritory?.areas || selectedTerritory.areas.length === 0) && (
              <p className="text-center text-sm text-gray-500 py-4">Belum ada area</p>
            )}
          </div>
        </ModalBody>
      </SimpleModal>
    </div>
  );
}
