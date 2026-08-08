'use client';

import { useState, useEffect } from 'react';
import { Loader2, Users, Receipt, Wifi, AlertTriangle, Activity, TrendingUp, TrendingDown } from 'lucide-react';
import { usePermissions } from '@/hooks/usePermissions';
import { formatWIB } from '@/lib/timezone';

interface DashboardData {
  customers: { total: number; active: number; isolated: number; stopped: number };
  invoices: { total: number; pending: number; paid: number; overdue: number; revenue: number; month: string };
  network: { totalOLTs: number; onlineOLTs: number; totalONUs: number; onlineONUs: number; activeSessions: number };
  alerts: { unresolved: number; pendingRegs: number; openTickets: number };
  timestamp: string;
}

interface TrendData {
  date: string;
  newCustomers: number;
  paidInvoices: number;
  revenue: number;
  isolations: number;
}

export default function DashboardOverviewPage() {
  const { loading: permLoading } = usePermissions();
  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<DashboardData | null>(null);
  const [trends, setTrends] = useState<TrendData[]>([]);

  useEffect(() => {
    loadDashboard();
    loadTrends();
    const interval = setInterval(loadDashboard, 30000);
    return () => clearInterval(interval);
  }, []);

  const loadDashboard = async () => {
    try {
      const res = await fetch('/api/integration/dashboard');
      const d = await res.json();
      setData(d);
    } catch (error) {
      console.error('Load dashboard error:', error);
    } finally {
      setLoading(false);
    }
  };

  const loadTrends = async () => {
    try {
      const res = await fetch('/api/integration/dashboard/trends');
      const d = await res.json();
      setTrends(d.data || []);
    } catch (error) {
      console.error('Load trends error:', error);
    }
  };

  const formatCurrency = (amount: number) => {
    return new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', minimumFractionDigits: 0 }).format(amount);
  };

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleDateString('id-ID', { day: 'numeric', month: 'short' });
  };

  if (permLoading || loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <Loader2 className="h-8 w-8 animate-spin text-cyan-400" />
      </div>
    );
  }

  const maxRevenue = Math.max(...trends.map(t => t.revenue), 1);

  return (
    <div className="space-y-6 p-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Dashboard Real-Time</h1>
          <p className="text-sm text-gray-400">Overview sistem — auto-refresh setiap 30 detik</p>
        </div>
        {data && (
          <span className="text-xs text-gray-500">
            Update: {formatWIB(data.timestamp)}
          </span>
        )}
      </div>

      {/* Customer Stats */}
      {data && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="rounded-lg border border-gray-800 bg-gray-900/50 p-4">
            <div className="flex items-center gap-2">
              <Users className="h-4 w-4 text-cyan-400" />
              <p className="text-xs text-gray-400">Total Pelanggan</p>
            </div>
            <p className="mt-1 text-2xl font-bold text-white">{data.customers.total}</p>
          </div>
          <div className="rounded-lg border border-gray-800 bg-gray-900/50 p-4">
            <div className="flex items-center gap-2">
              <Activity className="h-4 w-4 text-green-400" />
              <p className="text-xs text-gray-400">Aktif</p>
            </div>
            <p className="mt-1 text-2xl font-bold text-green-400">{data.customers.active}</p>
          </div>
          <div className="rounded-lg border border-gray-800 bg-gray-900/50 p-4">
            <div className="flex items-center gap-2">
              <AlertTriangle className="h-4 w-4 text-yellow-400" />
              <p className="text-xs text-gray-400">Isolir</p>
            </div>
            <p className="mt-1 text-2xl font-bold text-yellow-400">{data.customers.isolated}</p>
          </div>
          <div className="rounded-lg border border-gray-800 bg-gray-900/50 p-4">
            <div className="flex items-center gap-2">
              <Users className="h-4 w-4 text-red-400" />
              <p className="text-xs text-gray-400">Berhenti</p>
            </div>
            <p className="mt-1 text-2xl font-bold text-red-400">{data.customers.stopped}</p>
          </div>
        </div>
      )}

      {/* Invoice Stats */}
      {data && (
        <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
          <div className="rounded-lg border border-gray-800 bg-gray-900/50 p-4">
            <div className="flex items-center gap-2">
              <Receipt className="h-4 w-4 text-cyan-400" />
              <p className="text-xs text-gray-400">Invoice {data.invoices.month}</p>
            </div>
            <p className="mt-1 text-2xl font-bold text-white">{data.invoices.total}</p>
          </div>
          <div className="rounded-lg border border-gray-800 bg-gray-900/50 p-4">
            <p className="text-xs text-gray-400">Pending</p>
            <p className="mt-1 text-2xl font-bold text-yellow-400">{data.invoices.pending}</p>
          </div>
          <div className="rounded-lg border border-gray-800 bg-gray-900/50 p-4">
            <p className="text-xs text-gray-400">Lunas</p>
            <p className="mt-1 text-2xl font-bold text-green-400">{data.invoices.paid}</p>
          </div>
          <div className="rounded-lg border border-gray-800 bg-gray-900/50 p-4">
            <p className="text-xs text-gray-400">Overdue</p>
            <p className="mt-1 text-2xl font-bold text-red-400">{data.invoices.overdue}</p>
          </div>
          <div className="rounded-lg border border-cyan-800/50 bg-cyan-900/10 p-4">
            <p className="text-xs text-gray-400">Pendapatan</p>
            <p className="mt-1 text-lg font-bold text-cyan-400">{formatCurrency(data.invoices.revenue)}</p>
          </div>
        </div>
      )}

      {/* Network Stats */}
      {data && (
        <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
          <div className="rounded-lg border border-gray-800 bg-gray-900/50 p-4">
            <div className="flex items-center gap-2">
              <Wifi className="h-4 w-4 text-cyan-400" />
              <p className="text-xs text-gray-400">Total OLT</p>
            </div>
            <p className="mt-1 text-2xl font-bold text-white">{data.network.totalOLTs}</p>
          </div>
          <div className="rounded-lg border border-gray-800 bg-gray-900/50 p-4">
            <p className="text-xs text-gray-400">OLT Online</p>
            <p className="mt-1 text-2xl font-bold text-green-400">{data.network.onlineOLTs}</p>
          </div>
          <div className="rounded-lg border border-gray-800 bg-gray-900/50 p-4">
            <p className="text-xs text-gray-400">Total ONU</p>
            <p className="mt-1 text-2xl font-bold text-white">{data.network.totalONUs}</p>
          </div>
          <div className="rounded-lg border border-gray-800 bg-gray-900/50 p-4">
            <p className="text-xs text-gray-400">ONU Online</p>
            <p className="mt-1 text-2xl font-bold text-green-400">{data.network.onlineONUs}</p>
          </div>
          <div className="rounded-lg border border-gray-800 bg-gray-900/50 p-4">
            <p className="text-xs text-gray-400">Sesi Aktif</p>
            <p className="mt-1 text-2xl font-bold text-cyan-400">{data.network.activeSessions}</p>
          </div>
        </div>
      )}

      {/* Alerts */}
      {data && (
        <div className="grid grid-cols-3 gap-4">
          <div className="rounded-lg border border-red-800/30 bg-red-900/10 p-4">
            <div className="flex items-center gap-2">
              <AlertTriangle className="h-4 w-4 text-red-400" />
              <p className="text-xs text-gray-400">Alert OLT Belum Selesai</p>
            </div>
            <p className="mt-1 text-2xl font-bold text-red-400">{data.alerts.unresolved}</p>
          </div>
          <div className="rounded-lg border border-yellow-800/30 bg-yellow-900/10 p-4">
            <div className="flex items-center gap-2">
              <Users className="h-4 w-4 text-yellow-400" />
              <p className="text-xs text-gray-400">PSB Pending</p>
            </div>
            <p className="mt-1 text-2xl font-bold text-yellow-400">{data.alerts.pendingRegs}</p>
          </div>
          <div className="rounded-lg border border-gray-800 bg-gray-900/50 p-4">
            <div className="flex items-center gap-2">
              <Receipt className="h-4 w-4 text-gray-400" />
              <p className="text-xs text-gray-400">Tiket Terbuka</p>
            </div>
            <p className="mt-1 text-2xl font-bold text-white">{data.alerts.openTickets}</p>
          </div>
        </div>
      )}

      {/* 7-Day Trends */}
      {trends.length > 0 && (
        <div className="rounded-lg border border-gray-800 bg-gray-900/50 p-4">
          <h3 className="text-sm font-medium text-white mb-4">Tren 7 Hari Terakhir</h3>
          <div className="space-y-3">
            {trends.map((t) => (
              <div key={t.date} className="flex items-center gap-4">
                <span className="text-xs text-gray-400 w-16">{formatDate(t.date)}</span>
                <div className="flex-1 flex items-center gap-3">
                  <div className="flex items-center gap-1 text-xs">
                    <TrendingUp className="h-3 w-3 text-green-400" />
                    <span className="text-green-400">{t.newCustomers} pelanggan</span>
                  </div>
                  <div className="flex items-center gap-1 text-xs">
                    <Receipt className="h-3 w-3 text-cyan-400" />
                    <span className="text-cyan-400">{t.paidInvoices} invoice</span>
                  </div>
                  <div className="flex items-center gap-1 text-xs">
                    <span className="text-white">{formatCurrency(t.revenue)}</span>
                  </div>
                  {t.isolations > 0 && (
                    <div className="flex items-center gap-1 text-xs">
                      <TrendingDown className="h-3 w-3 text-red-400" />
                      <span className="text-red-400">{t.isolations} isolir</span>
                    </div>
                  )}
                </div>
                <div className="w-32 bg-gray-800 rounded-full h-2 overflow-hidden">
                  <div
                    className="bg-cyan-500 h-full rounded-full transition-all"
                    style={{ width: `${(t.revenue / maxRevenue) * 100}%` }}
                  />
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
