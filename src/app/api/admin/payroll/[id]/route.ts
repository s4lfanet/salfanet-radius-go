import { NextRequest, NextResponse } from 'next/server';
import { prisma } from '@/server/db/client';
import { getServerSession } from 'next-auth';
import { authOptions } from '@/server/auth/config';

// GET /api/admin/payroll/:id
export async function GET(req: NextRequest, { params }: { params: Promise<{ id: string }> }) {
  const session = await getServerSession(authOptions);
  if (!session) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  const { id } = await params;
  try {
    const rows: any[] = await prisma.$queryRawUnsafe(
      'SELECT * FROM payroll_records WHERE id = ? LIMIT 1', id
    );
    if (!rows.length) return NextResponse.json({ error: 'Not found' }, { status: 404 });
    return NextResponse.json({ success: true, payroll: rows[0] });
  } catch (error: any) {
    return NextResponse.json({ error: error.message }, { status: 500 });
  }
}

// PUT /api/admin/payroll/:id
export async function PUT(req: NextRequest, { params }: { params: Promise<{ id: string }> }) {
  const session = await getServerSession(authOptions);
  if (!session) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  const { id } = await params;
  try {
    const body = await req.json();
    const now = new Date().toISOString().slice(0, 19).replace('T', ' ');
    const fields = Object.keys(body).filter(k => k !== 'id' && k !== 'createdAt');
    if (!fields.length) return NextResponse.json({ error: 'No fields to update' }, { status: 400 });
    const set = fields.map(f => `\`${f}\` = ?`).join(', ');
    const vals = fields.map(f => body[f]);
    await prisma.$executeRawUnsafe(
      `UPDATE payroll_records SET ${set}, updatedAt = ? WHERE id = ?`,
      ...vals, now, id
    );
    const rows: any[] = await prisma.$queryRawUnsafe('SELECT * FROM payroll_records WHERE id = ? LIMIT 1', id);
    return NextResponse.json({ success: true, payroll: rows[0] });
  } catch (error: any) {
    return NextResponse.json({ error: error.message }, { status: 500 });
  }
}

// DELETE /api/admin/payroll/:id
export async function DELETE(req: NextRequest, { params }: { params: Promise<{ id: string }> }) {
  const session = await getServerSession(authOptions);
  if (!session) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  const { id } = await params;
  try {
    await prisma.$executeRawUnsafe('DELETE FROM payroll_records WHERE id = ?', id);
    return NextResponse.json({ success: true, message: 'Payroll record deleted' });
  } catch (error: any) {
    return NextResponse.json({ error: error.message }, { status: 500 });
  }
}
