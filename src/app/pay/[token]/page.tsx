'use client';
import { showError } from '@/lib/sweetalert';
import { formatWIB } from '@/lib/timezone';
import { useEffect, useState, useCallback, useRef } from 'react';
import { useParams } from 'next/navigation';
import { Wifi, CheckCircle, Clock, AlertCircle, CreditCard, Building2, Loader2, User, Phone, Package, Calendar, MapPin, Router, Network, Mail, Hash, Zap, QrCode, X, ExternalLink, RefreshCw } from 'lucide-react';
import { QRCodeSVG } from 'qrcode.react';

interface Invoice {
  id: string;
  invoiceNumber: string;
  customerName: string;
  customerPhone: string;
  amount: number;
  status: string;
  dueDate: string;
  createdAt: string;
  paidAt: string | null;
  user: {
    name: string;
    phone: string;
    email: string | null;
    username: string;
    address: string | null;
    customerId: string | null;
    subscriptionType: string;
    status: string;
    profile: { name: string; price: number; downloadSpeed: number; uploadSpeed: number; } | null;
    area: { name: string; } | null;
    router: { shortname: string; } | null;
  } | null;
}

interface PaymentGateway { id: string; name: string; provider: string; isActive: boolean; }
interface CompanySetting { name: string; address: string | null; phone: string | null; email: string | null; }

