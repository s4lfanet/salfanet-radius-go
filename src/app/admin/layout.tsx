import type { Metadata } from 'next';
import AdminClientLayout from './AdminClientLayout';

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
    title: `Admin Panel - ${name}`,
    manifest: '/manifest-admin.json',
  };
}

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  return <AdminClientLayout>{children}</AdminClientLayout>;
}
