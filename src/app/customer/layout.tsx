import type { Metadata } from 'next';
import CustomerClientLayout from './CustomerClientLayout';

async function getCompanyName(): Promise<string> {
  try {
    const res = await fetch(`${process.env.GO_API_URL || 'http://127.0.0.1:8080'}/api/public/company`, {
      next: { revalidate: 3600 },
    });
    if (res.ok) {
      const data = await res.json();
      return data.company?.name || 'SALFANET RADIUS';
    }
  } catch {}
  return 'SALFANET RADIUS';
}

export const dynamic = 'force-dynamic';

export async function generateMetadata(): Promise<Metadata> {
  const name = await getCompanyName();
  return {
    title: `Portal Pelanggan - ${name}`,
    description: `Cek tagihan, bayar online, dan pantau status langganan internet Anda di ${name}. Portal pelanggan ISP mudah dan cepat.`,
    manifest: '/manifest-customer.json',
    robots: { index: true, follow: true },
    openGraph: {
      title: `Portal Pelanggan - ${name}`,
      description: `Cek tagihan & bayar online langganan internet Anda di ${name}.`,
      type: 'website',
    },
  };
}

export default function CustomerLayout({ children }: { children: React.ReactNode }) {
  return <CustomerClientLayout>{children}</CustomerClientLayout>;
}

