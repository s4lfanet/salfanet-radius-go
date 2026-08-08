'use client';

import { useState, useEffect } from 'react';
import { Calendar, CheckCircle, Loader2, Download, User } from 'lucide-react';
import { showSuccess, showError } from '@/lib/sweetalert';
import { usePermissions } from '@/hooks/usePermissions';
import { useTranslation } from '@/hooks/useTranslation';

interface Collector {
  id: string;
  name: string;
  email: string;
}

interface InvoiceRow {
  id: string;
  invoiceNumber: string;
  amount: number;
  customerName: string | null;
  paidAt: string | null;
  method: string;
}

interface SettlementData {
  date: string;
  collectorId: string;
  totalAmount: number;
  invoiceCount: number;
  invoices: InvoiceRow[];
  settlement: boolean;
  settlementData: any;
}

interface DaySummary {
  date: string;
  count: number;
  total: number;
}

interface CollectorSummary {
  collectorId: string;
  collectorName: string | null;
  count: number;
  total: number;
}

export default function SettlementsPage() {
  const { hasPermission, loading: permLoading } = usePermissions();
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [collectors, setCollectors] = useState<Collector[]>([]);
  const [selectedCollector, setSelectedCollector] = useState('');
  const [date, setDate] = useState(new Date().toISOString().split('T')[0]);
  const [fromDate, setFromDate] = useState('');
  const [toDate, setToDate] = useState('');
  const [settlement, setSettlement] = useState<SettlementData | null>(null);
  const [rangeData, setRangeData] = useState<{ daily: DaySummary[]; collectors: CollectorSummary[] } | null>(null);
  const [activeTab, setActiveTab] = useState<'daily' | 'range'>('daily');

  useEffect(() => {
    loadCollectors();
  }, []);

  const loadCollectors = async () => {
    try {
      const res = await fetch('/api/territories/collectors');
      const data = await res.json();
      setCollectors(data.data || []);
    } catch (error) {
      console.error('Load collectors error:', error);
    }
  };

  const loadDailySettlement = async () => {
    if (!selectedCollector || !date) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/settlements?collectorId=${selectedCollector}&date=${date}`);
      const data = await res.json();
      setSettlement(data);
    } catch (error) {
      showError('Gagal memuat data setoran');
    } finally {
      setLoading(false);
    }
  };

  const loadRangeSettlement = async () => {
    if (!fromDate || !toDate) return;
    setLoading(true);
    try {
      const params = new URLSearchParams({ from: fromDate, to: toDate });
      if (selectedCollector) params.set('collectorId', selectedCollector);
      const res = await fetch(`/api/settlements/range?${params}`);
      const data = await res.json();
      setRangeData(data);
    } catch (error) {
      showError('Gagal memuat data setoran');
    } finally {
      setLoading(false);
    }
  };

  const handleConfirm = async () => {
    if (!selectedCollector || !date) return;
    try {
      const res = await fetch('/api/settlements/confirm', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ collectorId: selectedCollector, date }),
      });
      if (!res.ok) throw new Error('Failed to confirm');
      showSuccess('Settlement dikonfirmasi');
      loadDailySettlement();
    } catch (error) {
      showError('Gagal mengkonfirmasi settlement');
    }
  };

  const formatCurrency = (amount: number) => {
    return new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', minimumFractionDigits: 0 }).format(amount);
  };

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleDateString('id-ID', { day: 'numeric', month: 'long', year: 'numeric' });
  };

  if (permLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <Loader2 className="h-8 w-8 animate-spin text-cyan-400" />
      </div>
    );
  }

  return (
    <div className="space-y-6 p-6">
      <div>
        <h1 className="text-2xl font-bold text-white">Setoran Kolektor</h1>
        <p className="text-sm text-gray-400">Rekap setoran dan konfirmasi pembayaran per kolektor</p>
      </div>

      <div className="flex gap-2 border-b border-gray-800">
        <button
          onClick={() => setActiveTab('daily')}
          className={`px-4 py-2 text-sm font-medium ${activeTab === 'daily' ? 'border-b-2 border-cyan-500 text-cyan-400' : 'text-gray-400 hover:text-white'}`}
        >
          Harian
        </button>
        <button
          onClick={() => setActiveTab('range')}
          className={`px-4 py-2 text-sm font-medium ${activeTab === 'range' ? 'border-b-2 border-cyan-500 text-cyan-400' : 'text-gray-400 hover:text-white'}`}
        >
          Rentang Tanggal
        </button>
      </div>

      <div className="rounded-lg border border-gray-800 bg-gray-900/50 p-4">
        <div className="flex flex-wrap items-end gap-4">
          <div>
            <label className="block text-xs text-gray-400 mb-1">Kolektor</label>
            <select
              value={selectedCollector}
              onChange={(e) => setSelectedCollector(e.target.value)}
              className="rounded-lg border border-gray-700 bg-gray-900 px-3 py-2 text-sm text-white focus:border-cyan-500 focus:outline-none"
            >
              <option value="">Semua Kolektor</option>
              {collectors.map((c) => (
                <option key={c.id} value={c.id}>{c.name}</option>
              ))}
            </select>
          </div>

          {activeTab === 'daily' ? (
            <>
              <div>
                <label className="block text-xs text-gray-400 mb-1">Tanggal</label>
                <input
                  type="date"
                  value={date}
                  onChange={(e) => setDate(e.target.value)}
                  className="rounded-lg border border-gray-700 bg-gray-900 px-3 py-2 text-sm text-white focus:border-cyan-500 focus:outline-none"
                />
              </div>
              <button
                onClick={loadDailySettlement}
                disabled={loading || !selectedCollector}
                className="flex items-center gap-2 rounded-lg bg-cyan-500 px-4 py-2 text-sm font-medium text-black hover:bg-cyan-400 disabled:opacity-50"
              >
                {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Calendar className="h-4 w-4" />}
                Tampilkan
              </button>
            </>
          ) : (
            <>
              <div>
                <label className="block text-xs text-gray-400 mb-1">Dari Tanggal</label>
                <input
                  type="date"
                  value={fromDate}
                  onChange={(e) => setFromDate(e.target.value)}
                  className="rounded-lg border border-gray-700 bg-gray-900 px-3 py-2 text-sm text-white focus:border-cyan-500 focus:outline-none"
                />
              </div>
              <div>
                <label className="block text-xs text-gray-400 mb-1">Sampai Tanggal</label>
                <input
                  type="date"
                  value={toDate}
                  onChange={(e) => setToDate(e.target.value)}
                  className="rounded-lg border border-gray-700 bg-gray-900 px-3 py-2 text-sm text-white focus:border-cyan-500 focus:outline-none"
                />
              </div>
              <button
                onClick={loadRangeSettlement}
                disabled={loading || !fromDate || !toDate}
                className="flex items-center gap-2 rounded-lg bg-cyan-500 px-4 py-2 text-sm font-medium text-black hover:bg-cyan-400 disabled:opacity-50"
              >
                {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Calendar className="h-4 w-4" />}
                Tampilkan
              </button>
            </>
          )}
        </div>
      </div>

      {/* Daily Settlement Result */}
      {activeTab === 'daily' && settlement && (
        <div className="space-y-4">
          <div className="grid grid-cols-3 gap-4">
            <div className="rounded-lg border border-gray-800 bg-gray-900/50 p-4">
              <p className="text-xs text-gray-400">Total Setoran</p>
              <p className="mt-1 text-xl font-bold text-cyan-400">{formatCurrency(settlement.totalAmount)}</p>
            </div>
            <div className="rounded-lg border border-gray-800 bg-gray-900/50 p-4">
              <p className="text-xs text-gray-400">Jumlah Invoice</p>
              <p className="mt-1 text-xl font-bold text-white">{settlement.invoiceCount}</p>
            </div>
            <div className="rounded-lg border border-gray-800 bg-gray-900/50 p-4">
              <p className="text-xs text-gray-400">Status</p>
              <p className={`mt-1 text-xl font-bold ${settlement.settlement ? 'text-green-400' : 'text-yellow-400'}`}>
                {settlement.settlement ? 'Dikonfirmasi' : 'Pending'}
              </p>
            </div>
          </div>

          {!settlement.settlement && settlement.invoiceCount > 0 && (
            <button
              onClick={handleConfirm}
              className="flex items-center gap-2 rounded-lg bg-green-600 px-4 py-2 text-sm font-medium text-white hover:bg-green-500"
            >
              <CheckCircle className="h-4 w-4" />
              Konfirmasi Settlement
            </button>
          )}

          <div className="rounded-lg border border-gray-800 bg-gray-900/50 overflow-hidden">
            <table className="w-full text-sm">
              <thead className="border-b border-gray-800 bg-gray-900">
                <tr>
                  <th className="px-4 py-2 text-left text-gray-400">Invoice</th>
                  <th className="px-4 py-2 text-left text-gray-400">Pelanggan</th>
                  <th className="px-4 py-2 text-right text-gray-400">Jumlah</th>
                  <th className="px-4 py-2 text-left text-gray-400">Metode</th>
                  <th className="px-4 py-2 text-left text-gray-400">Tanggal Bayar</th>
                </tr>
              </thead>
              <tbody>
                {settlement.invoices.map((inv) => (
                  <tr key={inv.id} className="border-b border-gray-800/50">
                    <td className="px-4 py-2 text-white">{inv.invoiceNumber}</td>
                    <td className="px-4 py-2 text-gray-300">{inv.customerName || '-'}</td>
                    <td className="px-4 py-2 text-right text-white">{formatCurrency(inv.amount)}</td>
                    <td className="px-4 py-2 text-gray-300">{inv.method || '-'}</td>
                    <td className="px-4 py-2 text-gray-400">{inv.paidAt ? formatDate(inv.paidAt) : '-'}</td>
                  </tr>
                ))}
                {settlement.invoices.length === 0 && (
                  <tr>
                    <td colSpan={5} className="px-4 py-8 text-center text-gray-500">
                      Tidak ada invoice lunas pada tanggal ini
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Range Settlement Result */}
      {activeTab === 'range' && rangeData && (
        <div className="space-y-4">
          {rangeData.collectors && rangeData.collectors.length > 0 && (
            <div className="rounded-lg border border-gray-800 bg-gray-900/50 overflow-hidden">
              <h3 className="px-4 py-3 text-sm font-medium text-white border-b border-gray-800">Rekap per Kolektor</h3>
              <table className="w-full text-sm">
                <thead className="border-b border-gray-800 bg-gray-900">
                  <tr>
                    <th className="px-4 py-2 text-left text-gray-400">Kolektor</th>
                    <th className="px-4 py-2 text-right text-gray-400">Invoice</th>
                    <th className="px-4 py-2 text-right text-gray-400">Total</th>
                  </tr>
                </thead>
                <tbody>
                  {rangeData.collectors.map((c) => (
                    <tr key={c.collectorId} className="border-b border-gray-800/50">
                      <td className="px-4 py-2 text-white">{c.collectorName || 'Unknown'}</td>
                      <td className="px-4 py-2 text-right text-gray-300">{c.count}</td>
                      <td className="px-4 py-2 text-right text-white">{formatCurrency(c.total)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          <div className="rounded-lg border border-gray-800 bg-gray-900/50 overflow-hidden">
            <h3 className="px-4 py-3 text-sm font-medium text-white border-b border-gray-800">Rekap Harian</h3>
            <table className="w-full text-sm">
              <thead className="border-b border-gray-800 bg-gray-900">
                <tr>
                  <th className="px-4 py-2 text-left text-gray-400">Tanggal</th>
                  <th className="px-4 py-2 text-right text-gray-400">Invoice</th>
                  <th className="px-4 py-2 text-right text-gray-400">Total</th>
                </tr>
              </thead>
              <tbody>
                {rangeData.daily.map((d) => (
                  <tr key={d.date} className="border-b border-gray-800/50">
                    <td className="px-4 py-2 text-white">{formatDate(d.date)}</td>
                    <td className="px-4 py-2 text-right text-gray-300">{d.count}</td>
                    <td className="px-4 py-2 text-right text-white">{formatCurrency(d.total)}</td>
                  </tr>
                ))}
                {rangeData.daily.length === 0 && (
                  <tr>
                    <td colSpan={3} className="px-4 py-8 text-center text-gray-500">
                      Tidak ada data pada rentang ini
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