export default function PaymentPage() {
  const params = useParams();
  const token = params.token as string;
  const [loading, setLoading] = useState(true);
  const [invoice, setInvoice] = useState<Invoice | null>(null);
  const [paymentGateways, setPaymentGateways] = useState<PaymentGateway[]>([]);
  const [company, setCompany] = useState<CompanySetting | null>(null);
  const [qrisOwn, setQrisOwn] = useState<{ enabled: boolean; merchantName: string; hasListener?: boolean } | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [processing, setProcessing] = useState(false);
  const [duitkuMethods, setDuitkuMethods] = useState<{ code: string; name: string; group: string }[]>([]);
  const [loadingDuitkuMethods, setLoadingDuitkuMethods] = useState(false);

  // QRIS Dynamic States
  const [qrisData, setQrisData] = useState<{ qrString: string; paymentUrl: string; orderId: string; gateway: string; isQrisOwn?: boolean; uniqueAmount?: number; hasListener?: boolean } | null>(null);
  const [qrisStatus, setQrisStatus] = useState<'pending' | 'paid' | 'expired' | 'failed'>('pending');
  const [qrisCountdown, setQrisCountdown] = useState(1440); // 24 min default (seconds)
  const qrisPollingRef = useRef<NodeJS.Timeout | null>(null);
  const qrisCountdownRef = useRef<NodeJS.Timeout | null>(null);

  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => { loadInvoice(); }, [token]);

  const loadInvoice = async () => {
    try {
      setLoading(true);
      const res = await fetch(`/api/invoices/by-token/${token}`);
      const data = await res.json();
      if (!res.ok) { setError(data.error || 'Failed to load invoice'); return; }
      setInvoice(data.invoice);
      setPaymentGateways(data.paymentGateways || []);
      setCompany(data.company || null);
      setQrisOwn(data.qrisOwn || null);
      // If Duitku is in the list, fetch its payment methods
      if ((data.paymentGateways || []).some((g: PaymentGateway) => g.provider === 'duitku')) {
        fetchDuitkuMethods(data.invoice?.amount || 10000);
      }
    } catch (err) { setError('Failed to load invoice'); } finally { setLoading(false); }
  };

  const fetchDuitkuMethods = async (amount: number) => {
    setLoadingDuitkuMethods(true);
    try {
      const res = await fetch(`/api/payment/duitku-methods?amount=${amount}`);
      const data = await res.json();
      setDuitkuMethods(data.methods || []);
    } catch {
      // Use empty = will show nothing for Duitku methods
    } finally {
      setLoadingDuitkuMethods(false);
    }
  };

  const formatCurrency = (amount: number) => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', minimumFractionDigits: 0 }).format(amount);
  const formatDate = (dateStr: string) => formatWIB(dateStr, 'd MMM yyyy');

  const getStatusBadge = (status: string) => {
    const styles: Record<string, string> = {
      PAID: 'bg-[#00ff88]/20 text-[#00ff88] border border-[#00ff88]/40',
      PENDING: 'bg-[#ff44cc]/20 text-[#ff44cc] border border-[#ff44cc]/40',
      OVERDUE: 'bg-[#ff4466]/20 text-[#ff6b8a] border border-[#ff4466]/40'
    };
    const icons: Record<string, React.ReactNode> = { PAID: <CheckCircle className="w-3 h-3" />, PENDING: <Clock className="w-3 h-3" />, OVERDUE: <AlertCircle className="w-3 h-3" /> };
    return <span className={`inline-flex items-center gap-1 px-2.5 py-1 text-[10px] font-bold rounded-lg ${styles[status] || 'bg-gray-100'}`}>{icons[status]} {status}</span>;
  };

  // Cleanup QRIS polling
  const cleanupQrisPolling = useCallback(() => {
    if (qrisPollingRef.current) { clearInterval(qrisPollingRef.current); qrisPollingRef.current = null; }
    if (qrisCountdownRef.current) { clearInterval(qrisCountdownRef.current); qrisCountdownRef.current = null; }
  }, []);

  // Start QRIS polling
  const startQrisPolling = useCallback((orderId: string, isQrisOwn = false) => {
    cleanupQrisPolling();
    setQrisStatus('pending');
    // 15 min for qris_own (pending expires), 24 min for third-party gateways
    setQrisCountdown(isQrisOwn ? 900 : 1440);

    // Poll every 5 seconds
    qrisPollingRef.current = setInterval(async () => {
      try {
        // Gunakan qris-status untuk qris_own (lebih akurat), check-order untuk gateway lain
        const url = isQrisOwn
          ? `/api/payment/qris-status?orderId=${encodeURIComponent(orderId)}`
          : `/api/payment/check-order?orderId=${encodeURIComponent(orderId)}`;
        const res = await fetch(url);
        const data = await res.json();
        const status = data.status || '';
        if (status === 'settlement' || status === 'paid' || status === 'PAID') {
          setQrisStatus('paid');
          cleanupQrisPolling();
          // Reload invoice after 3 seconds
          setTimeout(() => { setQrisData(null); loadInvoice(); }, 3000);
        } else if (status === 'expired' || status === 'expire') {
          setQrisStatus('expired');
          cleanupQrisPolling();
        } else if (status === 'failed' || status === 'cancel') {
          setQrisStatus('failed');
          cleanupQrisPolling();
        }
      } catch { /* silent */ }
    }, 5000);

    // Countdown timer
    qrisCountdownRef.current = setInterval(() => {
      setQrisCountdown(prev => {
        if (prev <= 1) { cleanupQrisPolling(); setQrisStatus('expired'); return 0; }
        return prev - 1;
      });
    }, 1000);
  }, [cleanupQrisPolling]);

  // Cleanup on unmount
  useEffect(() => { return () => cleanupQrisPolling(); }, [cleanupQrisPolling]);

  const handlePayment = async (gateway: string, paymentMethod?: string) => {
    if (!invoice) return;
    setProcessing(true);
    try {
      const body: any = { invoiceId: invoice.id, gateway };
      if (paymentMethod) body.paymentMethod = paymentMethod;
      const res = await fetch('/api/payment/create', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
      const data = await res.json();
      if (!res.ok) { await showError(data.error || 'Failed'); return; }

      // If QR string available, show QRIS inline instead of redirect
      if (data.qrString) {
        const isOwn = data.isQrisOwn || false;
        const hasListener = !!(qrisOwn?.hasListener);
        setQrisData({
          qrString: data.qrString,
          paymentUrl: data.paymentUrl || '',
          orderId: data.orderId,
          gateway,
          isQrisOwn: isOwn,
          uniqueAmount: data.uniqueAmount,
          hasListener,
        });
        // Polling: untuk gateway pihak ke-3, atau qris_own dengan Android listener
        if (!isOwn || hasListener) {
          startQrisPolling(data.orderId, isOwn);
        }
        return;
      }

      if (data.paymentUrl) window.location.href = data.paymentUrl; else await showError('Payment URL not available');
    } catch { await showError('Failed to process payment'); } finally { setProcessing(false); }
  };

  const closeQris = () => { cleanupQrisPolling(); setQrisData(null); setQrisStatus('pending'); };
  const formatCountdown = (s: number) => `${Math.floor(s / 60).toString().padStart(2, '0')}:${(s % 60).toString().padStart(2, '0')}`;

  if (loading) return (
    <div className="min-h-screen bg-[#1a0f35] relative overflow-hidden flex items-center justify-center">
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        <div className="absolute top-1/4 left-1/4 w-96 h-96 bg-[#bc13fe]/20 rounded-full blur-3xl animate-pulse"></div>
        <div className="absolute bottom-1/4 right-1/4 w-96 h-96 bg-[#00f7ff]/20 rounded-full blur-3xl animate-pulse delay-1000"></div>
      </div>
      <div className="text-center relative z-10">
        <Loader2 className="w-10 h-10 animate-spin mx-auto text-[#00f7ff] drop-shadow-[0_0_20px_rgba(0,247,255,0.6)] mb-3" />
        <p className="text-xs text-[#e0d0ff]/70">Loading...</p>
      </div>
    </div>
  );

  if (error || !invoice) return (
    <div className="min-h-screen bg-[#1a0f35] relative overflow-hidden flex items-center justify-center p-4">
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        <div className="absolute top-1/4 left-1/4 w-96 h-96 bg-[#ff4466]/20 rounded-full blur-3xl"></div>
      </div>
      <div className="relative z-10 bg-[#1a0f35]/80 backdrop-blur-xl rounded-2xl border-2 border-[#ff4466]/50 p-6 max-w-sm w-full text-center shadow-[0_0_50px_rgba(255,68,102,0.2)]">
        <AlertCircle className="w-12 h-12 text-[#ff6b8a] mx-auto mb-3 drop-shadow-[0_0_15px_rgba(255,68,102,0.5)]" />
        <h2 className="text-base font-bold text-white mb-1">Tagihan Tidak Ditemukan</h2>
        <p className="text-xs text-[#e0d0ff]/70">{error || 'Link pembayaran tidak valid atau sudah kadaluarsa.'}</p>
      </div>
    </div>
  );

  if (invoice.status === 'PAID') return (
    <div className="min-h-screen bg-[#1a0f35] relative overflow-hidden flex items-center justify-center p-4">
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        <div className="absolute top-1/4 left-1/4 w-96 h-96 bg-[#00ff88]/20 rounded-full blur-3xl"></div>
        <div className="absolute bottom-1/4 right-1/4 w-96 h-96 bg-[#00f7ff]/20 rounded-full blur-3xl"></div>
      </div>
      <div className="relative z-10 bg-[#1a0f35]/80 backdrop-blur-xl rounded-2xl border-2 border-[#00ff88]/50 p-6 max-w-sm w-full text-center shadow-[0_0_50px_rgba(0,255,136,0.2)]">
        <div className="w-14 h-14 bg-[#00ff88]/20 rounded-full flex items-center justify-center mx-auto mb-4 border-2 border-[#00ff88]/50 shadow-[0_0_30px_rgba(0,255,136,0.3)]">
          <CheckCircle className="w-7 h-7 text-[#00ff88] drop-shadow-[0_0_10px_rgba(0,255,136,0.8)]" />
        </div>
        <h2 className="text-base font-bold text-white mb-1">Pembayaran Diterima</h2>
        <p className="text-xs text-[#e0d0ff]/70 mb-4">Tagihan ini sudah dibayar</p>
        <div className="bg-[#0a0520]/50 rounded-xl p-4 text-left space-y-2">
          <div className="flex justify-between text-xs"><span className="text-[#e0d0ff]/60">Tagihan</span><span className="font-mono font-bold text-white">{invoice.invoiceNumber}</span></div>
          <div className="flex justify-between text-xs"><span className="text-[#e0d0ff]/60">Jumlah</span><span className="font-bold text-[#00ff88]">{formatCurrency(invoice.amount)}</span></div>
          {invoice.paidAt && <div className="flex justify-between text-xs"><span className="text-[#e0d0ff]/60">Dibayar</span><span className="text-white">{formatDate(invoice.paidAt)}</span></div>}
        </div>
      </div>
    </div>
  );

  return (
    <div className="min-h-screen bg-[#1a0f35] relative py-6 px-4">
      {/* Background */}
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        <div className="absolute top-0 left-1/4 w-96 h-96 bg-[#bc13fe]/15 rounded-full blur-3xl"></div>
        <div className="absolute top-1/3 right-1/4 w-96 h-96 bg-[#00f7ff]/15 rounded-full blur-3xl"></div>
        <div className="absolute bottom-0 left-1/2 w-96 h-96 bg-[#ff44cc]/15 rounded-full blur-3xl"></div>
        <div className="absolute inset-0 bg-[linear-gradient(rgba(188,19,254,0.02)_1px,transparent_1px),linear-gradient(90deg,rgba(188,19,254,0.02)_1px,transparent_1px)] bg-[size:50px_50px]"></div>
      </div>

      <div className="max-w-lg mx-auto space-y-4 relative z-10">
        {/* Header */}
        <div className="text-center">
          <div className="inline-flex items-center gap-2 px-4 py-2 bg-gradient-to-r from-[#bc13fe] to-[#00f7ff] rounded-full mb-3 shadow-[0_0_20px_rgba(188,19,254,0.4)]">
            <Wifi className="w-4 h-4 text-white" />
            <span className="text-xs font-bold text-white">Tagihan Pembayaran</span>
          </div>
          <p className="text-xs text-[#e0d0ff]/70">Silakan periksa detail tagihan Anda di bawah ini</p>
        </div>

        {/* Invoice Card */}
        <div className="bg-[#1a0f35]/80 backdrop-blur-xl rounded-2xl border-2 border-[#bc13fe]/30 overflow-hidden shadow-[0_0_30px_rgba(188,19,254,0.15)]">
          <div className="bg-gradient-to-r from-[#bc13fe] to-[#00f7ff] px-4 py-3">
            <div className="flex items-center justify-between">
              <span className="text-sm font-bold text-white">Detail Tagihan</span>
              {getStatusBadge(invoice.status)}
            </div>
          </div>
          <div className="p-4 space-y-4">
            {/* Invoice Number */}
            <div className="flex justify-between items-center pb-3 border-b border-[#bc13fe]/20">
              <span className="text-xs text-[#e0d0ff]/60">Nomor Tagihan</span>
              <span className="font-mono font-bold text-sm text-[#00f7ff]">{invoice.invoiceNumber}</span>
            </div>

            {/* Customer Info */}
            <div>
              <p className="text-[10px] font-bold text-[#00f7ff] uppercase tracking-widest mb-2 flex items-center gap-2">
                <span className="w-6 h-[1px] bg-gradient-to-r from-[#00f7ff] to-transparent"></span>
                Informasi Pelanggan
              </p>
              <div className="bg-[#0a0520]/50 rounded-xl p-3 space-y-2.5">
                {/* Nama */}
                <div className="flex justify-between items-start text-xs gap-2">
                  <span className="text-[#e0d0ff]/60 flex items-center gap-1.5 shrink-0"><User className="w-3 h-3 text-[#bc13fe]" />Nama</span>
                  <span className="font-semibold text-white text-right">{invoice.user?.name || invoice.customerName}</span>
                </div>
                {/* Username */}
                {invoice.user?.username && (
                  <div className="flex justify-between items-start text-xs gap-2">
                    <span className="text-[#e0d0ff]/60 flex items-center gap-1.5 shrink-0"><Hash className="w-3 h-3 text-[#bc13fe]" />Username</span>
                    <span className="font-mono text-[#00f7ff] text-right">{invoice.user.username}</span>
                  </div>
                )}
                {/* Customer ID */}
                {invoice.user?.customerId && (
                  <div className="flex justify-between items-start text-xs gap-2">
                    <span className="text-[#e0d0ff]/60 flex items-center gap-1.5 shrink-0"><Hash className="w-3 h-3 text-[#00f7ff]" />ID Pelanggan</span>
                    <span className="font-mono text-white text-right">{invoice.user.customerId}</span>
                  </div>
                )}
                {/* Telepon */}
                <div className="flex justify-between items-start text-xs gap-2">
                  <span className="text-[#e0d0ff]/60 flex items-center gap-1.5 shrink-0"><Phone className="w-3 h-3 text-[#00f7ff]" />Telepon</span>
                  <span className="font-medium text-white text-right">{invoice.user?.phone || invoice.customerPhone}</span>
                </div>
                {/* Email */}
                {invoice.user?.email && (
                  <div className="flex justify-between items-start text-xs gap-2">
                    <span className="text-[#e0d0ff]/60 flex items-center gap-1.5 shrink-0"><Mail className="w-3 h-3 text-[#ff44cc]" />Email</span>
                    <span className="font-medium text-white text-right break-all">{invoice.user.email}</span>
                  </div>
                )}
                {/* Alamat */}
                {invoice.user?.address && (
                  <div className="flex justify-between items-start text-xs gap-2">
                    <span className="text-[#e0d0ff]/60 flex items-center gap-1.5 shrink-0"><MapPin className="w-3 h-3 text-[#ff44cc]" />Alamat</span>
                    <span className="font-medium text-white text-right max-w-[60%]">{invoice.user.address}</span>
                  </div>
                )}
                {/* Area */}
                {invoice.user?.area?.name && (
                  <div className="flex justify-between items-start text-xs gap-2">
                    <span className="text-[#e0d0ff]/60 flex items-center gap-1.5 shrink-0"><Network className="w-3 h-3 text-[#bc13fe]" />Area</span>
                    <span className="font-medium text-white text-right">{invoice.user.area.name}</span>
                  </div>
                )}

                {/* Divider */}
                <div className="border-t border-[#bc13fe]/15 pt-2 space-y-2.5">
                  {/* Paket */}
                  {invoice.user?.profile && (
                    <div className="flex justify-between items-start text-xs gap-2">
                      <span className="text-[#e0d0ff]/60 flex items-center gap-1.5 shrink-0"><Package className="w-3 h-3 text-[#ff44cc]" />Paket</span>
                      <div className="text-right">
                        <p className="font-semibold text-white">{invoice.user.profile.name}</p>
                        {(invoice.user.profile.downloadSpeed > 0) && (
                          <p className="text-[10px] text-[#00f7ff]/70 flex items-center justify-end gap-1"><Zap className="w-2.5 h-2.5" />{invoice.user.profile.downloadSpeed}M / {invoice.user.profile.uploadSpeed}M</p>
                        )}
                      </div>
                    </div>
                  )}
                  {/* Tipe & Status */}
                  <div className="flex justify-between items-center text-xs gap-2">
                    <span className="text-[#e0d0ff]/60 flex items-center gap-1.5 shrink-0"><CreditCard className="w-3 h-3 text-[#00f7ff]" />Tipe</span>
                    <div className="flex items-center gap-1.5">
                      <span className={`px-1.5 py-0.5 rounded text-[9px] font-bold ${
                        invoice.user?.subscriptionType === 'PREPAID'
                          ? 'bg-[#ff44cc]/20 text-[#ff44cc] border border-[#ff44cc]/30'
                          : 'bg-[#00f7ff]/15 text-[#00f7ff] border border-[#00f7ff]/30'
                      }`}>{invoice.user?.subscriptionType || 'POSTPAID'}</span>
                    </div>
                  </div>
                  {/* Router */}
                  {invoice.user?.router?.shortname && (
                    <div className="flex justify-between items-start text-xs gap-2">
                      <span className="text-[#e0d0ff]/60 flex items-center gap-1.5 shrink-0"><Router className="w-3 h-3 text-[#bc13fe]" />Router</span>
                      <span className="font-medium text-white text-right">{invoice.user.router.shortname}</span>
                    </div>
                  )}
                </div>
              </div>
            </div>

            {/* Amount */}
            <div className="bg-gradient-to-br from-[#bc13fe]/20 to-[#00f7ff]/20 rounded-xl p-5 text-center border border-[#bc13fe]/30">
              <p className="text-[10px] text-[#e0d0ff]/60 mb-1">Total Tagihan</p>
              <p className="text-3xl font-bold text-[#00f7ff] drop-shadow-[0_0_15px_rgba(0,247,255,0.5)]">{formatCurrency(invoice.amount)}</p>
            </div>

            {/* Dates */}
            <div className="grid grid-cols-2 gap-3">
              <div className="bg-[#0a0520]/50 rounded-xl p-3">
                <p className="text-[10px] text-[#e0d0ff]/60 mb-0.5 flex items-center gap-1"><Calendar className="w-3 h-3 text-[#bc13fe]" />Tanggal Terbit</p>
                <p className="text-xs font-medium text-white">{formatDate(invoice.createdAt)}</p>
              </div>
              <div className="bg-[#0a0520]/50 rounded-xl p-3">
                <p className="text-[10px] text-[#e0d0ff]/60 mb-0.5 flex items-center gap-1"><Calendar className="w-3 h-3 text-[#00f7ff]" />Jatuh Tempo</p>
                <p className="text-xs font-medium text-white">{formatDate(invoice.dueDate)}</p>
              </div>
            </div>

            {/* Overdue Warning */}
            {invoice.status === 'OVERDUE' && (
              <div className="bg-[#ff4466]/10 border border-[#ff4466]/30 rounded-xl p-3">
                <div className="flex items-start gap-2">
                  <AlertCircle className="w-4 h-4 text-[#ff6b8a] mt-0.5 flex-shrink-0" />
                  <div>
                    <p className="text-xs font-bold text-[#ff6b8a]">Pembayaran Terlambat</p>
                    <p className="text-[10px] text-[#ff6b8a]/80 mt-0.5">Segera lakukan pembayaran.</p>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Payment Methods */}
        <div className="bg-[#1a0f35]/80 backdrop-blur-xl rounded-2xl border-2 border-[#bc13fe]/30 overflow-hidden shadow-[0_0_30px_rgba(188,19,254,0.15)]">
          <div className="px-4 py-3 border-b border-[#bc13fe]/20">
            <h2 className="text-sm font-bold text-white flex items-center gap-2">
              <CreditCard className="w-4 h-4 text-[#00f7ff]" />
              Metode Pembayaran
            </h2>
          </div>
          <div className="p-4">
            {paymentGateways.length === 0 && !qrisOwn ? (
              <div className="text-center py-6">
                <Building2 className="w-10 h-10 text-[#e0d0ff]/40 mx-auto mb-2" />
                <p className="text-xs text-[#e0d0ff]/60">Tidak ada metode pembayaran tersedia.</p>
              </div>
            ) : (
              <div className="space-y-2">
                {qrisOwn && (
                  <button
                    onClick={() => handlePayment('qris_own')}
                    disabled={processing}
                    className="w-full flex items-center justify-between p-4 bg-gradient-to-r from-[#0d9488]/40 to-[#14b8a6]/40 border-2 border-[#14b8a6]/50 rounded-xl hover:border-[#2dd4bf] hover:shadow-[0_0_20px_rgba(45,212,191,0.2)] transition-all disabled:opacity-50"
                  >
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 bg-gradient-to-br from-[#0d9488] to-[#14b8a6] rounded-xl flex items-center justify-center shadow-[0_0_15px_rgba(20,184,166,0.3)]">
                        <QrCode className="w-5 h-5 text-white" />
                      </div>
                      <div className="text-left">
                        <p className="text-xs font-bold text-white">QRIS Langsung</p>
                        <p className="text-[10px] text-[#e0d0ff]/70">Tanpa biaya admin</p>
                      </div>
                    </div>
                    {processing ? (
                      <Loader2 className="w-4 h-4 animate-spin text-[#2dd4bf]" />
                    ) : (
                      <span className="text-[10px] text-[#2dd4bf] font-medium">Bayar Sekarang →</span>
                    )}
                  </button>
                )}
                {paymentGateways.map((gateway) => {
                  // For Duitku: show individual payment method options
                  if (gateway.provider === 'duitku') {
                    if (loadingDuitkuMethods) {
                      return (
                        <div key={gateway.id} className="flex items-center justify-center py-4">
                          <Loader2 className="w-5 h-5 animate-spin text-[#00f7ff] mr-2" />
                          <span className="text-xs text-[#e0d0ff]/60">Memuat metode Duitku...</span>
                        </div>
                      );
                    }
                    if (duitkuMethods.length > 0) {
                      return (
                        <div key={gateway.id} className="space-y-2">
                          <p className="text-[10px] font-bold text-[#00f7ff] uppercase tracking-widest px-1">{gateway.name}</p>
                          {duitkuMethods.map((method) => (
                            <button
                              key={method.code}
                              onClick={() => handlePayment('duitku', method.code)}
                              disabled={processing}
                              className="w-full flex items-center justify-between p-4 bg-[#0a0520]/50 border-2 border-[#bc13fe]/20 rounded-xl hover:border-[#00f7ff]/50 hover:bg-[#0a0520]/80 hover:shadow-[0_0_20px_rgba(0,247,255,0.1)] transition-all disabled:opacity-50"
                            >
                              <div className="flex items-center gap-3">
                                <div className="w-10 h-10 bg-gradient-to-br from-[#bc13fe] to-[#00f7ff] rounded-xl flex items-center justify-center shadow-[0_0_15px_rgba(188,19,254,0.3)]">
                                  <CreditCard className="w-5 h-5 text-white" />
                                </div>
                                <div className="text-left">
                                  <p className="text-xs font-bold text-white">{method.name}</p>
                                  <p className="text-[10px] text-[#e0d0ff]/60 uppercase">{method.code}</p>
                                </div>
                              </div>
                              {processing ? (
                                <Loader2 className="w-4 h-4 animate-spin text-[#00f7ff]" />
                              ) : (
                                <span className="text-[10px] text-[#00f7ff] font-medium">Bayar →</span>
                              )}
                            </button>
                          ))}
                        </div>
                      );
                    }
                    // Fallback: show single Duitku button with SP default
                  }

                  return (
                    <button
                      key={gateway.id}
                      onClick={() => handlePayment(gateway.provider)}
                      disabled={processing}
                      className="w-full flex items-center justify-between p-4 bg-[#0a0520]/50 border-2 border-[#bc13fe]/20 rounded-xl hover:border-[#00f7ff]/50 hover:bg-[#0a0520]/80 hover:shadow-[0_0_20px_rgba(0,247,255,0.1)] transition-all disabled:opacity-50"
                    >
                      <div className="flex items-center gap-3">
                        <div className="w-10 h-10 bg-gradient-to-br from-[#bc13fe] to-[#00f7ff] rounded-xl flex items-center justify-center shadow-[0_0_15px_rgba(188,19,254,0.3)]">
                          <CreditCard className="w-5 h-5 text-white" />
                        </div>
                        <div className="text-left">
                          <p className="text-xs font-bold text-white">{gateway.name}</p>
                          <p className="text-[10px] text-[#e0d0ff]/60 capitalize">{gateway.provider}</p>
                        </div>
                      </div>
                      {processing ? (
                        <Loader2 className="w-4 h-4 animate-spin text-[#00f7ff]" />
                      ) : (
                        <span className="text-[10px] text-[#00f7ff] font-medium">Bayar Sekarang →</span>
                      )}
                    </button>
                  );
                })}
              </div>
            )}
          </div>
        </div>

        {/* Company Info */}
        {company && (
          <div className="bg-[#1a0f35]/80 backdrop-blur-xl rounded-2xl border-2 border-[#bc13fe]/30 p-4 text-center shadow-[0_0_30px_rgba(188,19,254,0.1)]">
            <h3 className="text-sm font-bold text-white">{company.name}</h3>
            {company.address && <p className="text-[10px] text-[#e0d0ff]/60 mt-1">📍 {company.address}</p>}
            <div className="flex flex-wrap justify-center gap-3 text-[10px] text-[#e0d0ff]/60 mt-2">
              {company.phone && <span>📞 {company.phone}</span>}
              {company.email && <span>✉️ {company.email}</span>}
            </div>
          </div>
        )}

        {/* Footer */}
        <div className="text-center space-y-1">
          <p className="text-[10px] text-[#e0d0ff]/50">Pembayaran aman didukung oleh</p>
          <p className="text-xs font-bold bg-gradient-to-r from-[#00f7ff] to-[#bc13fe] bg-clip-text text-transparent">{company?.name || 'ISP Billing'}</p>
        </div>
      </div>

      {/* ========== QRIS OVERLAY ========== */}
      {qrisData && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4" style={{ background: 'rgba(10,5,32,0.95)', backdropFilter: 'blur(12px)' }}>
          <div className="max-w-sm w-full relative">
            {/* Close Button */}
            <button onClick={closeQris} className="absolute -top-2 -right-2 z-10 w-8 h-8 flex items-center justify-center bg-[#1a0f35] border-2 border-[#bc13fe]/40 rounded-full text-[#e0d0ff]/60 hover:text-white hover:border-[#ff44cc] transition-all">
              <X className="w-4 h-4" />
            </button>

            {/* SUCCESS STATE */}
            {qrisStatus === 'paid' && (
              <div className="bg-[#1a0f35]/90 backdrop-blur-xl rounded-2xl border-2 border-[#00ff88]/50 p-8 text-center shadow-[0_0_60px_rgba(0,255,136,0.3)]">
                <div className="w-20 h-20 bg-[#00ff88]/20 rounded-full flex items-center justify-center mx-auto mb-4 border-2 border-[#00ff88]/50 shadow-[0_0_40px_rgba(0,255,136,0.4)] animate-bounce">
                  <CheckCircle className="w-10 h-10 text-[#00ff88] drop-shadow-[0_0_15px_rgba(0,255,136,0.8)]" />
                </div>
                <h2 className="text-xl font-bold text-[#00ff88] mb-2">Pembayaran Berhasil!</h2>
                <p className="text-sm text-[#e0d0ff]/70">Terima kasih, pembayaran Anda telah dikonfirmasi.</p>
                <p className="text-xs text-[#e0d0ff]/50 mt-3">Memuat ulang halaman...</p>
              </div>
            )}

            {/* EXPIRED STATE */}
            {qrisStatus === 'expired' && (
              <div className="bg-[#1a0f35]/90 backdrop-blur-xl rounded-2xl border-2 border-[#ff4466]/50 p-8 text-center shadow-[0_0_40px_rgba(255,68,102,0.2)]">
                <AlertCircle className="w-12 h-12 text-[#ff6b8a] mx-auto mb-3 drop-shadow-[0_0_15px_rgba(255,68,102,0.5)]" />
                <h2 className="text-lg font-bold text-white mb-2">QRIS Kadaluarsa</h2>
                <p className="text-xs text-[#e0d0ff]/60 mb-4">Silakan buat pembayaran baru.</p>
                <button onClick={closeQris} className="px-6 py-2.5 bg-gradient-to-r from-[#bc13fe] to-[#00f7ff] text-white text-sm font-bold rounded-xl">
                  Tutup
                </button>
              </div>
            )}

            {/* FAILED STATE */}
            {qrisStatus === 'failed' && (
              <div className="bg-[#1a0f35]/90 backdrop-blur-xl rounded-2xl border-2 border-[#ff4466]/50 p-8 text-center shadow-[0_0_40px_rgba(255,68,102,0.2)]">
                <AlertCircle className="w-12 h-12 text-[#ff6b8a] mx-auto mb-3" />
                <h2 className="text-lg font-bold text-white mb-2">Pembayaran Gagal</h2>
                <p className="text-xs text-[#e0d0ff]/60 mb-4">Silakan coba lagi.</p>
                <button onClick={closeQris} className="px-6 py-2.5 bg-gradient-to-r from-[#bc13fe] to-[#00f7ff] text-white text-sm font-bold rounded-xl">
                  Tutup
                </button>
              </div>
            )}

            {/* PENDING STATE - Show QR */}
            {qrisStatus === 'pending' && (
              <div className="bg-[#1a0f35]/90 backdrop-blur-xl rounded-2xl border-2 border-[#bc13fe]/40 overflow-hidden shadow-[0_0_50px_rgba(188,19,254,0.2)]">
                {/* Header */}
                <div className="bg-gradient-to-r from-[#bc13fe] to-[#00f7ff] px-4 py-3 flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <QrCode className="w-5 h-5 text-white" />
                    <span className="text-sm font-bold text-white">Scan QRIS</span>
                  </div>
                  <div className="flex items-center gap-1.5 bg-white/20 px-2.5 py-1 rounded-full">
                    <Clock className="w-3 h-3 text-white" />
                    <span className="text-xs font-mono font-bold text-white">{formatCountdown(qrisCountdown)}</span>
                  </div>
                </div>

                <div className="p-5 space-y-4">
                  {/* Amount */}
                  <div className="text-center">
                    <p className="text-[10px] text-[#e0d0ff]/50 uppercase tracking-wider">Total Pembayaran</p>
                    <p className="text-2xl font-bold text-[#00f7ff] drop-shadow-[0_0_10px_rgba(0,247,255,0.4)]">{formatCurrency(invoice?.amount || 0)}</p>
                  </div>

                  {/* QR Code */}
                  <div className="flex justify-center">
                    <div className="bg-white p-4 rounded-2xl shadow-[0_0_30px_rgba(188,19,254,0.3)]">
                      <QRCodeSVG
                        value={qrisData.qrString}
                        size={220}
                        level="M"
                        includeMargin={false}
                        bgColor="#ffffff"
                        fgColor="#1a0f35"
                      />
                    </div>
                  </div>

                  {/* Instructions */}
                  <div className="bg-[#0a0520]/60 rounded-xl p-3 space-y-2">
                    <p className="text-[10px] font-bold text-[#00f7ff] uppercase tracking-widest">Cara Bayar:</p>
                    <div className="space-y-1.5">
                      {['Buka aplikasi e-wallet (GoPay, OVO, DANA, ShopeePay, dll)', 'Pilih menu "Scan QR" atau "Bayar"', 'Scan kode QR di atas', 'Konfirmasi dan selesaikan pembayaran'].map((step, i) => (
                        <div key={i} className="flex items-start gap-2">
                          <span className="w-4 h-4 bg-[#bc13fe]/30 rounded-full flex items-center justify-center text-[8px] font-bold text-[#e0d0ff] flex-shrink-0 mt-0.5">{i + 1}</span>
                          <p className="text-[11px] text-[#e0d0ff]/70">{step}</p>
                        </div>
                      ))}
                    </div>
                  </div>

                  {qrisData.isQrisOwn ? (
                    <>
                      {/* Unique amount warning — KRITIS untuk matching otomatis */}
                      {qrisData.uniqueAmount && qrisData.uniqueAmount !== invoice?.amount && (
                        <div className="bg-[#ff4466]/10 border border-[#ff4466]/40 rounded-xl p-3">
                          <p className="text-[11px] font-bold text-[#ff4466] mb-1">⚠️ Transfer TEPAT nominal berikut:</p>
                          <p className="text-lg font-bold text-white text-center py-1">{formatCurrency(qrisData.uniqueAmount)}</p>
                          <p className="text-[10px] text-[#e0d0ff]/70 leading-relaxed">
                            Nominal ini berbeda tipis dari tagihan Anda agar sistem dapat mencocokkan pembayaran secara otomatis. <strong>Jangan dibulatkan.</strong>
                          </p>
                        </div>
                      )}

                      {qrisData.hasListener ? (
                        /* Android listener aktif — polling otomatis */
                        <div className="bg-[#00ff88]/10 border border-[#00ff88]/30 rounded-xl p-3">
                          <p className="text-[11px] font-bold text-[#00ff88] mb-1 flex items-center gap-1.5">
                            <RefreshCw className="w-3.5 h-3.5 animate-spin" /> Menunggu konfirmasi otomatis...
                          </p>
                          <p className="text-[10px] text-[#00ff88]/80 leading-relaxed">
                            Setelah Anda transfer, sistem akan mendeteksi pembayaran secara otomatis dalam beberapa detik. Halaman ini akan otomatis diperbarui.
                          </p>
                        </div>
                      ) : (
                        /* Tidak ada listener — konfirmasi manual */
                        <div className="bg-[#00ff88]/10 border border-[#00ff88]/30 rounded-xl p-3">
                          <p className="text-[11px] font-bold text-[#00ff88] mb-1 flex items-center gap-1.5">
                            <CheckCircle className="w-3.5 h-3.5" /> Setelah bayar:
                          </p>
                          <p className="text-[10px] text-[#00ff88]/80 leading-relaxed">
                            Hubungi admin via WhatsApp untuk konfirmasi pembayaran. Layanan akan aktif setelah admin memproses tagihan Anda.
                          </p>
                        </div>
                      )}
                      {!qrisData.hasListener && company?.phone && (
                        <a
                          href={`https://wa.me/${company.phone.replace(/\D/g, '')}?text=${encodeURIComponent(`Halo Admin, saya sudah bayar tagihan internet.\n\nUsername: ${invoice?.user?.username || '-'}\nNomor Invoice: ${invoice?.invoiceNumber || '-'}\nNominal: ${formatCurrency(qrisData.uniqueAmount || invoice?.amount || 0)}\n\nMohon konfirmasi pembayarannya. Terima kasih.`)}`}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="flex items-center justify-center gap-2 w-full py-3 bg-[#25D366] text-white text-xs font-bold rounded-xl hover:bg-[#128C7E] transition-all shadow-[0_0_20px_rgba(37,211,102,0.3)]"
                        >
                          <Phone className="w-4 h-4" /> Konfirmasi via WhatsApp
                        </a>
                      )}
                    </>
                  ) : (
                    <>
                      {/* Polling indicator */}
                      <div className="flex items-center justify-center gap-2">
                        <RefreshCw className="w-3 h-3 text-[#00f7ff] animate-spin" />
                        <p className="text-[10px] text-[#e0d0ff]/50">Menunggu pembayaran... (auto-check setiap 5 detik)</p>
                      </div>

                      {/* Fallback link */}
                      {qrisData.paymentUrl && (
                        <a
                          href={qrisData.paymentUrl}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="flex items-center justify-center gap-2 w-full py-2.5 bg-[#0a0520]/50 border border-[#bc13fe]/30 rounded-xl text-[11px] text-[#e0d0ff]/60 hover:text-white hover:border-[#00f7ff]/50 transition-all"
                        >
                          <ExternalLink className="w-3 h-3" /> Buka halaman pembayaran alternatif
                        </a>
                      )}
                    </>
                  )}
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
