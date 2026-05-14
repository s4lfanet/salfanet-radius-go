import { NextRequest, NextResponse } from 'next/server';
import { prisma } from '@/server/db/client';
import { getServerSession } from 'next-auth';
import { authOptions } from '@/server/auth/config';

type Params = Promise<{ id: string }>;

// POST /api/payroll-templates/[id]/default
export async function POST(_req: NextRequest, { params }: { params: Params }) {
  try {
    const session = await getServerSession(authOptions);
    if (!session) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

    const { id } = await params;
    await prisma.$executeRaw`UPDATE payroll_templates SET isDefault = FALSE`;
    await prisma.$executeRaw`UPDATE payroll_templates SET isDefault = TRUE WHERE id = ${id}`;
    return NextResponse.json({ success: true, message: 'Default payroll template updated' });
  } catch (error) {
    console.error('[PAYROLL_TEMPLATE_SET_DEFAULT]', error);
    return NextResponse.json({ error: 'Failed to set default' }, { status: 500 });
  }
}
