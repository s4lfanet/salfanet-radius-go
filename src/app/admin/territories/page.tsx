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

interface PppoeArea {
  id: string;
  name: string;
  description: string | null;
  isActive: boolean;
  territoryId: string | null;
}

interface Territory {
  id: string;
  name: string;
  description: string | null;
  collectorId: string | null;
  isActive: boolean;
  userCount: number;
  collector?: Collector | null;
  areas?: PppoeArea[];
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
  const [availableAreas, setAvailableAreas] = useState<PppoeArea[]>([]);
  const [allAreas, setAllAreas] = useState<PppoeArea[]>([]);
  const [selectedAreaIds, setSelectedAreaIds] = useState<string[]>([]);
  const [selectedAreaId, setSelectedAreaId] = useState('');

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    try {
      const [terrRes, collRes, allAreasRes] = await Promise.all([
        fetch('/api/territories'),
        fetch('/api/territories/collectors'),
        fetch('/api/territories/all-areas'),
      ]);
      const terrData = await terrRes.json();
      const collData = await collRes.json();
      const allAreasData = await allAreasRes.json();
      setTerritories(terrData.data || []);
      setCollectors(collData.data || []);
      setAllAreas(allAreasData.data || []);
      // Available areas = areas without territoryId
      setAvailableAreas((allAreasData.data || []).filter((a: PppoeArea) => !a.territoryId));
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
      setSelectedAreaIds(territory.areas?.map((a) => a.id) || []);
    } else {
      setEditingTerritory(null);
      setFormData({ name: '', description: '', collectorId: '', isActive: true });
      setSelectedAreaIds([]);
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
        areaIds: selectedAreaIds,
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

  const handleAddArea = async () => {
    if (!selectedTerritory || !selectedAreaId) return;
    try {
      const res = await fetch(`/api/territories/${selectedTerritory.id}/areas`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ areaId: selectedAreaId }),
      });
      if (!res.ok) {
        const err = await res.json();
        throw new Error(err.error || 'Failed to assign area');
      }
      showSuccess('Area berhasil ditambahkan ke wilayah');
      setSelectedAreaId('');
      loadTerritoryDetail(selectedTerritory.id);
      // Refresh all areas
      const areasRes = await fetch('/api/territories/all-areas');
      const areasData = await areasRes.json();
      setAllAreas(areasData.data || []);
      setAvailableAreas((areasData.data || []).filter((a: PppoeArea) => !a.territoryId));
    } catch (error) {
      showError(error instanceof Error ? error.message : 'Gagal menambahkan area');
    }
  };

  const handleRemoveArea = async (areaId: string) => {
    if (!selectedTerritory) return;
    const confirmed = await showConfirm('Lepaskan area ini dari wilayah?');
    if (!confirmed) return;
    try {
      const res = await fetch(`/api/territories/${selectedTerritory.id}/areas/${areaId}`, { method: 'DELETE' });
      if (!res.ok) throw new Error('Failed to remove area');
      showSuccess('Area dilepaskan dari wilayah');
      loadTerritoryDetail(selectedTerritory.id);
      // Refresh all areas
      const areasRes = await fetch('/api/territories/all-areas');
      const areasData = await areasRes.json();
      setAllAreas(areasData.data || []);
      setAvailableAreas((areasData.data || []).filter((a: PppoeArea) => !a.territoryId));
    } catch (error) {
      showError('Gagal melepaskan area');
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
          <h1 className="text-2xl font-bold text-white">Manajemen Kolektor</h1>
          <p className="text-sm text-gray-400">Kelola kolektor dan area penagihan</p>
        </div>
        <button
          onClick={() => handleOpenDialog(null)}
          className="flex items-center gap-2 rounded-lg bg-cyan-500 px-4 py-2 text-sm font-medium text-black hover:bg-cyan-400"
        >
          <Plus className="h-4 w-4" />
          Tambah Kolektor
        </button>
      </div>

      <div className="flex items-center gap-2">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-500" />
          <input
            type="text"
            placeholder="Cari kolektor..."
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
                      {area.name}
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
          <p>Belum ada kolektor. Klik "Tambah Kolektor" untuk membuat.</p>
        </div>
      )}

      {/* Create/Edit Territory Modal */}
      <SimpleModal isOpen={isDialogOpen} onClose={() => setIsDialogOpen(false)}>
        <form onSubmit={handleSubmit}>
          <ModalHeader>
            <ModalTitle>{editingTerritory ? 'Edit Kolektor' : 'Tambah Kolektor'}</ModalTitle>
            <ModalDescription>
              {editingTerritory ? 'Perbarui informasi kolektor' : 'Buat kolektor baru dan tentukan area penagihan'}
            </ModalDescription>
          </ModalHeader>
          <ModalBody>
            <div className="space-y-4">
              <div>
                <ModalLabel>Nama Kolektor</ModalLabel>
                <ModalInput
                  value={formData.name}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                    setFormData({ ...formData, name: e.target.value })
                  }
                  placeholder="contoh: Kolektor Utara"
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
                  placeholder="Deskripsi kolektor (opsional)"
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
              {/* Area multi-select from pppoe_areas (Kelola Area) */}
              <div>
                <ModalLabel>Area (dari Kelola Area)</ModalLabel>
                <p className="text-xs text-gray-500 mb-2">Pilih area yang sudah diinput di menu Kelola Area untuk wilayah ini.</p>
                <div className="max-h-40 overflow-y-auto rounded-lg border border-gray-700 bg-gray-900 p-2 space-y-1">
                  {allAreas.length === 0 ? (
                    <p className="text-xs text-gray-500 text-center py-2">
                      Belum ada area. Tambah area di menu <a href="/admin/pppoe/areas" className="text-cyan-400 hover:underline">Kelola Area</a>.
                    </p>
                  ) : (
                    allAreas.map((area) => {
                      const isAssignedToOther = area.territoryId && area.territoryId !== editingTerritory?.id;
                      const isChecked = selectedAreaIds.includes(area.id);
                      return (
                        <label
                          key={area.id}
                          className={`flex items-center gap-2 rounded px-2 py-1 text-sm ${isAssignedToOther && !isChecked ? 'opacity-40' : 'hover:bg-gray-800'}`}
                        >
                          <input
                            type="checkbox"
                            checked={isChecked}
                            disabled={Boolean(isAssignedToOther && !isChecked)}
                            onChange={(e) => {
                              if (e.target.checked) {
                                setSelectedAreaIds([...selectedAreaIds, area.id]);
                              } else {
                                setSelectedAreaIds(selectedAreaIds.filter((id) => id !== area.id));
                              }
                            }}
                            className="rounded border-gray-700"
                          />
                          <span className="text-gray-300">{area.name}</span>
                          {area.description && <span className="text-xs text-gray-500">- {area.description}</span>}
                          {isAssignedToOther && !isChecked && <span className="text-xs text-gray-600 ml-auto">(wilayah lain)</span>}
                        </label>
                      );
                    })
                  )}
                </div>
                {selectedAreaIds.length > 0 && (
                  <p className="text-xs text-gray-500 mt-1">{selectedAreaIds.length} area dipilih</p>
                )}
              </div>
            </div>
          </ModalBody>
          <ModalFooter>
            <ModalButton type="button" variant="secondary" onClick={() => setIsDialogOpen(false)}>
              Batal
            </ModalButton>
            <ModalButton type="submit">{editingTerritory ? 'Simpan' : 'Buat Kolektor'}</ModalButton>
          </ModalFooter>
        </form>
      </SimpleModal>

      {/* Territory Areas Detail Modal */}
      <SimpleModal
        isOpen={!!selectedTerritory}
        onClose={() => setSelectedTerritory(null)}
      >
        <ModalHeader>
          <ModalTitle>Area Kolektor: {selectedTerritory?.name}</ModalTitle>
          <ModalDescription>Pilih area dari data yang sudah diinput di Kelola Area</ModalDescription>
        </ModalHeader>
        <ModalBody>
          <div className="space-y-3 mb-4">
            <div>
              <ModalLabel>Pilih Area</ModalLabel>
              <div className="flex gap-2">
                <select
                  value={selectedAreaId}
                  onChange={(e) => setSelectedAreaId(e.target.value)}
                  className="flex-1 rounded-lg border border-gray-700 bg-gray-900 px-3 py-2 text-sm text-white focus:border-cyan-500 focus:outline-none"
                >
                  <option value="">-- Pilih Area --</option>
                  {availableAreas.map((area) => (
                    <option key={area.id} value={area.id}>
                      {area.name}{area.description ? ` (${area.description})` : ''}
                    </option>
                  ))}
                </select>
                <button
                  type="button"
                  onClick={handleAddArea}
                  disabled={!selectedAreaId}
                  className="flex items-center gap-2 rounded-lg bg-cyan-500 px-3 py-1.5 text-sm font-medium text-black hover:bg-cyan-400 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  <Plus className="h-4 w-4" />
                  Tambah
                </button>
              </div>
              {availableAreas.length === 0 && (
                <p className="text-xs text-gray-500 mt-1">
                  Semua area sudah ditugaskan ke wilayah. Tambah area baru di menu Kelola Area.
                </p>
              )}
            </div>
          </div>

          <div className="space-y-2 max-h-60 overflow-y-auto">
            {selectedTerritory?.areas?.map((area) => (
              <div
                key={area.id}
                className="flex items-center justify-between rounded-lg border border-gray-800 bg-gray-900/50 px-3 py-2"
              >
                <div className="text-sm text-gray-300">
                  <span className="font-medium">{area.name}</span>
                  {area.description && <span className="text-gray-500"> - {area.description}</span>}
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
              <p className="text-center text-sm text-gray-500 py-4">Belum ada area ditugaskan</p>
            )}
          </div>
        </ModalBody>
      </SimpleModal>
    </div>
  );
}
