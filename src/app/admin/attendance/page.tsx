'use client';

import { useState, useEffect } from 'react';
import { showSuccess, showError, showConfirm } from '@/lib/sweetalert';
import { Plus, Trash2, Users, Search, Calendar, MapPin } from 'lucide-react';
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

interface AttendanceRecord {
  id: string;
  employeeId: string;
  date: string;
  checkIn?: string;
  checkOut?: string;
  status: string;
  notes?: string;
  locationLat?: number;
  locationLng?: number;
  createdAt: string;
}

const STATUS_OPTS = ['PRESENT', 'ABSENT', 'LATE', 'HALF_DAY', 'SICK', 'LEAVE'];
const STATUS_COLORS: Record<string, string> = {
  PRESENT: 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400',
  ABSENT: 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400',
  LATE: 'bg-orange-100 dark:bg-orange-900/30 text-orange-700 dark:text-orange-400',
  HALF_DAY: 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-400',
  SICK: 'bg-purple-100 dark:bg-purple-900/30 text-purple-700 dark:text-purple-400',
  LEAVE: 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-400',
};

export default function AttendancePage() {
  const [records, setRecords] = useState<AttendanceRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [filterDate, setFilterDate] = useState('');
  const [isOpen, setIsOpen] = useState(false);
  const [selected, setSelected] = useState<string[]>([]);
  const [form, setForm] = useState({
    employeeId: '',
    date: new Date().toISOString().slice(0, 10),
    status: 'PRESENT',
    checkIn: '',
    checkOut: '',
    notes: '',
  });

  const load = async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams({ page: '1', limit: '100' });
      if (filterDate) params.set('date', filterDate);
      const res = await fetch(`/api/admin/attendance?${params}`);
      const data = await res.json();
      setRecords(data.attendance || []);
    } catch {
      showError('Gagal', 'Gagal memuat data absensi');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, [filterDate]);

  const handleSave = async () => {
    if (!form.employeeId.trim()) return showError('Error', 'ID Karyawan wajib diisi');
    try {
      const res = await fetch('/api/admin/attendance', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          ...form,
          checkIn: form.checkIn ? new Date(`${form.date}T${form.checkIn}:00`).toISOString() : null,
          checkOut: form.checkOut ? new Date(`${form.date}T${form.checkOut}:00`).toISOString() : null,
        }),
      });
      if (!res.ok) throw new Error();
      showSuccess('Berhasil', 'Absensi ditambahkan');
      setIsOpen(false);
      load();
    } catch {
      showError('Gagal', 'Gagal menyimpan absensi');
    }
  };

  const handleBulkDelete = async () => {
    if (selected.length === 0) return;
    const ok = await showConfirm('Hapus Absensi?', `Hapus ${selected.length} data absensi?`);
    if (!ok) return;
    try {
      await fetch('/api/admin/attendance/bulk-delete', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ids: selected }),
      });
      showSuccess('Dihapus', 'Data absensi dihapus');
      setSelected([]);
      load();
    } catch {
      showError('Gagal', 'Gagal menghapus data');
    }
  };

  const toggleSelect = (id: string) =>
    setSelected(prev => prev.includes(id) ? prev.filter(x => x !== id) : [...prev, id]);

  const filtered = records.filter(r =>
    r.employeeId.toLowerCase().includes(search.toLowerCase()) ||
    r.date.includes(search)
  );

  const stats = {
    present: records.filter(r => r.status === 'PRESENT').length,
    absent: records.filter(r => r.status === 'ABSENT').length,
    late: records.filter(r => r.status === 'LATE').length,
  };

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="p-2 bg-blue-500/10 rounded-xl">
            <Users className="w-5 h-5 text-blue-500" />
          </div>
          <div>
            <h1 className="text-xl font-bold text-gray-900 dark:text-white">Manajemen Absensi</h1>
            <p className="text-sm text-gray-500 dark:text-gray-400">Kelola kehadiran karyawan & teknisi</p>
          </div>
        </div>
        <div className="flex gap-2">
          {selected.length > 0 && (
            <button onClick={handleBulkDelete} className="flex items-center gap-2 px-3 py-2 bg-red-500 hover:bg-red-600 text-white rounded-xl text-sm font-semibold transition-colors">
              <Trash2 className="w-4 h-4" />
              Hapus ({selected.length})
            </button>
          )}
          <button
            onClick={() => { setForm({ employeeId: '', date: new Date().toISOString().slice(0, 10), status: 'PRESENT', checkIn: '', checkOut: '', notes: '' }); setIsOpen(true); }}
            className="flex items-center gap-2 px-4 py-2 bg-brand-500 hover:bg-brand-600 text-white rounded-xl text-sm font-semibold transition-colors"
          >
            <Plus className="w-4 h-4" />
            Tambah Absensi
          </button>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-3 gap-4">
        {[
          { label: 'Hadir', count: stats.present, color: 'text-green-500', bg: 'bg-green-500/10' },
          { label: 'Tidak Hadir', count: stats.absent, color: 'text-red-500', bg: 'bg-red-500/10' },
          { label: 'Terlambat', count: stats.late, color: 'text-orange-500', bg: 'bg-orange-500/10' },
        ].map(s => (
          <div key={s.label} className="bg-white dark:bg-gray-800/50 rounded-2xl border border-gray-200 dark:border-gray-700 p-4">
            <p className={`text-2xl font-bold ${s.color}`}>{s.count}</p>
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">{s.label}</p>
          </div>
        ))}
      </div>

      {/* Filters */}
      <div className="flex gap-3 flex-wrap">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <input value={search} onChange={e => setSearch(e.target.value)} placeholder="Cari karyawan..."
            className="pl-9 pr-3 py-2 text-sm border border-gray-200 dark:border-gray-700 rounded-xl bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-brand-500" />
        </div>
        <div className="relative flex items-center">
          <Calendar className="absolute left-3 w-4 h-4 text-gray-400" />
          <input type="date" value={filterDate} onChange={e => setFilterDate(e.target.value)}
            className="pl-9 pr-3 py-2 text-sm border border-gray-200 dark:border-gray-700 rounded-xl bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-brand-500" />
        </div>
        {filterDate && (
          <button onClick={() => setFilterDate('')} className="px-3 py-2 text-sm text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 transition-colors">
            Reset Tanggal
          </button>
        )}
      </div>

      {/* Table */}
      {loading ? (
        <div className="text-center py-12 text-gray-400">Memuat data...</div>
      ) : filtered.length === 0 ? (
        <div className="text-center py-12 text-gray-400">
          <Users className="w-10 h-10 mx-auto mb-2 opacity-30" />
          <p>Belum ada data absensi</p>
        </div>
      ) : (
        <div className="bg-white dark:bg-gray-800/50 rounded-2xl border border-gray-200 dark:border-gray-700 overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800">
                <th className="px-4 py-3 w-8"><input type="checkbox" onChange={e => setSelected(e.target.checked ? filtered.map(r => r.id) : [])} className="rounded" /></th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-400">ID Karyawan</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-400">Tanggal</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-400">Status</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-400">Masuk</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-400">Keluar</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-400">Lokasi</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
              {filtered.map(r => (
                <tr key={r.id} className="hover:bg-gray-50 dark:hover:bg-gray-700/30 transition-colors">
                  <td className="px-4 py-3"><input type="checkbox" checked={selected.includes(r.id)} onChange={() => toggleSelect(r.id)} className="rounded" /></td>
                  <td className="px-4 py-3 font-mono text-xs text-gray-700 dark:text-gray-300">{r.employeeId}</td>
                  <td className="px-4 py-3 text-gray-600 dark:text-gray-400">{r.date}</td>
                  <td className="px-4 py-3">
                    <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${STATUS_COLORS[r.status] || 'bg-gray-100 text-gray-600'}`}>
                      {r.status}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-gray-600 dark:text-gray-400 text-xs">
                    {r.checkIn ? new Date(r.checkIn).toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' }) : '-'}
                  </td>
                  <td className="px-4 py-3 text-gray-600 dark:text-gray-400 text-xs">
                    {r.checkOut ? new Date(r.checkOut).toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' }) : '-'}
                  </td>
                  <td className="px-4 py-3 text-gray-400 text-xs">
                    {r.locationLat && r.locationLng ? (
                      <a href={`https://maps.google.com/?q=${r.locationLat},${r.locationLng}`} target="_blank" rel="noopener noreferrer"
                        className="flex items-center gap-1 text-brand-500 hover:underline">
                        <MapPin className="w-3 h-3" /> Maps
                      </a>
                    ) : '-'}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Modal */}
      <SimpleModal isOpen={isOpen} onClose={() => setIsOpen(false)}>
        <ModalHeader><ModalTitle>Tambah Data Absensi</ModalTitle></ModalHeader>
        <ModalBody>
          <div className="space-y-4">
            <div>
              <ModalLabel>ID Karyawan *</ModalLabel>
              <ModalInput value={form.employeeId} onChange={e => setForm(f => ({ ...f, employeeId: e.target.value }))} placeholder="ID Karyawan / Teknisi" />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <ModalLabel>Tanggal</ModalLabel>
                <ModalInput type="date" value={form.date} onChange={e => setForm(f => ({ ...f, date: e.target.value }))} />
              </div>
              <div>
                <ModalLabel>Status</ModalLabel>
                <ModalSelect value={form.status} onChange={e => setForm(f => ({ ...f, status: e.target.value }))}>
                  {STATUS_OPTS.map(s => <option key={s} value={s}>{s}</option>)}
                </ModalSelect>
              </div>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <ModalLabel>Jam Masuk</ModalLabel>
                <ModalInput type="time" value={form.checkIn} onChange={e => setForm(f => ({ ...f, checkIn: e.target.value }))} />
              </div>
              <div>
                <ModalLabel>Jam Keluar</ModalLabel>
                <ModalInput type="time" value={form.checkOut} onChange={e => setForm(f => ({ ...f, checkOut: e.target.value }))} />
              </div>
            </div>
            <div>
              <ModalLabel>Catatan</ModalLabel>
              <ModalTextarea value={form.notes} onChange={e => setForm(f => ({ ...f, notes: e.target.value }))} rows={2} placeholder="Keterangan tambahan..." />
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
