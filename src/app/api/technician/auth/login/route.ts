import { NextResponse } from 'next/server';

// Password-based login disabled — technicians use OTP (WhatsApp) login only.
export async function POST() {
  return NextResponse.json(
    { error: 'Login dengan password tidak didukung. Gunakan login OTP via WhatsApp.' },
    { status: 410 },
  );
}
