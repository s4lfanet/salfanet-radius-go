import { NextRequest, NextResponse } from 'next/server';
import { prisma } from '@/server/db/client';
import { getServerSession } from 'next-auth';
import { authOptions } from '@/server/auth/config';
import { randomUUID } from 'crypto';

export const dynamic = 'force-dynamic';

export async function GET() {
  try {
    const session = await getServerSession(authOptions);
    if (!session?.user) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
    }

    const checklists = await prisma.$queryRaw<any[]>`
      SELECT id, title, description, category, steps, isActive, createdAt
      FROM troubleshooting_checklists
      WHERE isActive = 1
      ORDER BY title ASC
    `;

    return NextResponse.json({ success: true, checklists });
  } catch {
    return NextResponse.json({ error: 'Failed to fetch checklists' }, { status: 500 });
  }
}

export async function POST(request: NextRequest) {
  try {
    const session = await getServerSession(authOptions);
    if (!session?.user) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
    }

    const body = await request.json();
    const { title, description, category, steps } = body;

    if (!title?.trim()) {
      return NextResponse.json({ error: 'Title is required' }, { status: 400 });
    }

    const id = randomUUID();
    const now = new Date();

    await prisma.$executeRaw`
      INSERT INTO troubleshooting_checklists (id, title, description, category, steps, isActive, createdAt)
      VALUES (${id}, ${title}, ${description || null}, ${category || 'OTHER'}, ${steps || ''}, 1, ${now})
    `;

    return NextResponse.json({
      success: true,
      checklist: { id, title, description, category, steps, isActive: true, createdAt: now },
    }, { status: 201 });
  } catch {
    return NextResponse.json({ error: 'Failed to create checklist' }, { status: 500 });
  }
}
