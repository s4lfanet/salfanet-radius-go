import { NextRequest, NextResponse } from 'next/server';
import { prisma } from '@/server/db/client';
import { getServerSession } from 'next-auth';
import { authOptions } from '@/server/auth/config';

type Params = Promise<{ id: string }>;

// GET /api/payroll-templates/[id]
export async function GET(_req: NextRequest, { params }: { params: Params }) {
  try {
    const session = await getServerSession(authOptions);
    if (!session) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

    const { id } = await params;
    const [template] = await prisma.$queryRaw<any[]>`SELECT * FROM payroll_templates WHERE id = ${id}`;
    if (!template) return NextResponse.json({ error: 'Not found' }, { status: 404 });
    return NextResponse.json({ success: true, template });
  } catch (error) {
    console.error('[PAYROLL_TEMPLATE_GET]', error);
    return NextResponse.json({ error: 'Failed to fetch template' }, { status: 500 });
  }
}

// PUT /api/payroll-templates/[id]
export async function PUT(request: NextRequest, { params }: { params: Params }) {
  try {
    const session = await getServerSession(authOptions);
    if (!session) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

    const { id } = await params;
    const body = await request.json();
    const { name, baseWage, allowance, deduction, notes } = body;
    const now = new Date();

    await prisma.$executeRaw`
      UPDATE payroll_templates
      SET name = COALESCE(${name ?? null}, name),
          baseWage = COALESCE(${baseWage ?? null}, baseWage),
          allowance = COALESCE(${allowance ?? null}, allowance),
          deduction = COALESCE(${deduction ?? null}, deduction),
          notes = ${notes ?? null},
          updatedAt = ${now}
      WHERE id = ${id}
    `;
    const [template] = await prisma.$queryRaw<any[]>`SELECT * FROM payroll_templates WHERE id = ${id}`;
    return NextResponse.json({ success: true, template });
  } catch (error) {
    console.error('[PAYROLL_TEMPLATE_UPDATE]', error);
    return NextResponse.json({ error: 'Failed to update template' }, { status: 500 });
  }
}

// DELETE /api/payroll-templates/[id]
export async function DELETE(_req: NextRequest, { params }: { params: Params }) {
  try {
    const session = await getServerSession(authOptions);
    if (!session) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

    const { id } = await params;
    await prisma.$executeRaw`DELETE FROM payroll_templates WHERE id = ${id}`;
    return NextResponse.json({ success: true, message: 'Template deleted' });
  } catch (error) {
    console.error('[PAYROLL_TEMPLATE_DELETE]', error);
    return NextResponse.json({ error: 'Failed to delete template' }, { status: 500 });
  }
}
