import { NextRequest, NextResponse } from 'next/server';
import { prisma } from '@/server/db/client';
import { getServerSession } from 'next-auth';
import { authOptions } from '@/server/auth/config';

// GET /api/payroll-templates
export async function GET() {
  try {
    const session = await getServerSession(authOptions);
    if (!session) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

    const templates = await prisma.$queryRaw`
      SELECT id, name, baseWage, allowance, deduction, notes, isDefault, createdAt, updatedAt
      FROM payroll_templates
      ORDER BY createdAt DESC
    `;
    return NextResponse.json({ success: true, templates });
  } catch (error) {
    console.error('[PAYROLL_TEMPLATES_LIST]', error);
    return NextResponse.json({ success: true, templates: [] });
  }
}

// POST /api/payroll-templates
export async function POST(request: NextRequest) {
  try {
    const session = await getServerSession(authOptions);
    if (!session) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

    const body = await request.json();
    const { name, baseWage = 0, allowance = 0, deduction = 0, notes } = body;
    if (!name) return NextResponse.json({ error: 'name is required' }, { status: 400 });

    const id = crypto.randomUUID();
    const now = new Date();
    await prisma.$executeRaw`
      INSERT INTO payroll_templates (id, name, baseWage, allowance, deduction, notes, isDefault, createdAt, updatedAt)
      VALUES (${id}, ${name}, ${baseWage}, ${allowance}, ${deduction}, ${notes ?? null}, FALSE, ${now}, ${now})
    `;
    const [template] = await prisma.$queryRaw<any[]>`SELECT * FROM payroll_templates WHERE id = ${id}`;
    return NextResponse.json({ success: true, template }, { status: 201 });
  } catch (error) {
    console.error('[PAYROLL_TEMPLATES_CREATE]', error);
    return NextResponse.json({ error: 'Failed to create template' }, { status: 500 });
  }
}
