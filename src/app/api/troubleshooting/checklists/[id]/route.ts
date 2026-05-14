import { NextRequest, NextResponse } from 'next/server';
import { prisma } from '@/server/db/client';
import { getServerSession } from 'next-auth';
import { authOptions } from '@/server/auth/config';

export const dynamic = 'force-dynamic';

export async function PUT(request: NextRequest, { params }: { params: Promise<{ id: string }> }) {
  try {
    const session = await getServerSession(authOptions);
    if (!session?.user) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
    }

    const { id } = await params;
    const body = await request.json();
    const { title, description, category, steps } = body;

    await prisma.$executeRaw`
      UPDATE troubleshooting_checklists
      SET title = ${title}, description = ${description || null},
          category = ${category || 'OTHER'}, steps = ${steps || ''}
      WHERE id = ${id}
    `;

    return NextResponse.json({ success: true });
  } catch {
    return NextResponse.json({ error: 'Failed to update checklist' }, { status: 500 });
  }
}

export async function DELETE(_request: NextRequest, { params }: { params: Promise<{ id: string }> }) {
  try {
    const session = await getServerSession(authOptions);
    if (!session?.user) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
    }

    const { id } = await params;

    await prisma.$executeRaw`
      UPDATE troubleshooting_checklists SET isActive = 0 WHERE id = ${id}
    `;

    return NextResponse.json({ success: true });
  } catch {
    return NextResponse.json({ error: 'Failed to delete checklist' }, { status: 500 });
  }
}
