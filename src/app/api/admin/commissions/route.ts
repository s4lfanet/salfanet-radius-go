import { NextRequest, NextResponse } from 'next/server';
import { getServerSession } from 'next-auth';
import { authOptions } from '@/lib/auth';
import { prisma } from '@/lib/prisma';

// GET /api/admin/commissions
export async function GET(req: NextRequest) {
  const session = await getServerSession(authOptions);
  if (!session) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  const { searchParams } = new URL(req.url);
  const status = searchParams.get('status');
  const page = Math.max(1, parseInt(searchParams.get('page') || '1'));
  const limit = Math.min(200, Math.max(1, parseInt(searchParams.get('limit') || '20')));
  const offset = (page - 1) * limit;

  try {
    const conditions: string[] = [];
    const params: any[] = [];

    if (status) {
      conditions.push('`status` = ?');
      params.push(status);
    }

    const where = conditions.length ? `WHERE ${conditions.join(' AND ')}` : '';
    const countResult: any[] = await prisma.$queryRawUnsafe(
      `SELECT COUNT(*) as total FROM commissions ${where}`,
      ...params
    );
    const total = Number(countResult[0]?.total ?? 0);

    const commissions: any[] = await prisma.$queryRawUnsafe(
      `SELECT * FROM commissions ${where} ORDER BY createdAt DESC LIMIT ? OFFSET ?`,
      ...params, limit, offset
    );

    return NextResponse.json({
      success: true,
      commissions,
      pagination: { page, limit, total },
    });
  } catch (error: any) {
    console.error('[ADMIN_COMMISSIONS_LIST]', error);
    return NextResponse.json({ success: true, commissions: [], pagination: { page, limit, total: 0 } });
  }
}

// POST /api/admin/commissions
export async function POST(req: NextRequest) {
  const session = await getServerSession(authOptions);
  if (!session) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  try {
    const body = await req.json();
    const id = crypto.randomUUID();
    const now = new Date().toISOString().slice(0, 19).replace('T', ' ');
    await prisma.$executeRawUnsafe(
      `INSERT INTO commissions (id, employeeId, type, amount, description, status, createdAt, updatedAt)
       VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
      id, body.employeeId, body.type ?? 'SALES', body.amount ?? 0,
      body.description ?? '', 'PENDING', now, now
    );
    return NextResponse.json({ success: true, commission: { id, ...body, status: 'PENDING' } }, { status: 201 });
  } catch (error: any) {
    console.error('[ADMIN_COMMISSIONS_CREATE]', error);
    return NextResponse.json({ error: error.message }, { status: 500 });
  }
}
