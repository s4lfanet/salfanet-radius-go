import { NextRequest, NextResponse } from 'next/server';
import { jwtVerify } from 'jose';
import { prisma } from '@/server/db/client';
import { TECH_JWT_SECRET } from '@/server/auth/technician-secret';

async function getTechUser(req: NextRequest) {
  const token = req.cookies.get('technician-token')?.value;
  if (!token) return null;
  try {
    const { payload } = await jwtVerify(token, TECH_JWT_SECRET);
    return { id: payload.id as string };
  } catch {
    return null;
  }
}

export async function GET(req: NextRequest) {
  try {
    const auth = await getTechUser(req);
    if (!auth) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

    const tech = await prisma.technician.findUnique({
      where: { id: auth.id },
      select: { id: true, name: true, phoneNumber: true, email: true, createdAt: true },
    });
    if (!tech) return NextResponse.json({ error: 'Not found' }, { status: 404 });
    return NextResponse.json({ success: true, profile: { ...tech, username: tech.phoneNumber, phone: tech.phoneNumber } });
  } catch (error) {
    console.error('Get technician profile error:', error);
    return NextResponse.json({ error: 'Server error' }, { status: 500 });
  }
}

export async function PUT(req: NextRequest) {
  try {
    const auth = await getTechUser(req);
    if (!auth) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

    const body = await req.json();
    const { name, email } = body;

    const updateData: Record<string, string> = {};
    if (name) updateData.name = name;
    if (email !== undefined) updateData.email = email;

    await prisma.technician.update({ where: { id: auth.id }, data: updateData });
    return NextResponse.json({ success: true, message: 'Profile updated' });
  } catch (error) {
    console.error('Update technician profile error:', error);
    return NextResponse.json({ error: 'Server error' }, { status: 500 });
  }
}
