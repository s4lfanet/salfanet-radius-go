import { NextRequest, NextResponse } from 'next/server';
import { prisma } from '@/server/db/client';
import { getServerSession } from 'next-auth';
import { authOptions } from '@/server/auth/config';

// GET /api/admin/attendance-locations
export async function GET(req: NextRequest) {
  const session = await getServerSession(authOptions);
  if (!session) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  try {
    const locations: any[] = await prisma.$queryRawUnsafe(
      'SELECT * FROM attendance_locations WHERE isActive = 1 ORDER BY name ASC'
    );
    return NextResponse.json({ success: true, locations });
  } catch (error: any) {
    console.error('[ADMIN_ATTENDANCE_LOCATIONS_LIST]', error);
    return NextResponse.json({ success: true, locations: [] });
  }
}

// POST /api/admin/attendance-locations
export async function POST(req: NextRequest) {
  const session = await getServerSession(authOptions);
  if (!session) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  try {
    const body = await req.json();
    const id = crypto.randomUUID();
    const now = new Date().toISOString().slice(0, 19).replace('T', ' ');
    await prisma.$executeRawUnsafe(
      'INSERT INTO attendance_locations (id, name, lat, lng, radius, isActive, createdAt) VALUES (?, ?, ?, ?, ?, ?, ?)',
      id, body.name, body.lat ?? 0, body.lng ?? 0, body.radius ?? 100, 1, now
    );
    return NextResponse.json({ success: true, location: { id, ...body, isActive: true } }, { status: 201 });
  } catch (error: any) {
    console.error('[ADMIN_ATTENDANCE_LOCATIONS_CREATE]', error);
    return NextResponse.json({ error: error.message }, { status: 500 });
  }
}
