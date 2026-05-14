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

    const templates = await prisma.$queryRaw<any[]>`
      SELECT id, name, subject, htmlBody, isDefault, templateType, createdAt, updatedAt
      FROM invoice_templates
      ORDER BY createdAt DESC
    `;

    return NextResponse.json({ success: true, templates });
  } catch {
    return NextResponse.json({ error: 'Failed to fetch templates' }, { status: 500 });
  }
}

export async function POST(request: NextRequest) {
  try {
    const session = await getServerSession(authOptions);
    if (!session?.user) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
    }

    const body = await request.json();
    const { name, subject, htmlBody, templateType } = body;

    if (!name?.trim()) {
      return NextResponse.json({ error: 'Name is required' }, { status: 400 });
    }

    const id = randomUUID();
    const now = new Date();
    const type = templateType || 'INVOICE';

    await prisma.$executeRaw`
      INSERT INTO invoice_templates (id, name, subject, htmlBody, isDefault, templateType, createdAt, updatedAt)
      VALUES (${id}, ${name}, ${subject || ''}, ${htmlBody || ''}, 0, ${type}, ${now}, ${now})
    `;

    return NextResponse.json({
      success: true,
      template: { id, name, subject, htmlBody, isDefault: false, templateType: type, createdAt: now, updatedAt: now },
    }, { status: 201 });
  } catch {
    return NextResponse.json({ error: 'Failed to create template' }, { status: 500 });
  }
}
