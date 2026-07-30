'use client';

import { useEffect, useState, Suspense } from 'react';
import { useSearchParams, useRouter } from 'next/navigation';
import { XCircle, RefreshCw, Home, Loader2, AlertTriangle, MessageCircle } from 'lucide-react';
import { useTranslation } from '@/hooks/useTranslation';

function PaymentFailedContent() {
  const { t } = useTranslation();
  const searchParams = useSearchParams();
  const router = useRouter();
  const token = searchParams.get('token');
  const orderId = searchParams.get('order_id');
  const reason = searchParams.get('reason');
  const [loading, setLoading] = useState(true);
  const [invoiceNumber, setInvoiceNumber] = useState<string>('');
  const [companyPhone, setCompanyPhone] = useState('6281234567890');

  useEffect(() => {
    if (token || orderId) fetchInvoiceInfo(); else setLoading(false);
    fetch('/api/company').then(res => res.json()).then(data => { if (data.phone) { let phone = data.phone.replace(/^0/, '62'); if (!phone.startsWith('62')) phone = '62' + phone; setCompanyPhone(phone); } }).catch(() => { });
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token, orderId]);

  const fetchInvoiceInfo = async () => {
    try {
      const endpoint = token
        ? `/api/invoices/check?token=${encodeURIComponent(token)}`
        : `/api/payment/check-order?orderId=${encodeURIComponent(orderId || '')}`;

      const res = await fetch(endpoint);
      const data = await res.json();
      if (res.ok && data.invoice) setInvoiceNumber(data.invoice.invoiceNumber);
    } catch { } finally { setLoading(false); }
  };

  if (loading) return (
    <div className="min-h-screen bg-secondary relative overflow-hidden flex items-center justify-center p-4">
      
      <div className="text-center relative z-10">
        <Loader2 className="w-10 h-10 animate-spin mx-auto text-[#ff6b8a] drop-shadow-[0_0_20px_rgba(255,68,102,0.6)] mb-3" />
        <p className="text-xs text-muted-foreground/70">{t('payment.loadingInfo')}</p>
      </div>
    </div>
  );

  return (
    <div className="min-h-screen bg-secondary relative py-6 px-4">
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        <div className="absolute inset-0 bg-[linear-gradient(rgba(122, 90, 248,0.02)_1px,transparent_1px),linear-gradient(90deg,rgba(122, 90, 248,0.02)_1px,transparent_1px)] bg-[size:50px_50px]"></div>
      </div>

      <div className="max-w-md mx-auto space-y-4 relative z-10">
        <div className="text-center">
          <div className="inline-block relative mb-4">
            <div className="absolute inset-0 bg-[#ff4466]/30 rounded-full animate-pulse"></div>
            <div className="relative w-20 h-20 bg-gradient-to-br from-[#ff4466] to-accent-foreground rounded-full flex items-center justify-center mx-auto shadow-[0_0_40px_rgba(255,68,102,0.5)]">
              <XCircle className="w-10 h-10 text-white drop-shadow-[0_0_10px_rgba(255,255,255,0.8)]" />
            </div>
          </div>
          <h1 className="text-xl font-bold text-[#ff6b8a] mb-1">{t('payment.paymentFailed')}</h1>
          <p className="text-xs text-muted-foreground/70">{t('payment.transactionNotCompleted')}</p>
        </div>

        <div className="bg-secondary/80 backdrop-blur-xl rounded-xl border-2 border-[#ff4466]/50 overflow-hidden shadow-[0_0_40px_rgba(255,68,102,0.2)]">
          <div className="p-5 space-y-4">
            <div className="flex justify-center">
              <div className="w-14 h-14 bg-accent/20 rounded-full flex items-center justify-center border-2 border-accent/30">
                <AlertTriangle className="w-7 h-7 text-accent-foreground drop-shadow-[0_0_10px_rgba(238, 70, 188,0.5)]" />
              </div>
            </div>

            {invoiceNumber && (
              <div className="text-center">
                <p className="text-[10px] text-muted-foreground/60 mb-0.5">Invoice</p>
                <p className="text-sm font-bold text-white">#{invoiceNumber}</p>
              </div>
            )}

            {reason && (
              <div className="bg-[#ff4466]/10 border border-[#ff4466]/30 rounded-xl p-3">
                <p className="text-xs font-medium text-[#ff6b8a] text-center">{decodeURIComponent(reason)}</p>
              </div>
            )}

            <div className="bg-input/50 rounded-xl p-4">
              <p className="text-xs font-bold text-muted-foreground mb-2">{t('payment.possibleCauses')}</p>
              <ul className="text-[10px] text-muted-foreground/70 space-y-1.5 list-disc list-inside">
                <li>{t('payment.cause1')}</li>
                <li>{t('payment.cause2')}</li>
                <li>{t('payment.cause3')}</li>
                <li>{t('payment.cause4')}</li>
                <li>{t('payment.cause5')}</li>
              </ul>
            </div>

            <div className="text-center">
              <p className="text-[10px] text-muted-foreground/60">{t('payment.noCharge')}</p>
              <p className="text-[10px] text-muted-foreground/60">{t('payment.tryAgainOrContact')}</p>
            </div>
          </div>
        </div>

        <div className="flex gap-3">
          <button onClick={() => router.push('/')} className="flex-1 flex items-center justify-center gap-2 px-4 py-3 text-xs font-medium bg-input text-white rounded-xl border-2 border-brand-600/30 hover:border-[#ff4466]">
            <Home className="w-4 h-4 text-[#ff6b8a]" />{t('common.back')}
          </button>
          {token && (
            <button onClick={() => router.push(`/pay/${token}`)} className="flex-1 flex items-center justify-center gap-2 px-4 py-3 text-xs font-bold text-white bg-gradient-to-r from-[#ff4466] to-accent-foreground rounded-xl shadow-[0_0_20px_rgba(255,68,102,0.3)]">
              <RefreshCw className="w-4 h-4" />{t('payment.tryAgain')}
            </button>
          )}
        </div>

        <div className="text-center space-y-2">
          <p className="text-[10px] text-muted-foreground/60">{t('payment.needHelp')}</p>
          <a href={`https://wa.me/${companyPhone}`} target="_blank" rel="noopener noreferrer" className="inline-flex items-center gap-1.5 text-sm text-[success] hover:text-brand-400 font-medium transition-colors">
            <MessageCircle className="w-4 h-4" />
            {t('payment.contactCS')}
          </a>
        </div>
      </div>
    </div>
  );
}

export default function PaymentFailedPage() {
  return (
    <Suspense fallback={
      <div className="min-h-screen bg-secondary flex items-center justify-center">
        <Loader2 className="w-10 h-10 animate-spin text-[#ff6b8a] drop-shadow-[0_0_20px_rgba(255,68,102,0.6)]" />
      </div>
    }>
      <PaymentFailedContent />
    </Suspense>
  );
}
