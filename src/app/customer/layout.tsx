import type { Metadata } from 'next';
import CustomerClientLayout from './CustomerClientLayout';

async function getCompanyName(): Promise<string> {
  try {
    const res = await fetch('http://127.0.0.1:8080/api/public/company', {
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
    title: `Customer Portal - ${name}`,
    manifest: '/manifest-customer.json',
  };
}

export default function CustomerLayout({ children }: { children: React.ReactNode }) {
  return <CustomerClientLayout>{children}</CustomerClientLayout>;
}

