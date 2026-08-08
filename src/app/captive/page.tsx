'use client';

import { useState, useEffect } from 'react';
import { Loader2, Wifi, WifiOff, Receipt, CreditCard, Phone, User, AlertCircle } from 'lucide-react';

interface CaptiveUser {
  identified: boolean;
  username?: string;
  user?: {
    id: string;
    name: string | null;
    customerId: string | null;
    phone: string | null;
    status: string;
  };
  invoices?: Array<{
    id: string;
    invoiceNo: string;
    amount: number;
    dueDate: string;
    status: string;
  }>;
  message?: string;
}

export default function CaptivePortalPage() {
  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<CaptiveUser | null>(null);
  const [error, setError] = useState('');

  useEffect(() => {
    identifyUser();
  }, []);

  const identifyUser = async () => {
    try {
      // Try to get client IP from browser
      const ipRes = await fetch('https://api.ipify.org?format=json');
      const ipData = await ipRes.json();

      const res = await fetch(`/api/captive/identify?ip=${ipData.ip}`);
      const d = await res.json();
      setData(d);
    } catch (err) {
      setError('Gagal mengidentifikasi sesi. Pastikan Anda terhubung ke jaringan.');
    } finally {
      setLoading(false);
    }
  };

  const formatCurrency = (amount: number) => {
    return new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', minimumFractionDigits: 0 }).format(amount);
  };

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleDateString('id-ID', { day: 'numeric', month: 'long', year: 'numeric' });
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen bg-gradient-to-br from-slate-900 to-slate-800">
        <Loader2 className="h-8 w-8 animate-spin text-cyan-400" />
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 to-slate-800 text-white">
      <div className="container mx-auto max-w-md px-4 py-8">
        {/* Header */}
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-red-900/30 mb-3">
            <WifiOff className="h-8 w-8 text-red-400" />
          </div>
          <h1 className="text-xl font-bold">Layanan Anda Diisolir</h1>
          <p className="text-sm text-gray-400 mt-1">Lunasi tagihan untuk mengaktifkan kembali internet</p>
        </div>

        {error && (
          <div className="rounded-lg border border-red-800 bg-red-900/20 p-4 mb-4">
            <div className="flex items-center gap-2 text-red-400">
              <AlertCircle className="h-4 w-4" />
              <p className="text-sm">{error}</p>
            </div>
          </div>
        )}

        {data && !data.identified && (
          <div className="rounded-lg border border-gray-800 bg-gray-900/50 p-6 text-center">
            <Wifi className="h-8 w-8 text-gray-500 mx-auto mb-2" />
            <p className="text-gray-400 text-sm">{data.message || 'Sesi tidak ditemukan'}</p>
          </div>
        )}

        {data && data.identified && (
          <>
            {/* User Info */}
            <div className="rounded-lg border border-gray-800 bg-gray-900/50 p-4 mb-4">
              <div className="flex items-center gap-3 mb-3">
                <div className="w-10 h-10 rounded-full bg-cyan-900/30 flex items-center justify-center">
                  <User className="h-5 w-5 text-cyan-400" />
                </div>
                <div>
                  <p className="font-medium text-white">{data.user?.name || data.username}</p>
                  <p className="text-xs text-gray-400">{data.username}</p>
                </div>
              </div>
              {data.user?.phone && (
                <div className="flex items-center gap-2 text-sm text-gray-400">
                  <Phone className="h-3 w-3" />
                  {data.user.phone}
                </div>
              )}
            </div>

            {/* Invoices */}
            {data.invoices && data.invoices.length > 0 ? (
              <div className="space-y-3">
                <h2 className="text-sm font-medium text-gray-400 flex items-center gap-2">
                  <Receipt className="h-4 w-4" /> Tagihan Belum Lunas
                </h2>
                {data.invoices.map((inv) => (
                  <div key={inv.id} className="rounded-lg border border-gray-800 bg-gray-900/50 p-4">
                    <div className="flex items-center justify-between mb-2">
                      <span className="text-xs text-gray-500 font-mono">{inv.invoiceNo}</span>
                      <span className={`text-xs px-2 py-0.5 rounded ${
                        inv.status === 'OVERDUE' ? 'bg-red-900/50 text-red-400' : 'bg-yellow-900/50 text-yellow-400'
                      }`}>
                        {inv.status === 'OVERDUE' ? 'TERLAMBAT' : 'PENDING'}
                      </span>
                    </div>
                    <p className="text-2xl font-bold text-white mb-1">{formatCurrency(inv.amount)}</p>
                    <p className="text-xs text-gray-400 mb-3">Jatuh tempo: {formatDate(inv.dueDate)}</p>
                    <button
                      onClick={() => window.location.href = `/customer/invoices`}
                      className="w-full rounded-lg bg-cyan-600 hover:bg-cyan-500 text-white py-2 text-sm font-medium flex items-center justify-center gap-2 transition-colors"
                    >
                      <CreditCard className="h-4 w-4" /> Bayar Sekarang
                    </button>
                  </div>
                ))}
              </div>
            ) : (
              <div className="rounded-lg border border-green-800 bg-green-900/20 p-6 text-center">
                <Wifi className="h-8 w-8 text-green-400 mx-auto mb-2" />
                <p className="text-green-400 text-sm">Tidak ada tagihan tertunggak</p>
                <p className="text-xs text-gray-400 mt-1">Internet Anda akan aktif kembali dalam 1-2 menit</p>
              </div>
            )}

            {/* Footer */}
            <div className="mt-6 text-center">
              <p className="text-xs text-gray-500">
                Butuh bantuan? Hubungi customer service ISP Anda
              </p>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
