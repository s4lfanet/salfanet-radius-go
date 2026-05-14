import { NextRequest, NextResponse } from 'next/server';
import { prisma } from '@/server/db/client';
import { getServerSession } from 'next-auth';
import { authOptions } from '@/server/auth/config';

// GET /api/admin/payroll
export async function GET(req: NextRequest) {
  const session = await getServerSession(authOptions);
  if (!session) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  const { searchParams } = new URL(req.url);
  const month = searchParams.get('month');
  const status = searchParams.get('status');
  const page = Math.max(1, parseInt(searchParams.get('page') || '1'));
  const limit = Math.min(200, Math.max(1, parseInt(searchParams.get('limit') || '20')));
  const offset = (page - 1) * limit;

  try {
    const conditions: string[] = [];
    const params: any[] = [];

    if (month) {
      conditions.push('`month` = ?');
      params.push(month);
    }
    if (status) {
      conditions.push('`status` = ?');
      params.push(status);
    }

    const where = conditions.length ? `WHERE ${conditions.join(' AND ')}` : '';
    const countResult: any[] = await prisma.$queryRawUnsafe(
      `SELECT COUNT(*) as total FROM payroll_records ${where}`,
      ...params
    );
    const total = Number(countResult[0]?.total ?? 0);

    const payroll: any[] = await prisma.$queryRawUnsafe(
      `SELECT * FROM payroll_records ${where} ORDER BY createdAt DESC LIMIT ? OFFSET ?`,
      ...params, limit, offset
    );

    return NextResponse.json({
      success: true,
      payroll,
      pagination: { page, limit, total },
    });
  } catch (error: any) {
    console.error('[ADMIN_PAYROLL_LIST]', error);
    return NextResponse.json({ success: true, payroll: [], pagination: { page, limit, total: 0 } });
  }
}

// POST /api/admin/payroll — create payroll record
export async function POST(req: NextRequest) {
  const session = await getServerSession(authOptions);
  if (!session) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  try {
    const body = await req.json();
    const id = crypto.randomUUID();
    const now = new Date().toISOString().slice(0, 19).replace('T', ' ');
    const net = (body.baseWage ?? 0) + (body.allowance ?? 0) + (body.bonus ?? 0) + (body.overtime ?? 0) - (body.deduction ?? 0);
    await prisma.$executeRawUnsafe(
      `INSERT INTO payroll_records (id, employeeId, month, baseWage, allowance, deduction, bonus, overtime, netAmount, status, notes, createdAt, updatedAt)
       VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
      id, body.employeeId, body.month, body.baseWage ?? 0,
      body.allowance ?? 0, body.deduction ?? 0, body.bonus ?? 0, body.overtime ?? 0,
      net, 'DRAFT', body.notes ?? null, now, now
    );
    return NextResponse.json({ success: true, payroll: { id, ...body, netAmount: net, status: 'DRAFT' } }, { status: 201 });
  } catch (error: any) {
    console.error('[ADMIN_PAYROLL_CREATE]', error);
    return NextResponse.json({ error: error.message }, { status: 500 });
  }
}
