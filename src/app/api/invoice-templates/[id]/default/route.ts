import { NextRequest, NextResponse } from 'next/server';
import { prisma } from '@/server/db/client';
import { getServerSession } from 'next-auth';
import { authOptions } from '@/server/auth/config';

export const dynamic = 'force-dynamic';

export async function POST(_request: NextRequest, { params }: { params: Promise<{ id: string }> }) {
  try {
    const session = await getServerSession(authOptions);
    if (!session?.user) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
    }

    const { id } = await params;

    // Unset all defaults, then set this one
    await prisma.$executeRaw`UPDATE invoice_templates SET isDefault = 0`;
    await prisma.$executeRaw`UPDATE invoice_templates SET isDefault = 1 WHERE id = ${id}`;

    return NextResponse.json({ success: true, message: 'Default template updated' });
  } catch {
    return NextResponse.json({ error: 'Failed to set default template' }, { status: 500 });
  }
}
