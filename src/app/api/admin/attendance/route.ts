import { NextRequest, NextResponse } from 'next/server';
import { prisma } from '@/server/db/client';
import { getServerSession } from 'next-auth';
import { authOptions } from '@/server/auth/config';

// GET /api/admin/attendance
export async function GET(req: NextRequest) {
  const session = await getServerSession(authOptions);
  if (!session) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  const { searchParams } = new URL(req.url);
  const date = searchParams.get('date');
  const employeeId = searchParams.get('employeeId');
  const page = Math.max(1, parseInt(searchParams.get('page') || '1'));
  const limit = Math.min(200, Math.max(1, parseInt(searchParams.get('limit') || '20')));
  const offset = (page - 1) * limit;

  try {
    const conditions: string[] = [];
    const params: any[] = [];

    if (date) {
      conditions.push('`date` = ?');
      params.push(date);
    }
    if (employeeId) {
      conditions.push('`employeeId` = ?');
      params.push(employeeId);
    }

    const where = conditions.length ? `WHERE ${conditions.join(' AND ')}` : '';
    const countResult: any[] = await prisma.$queryRawUnsafe(
      `SELECT COUNT(*) as total FROM attendance_records ${where}`,
      ...params
    );
    const total = Number(countResult[0]?.total ?? 0);

    const records: any[] = await prisma.$queryRawUnsafe(
      `SELECT * FROM attendance_records ${where} ORDER BY \`date\` DESC, checkIn DESC LIMIT ? OFFSET ?`,
      ...params, limit, offset
    );

    return NextResponse.json({
      success: true,
      attendance: records,
      pagination: { page, limit, total },
    });
  } catch (error: any) {
    console.error('[ADMIN_ATTENDANCE_LIST]', error);
    return NextResponse.json({ success: true, attendance: [], pagination: { page, limit, total: 0 } });
  }
}

// POST /api/admin/attendance
export async function POST(req: NextRequest) {
  const session = await getServerSession(authOptions);
  if (!session) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  try {
    const body = await req.json();
    const id = crypto.randomUUID();
    const now = new Date().toISOString().slice(0, 19).replace('T', ' ');
    await prisma.$executeRawUnsafe(
      `INSERT INTO attendance_records (id, employeeId, \`date\`, checkIn, checkOut, status, notes, locationLat, locationLng, createdAt, updatedAt)
       VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
      id, body.employeeId, body.date, body.checkIn ?? null, body.checkOut ?? null,
      body.status ?? 'PRESENT', body.notes ?? null,
      body.locationLat ?? null, body.locationLng ?? null, now, now
    );
    return NextResponse.json({ success: true, attendance: { id, ...body } }, { status: 201 });
  } catch (error: any) {
    console.error('[ADMIN_ATTENDANCE_CREATE]', error);
    return NextResponse.json({ error: error.message }, { status: 500 });
  }
}
