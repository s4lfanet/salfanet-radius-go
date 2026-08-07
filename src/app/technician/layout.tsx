import type { Metadata, Viewport } from 'next';

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

export async function generateMetadata(): Promise<Metadata> {
  const name = await getCompanyName();
  return {
    title: `Portal Teknisi - ${name}`,
    description: 'Portal Teknisi untuk manajemen tiket dan pelanggan',
    manifest: '/manifest-technician.json',
  };
}

export const viewport: Viewport = {
  width: 'device-width',
  initialScale: 1,
  themeColor: '#f59e0b',
  // maximumScale and userScalable intentionally NOT set -- allow pinch-zoom
};

export default function TechnicianRootLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}
