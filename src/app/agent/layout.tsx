import type { Metadata, Viewport } from 'next';
import AgentLayoutClient from './AgentLayoutClient';

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
    title: `Agent Portal - ${name}`,
    description: 'Portal Agent untuk Generate Voucher',
    manifest: '/manifest-agent.json',
  };
}

export const viewport: Viewport = {
  width: 'device-width',
  initialScale: 1,
  themeColor: '#465fff',
  // maximumScale and userScalable intentionally NOT set -- allow pinch-zoom
};

export default function AgentLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <AgentLayoutClient>{children}</AgentLayoutClient>;
}
