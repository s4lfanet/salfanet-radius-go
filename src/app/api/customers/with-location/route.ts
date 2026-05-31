import { NextResponse } from 'next/server';
import { checkAuth } from '@/server/middleware/api-auth';
import { prisma } from '@/server/db/client';

/**
 * GET /api/customers/with-location
 *
 * Returns PPPoE customers that have GPS coordinates set.
 * Served from Next.js (not Go) so that getServerSession handles auth —
 * bypassing the cookie-forwarding issue that occurs when Cloudflare
 * strips __Secure-* cookies before they reach the Go backend.
 *
 * Response: { success: true, data: [...], count: N }
 */
export async function GET(request: Request) {
  const auth = await checkAuth();
  if (!auth.authorized) return auth.response;

  const { searchParams } = new URL(request.url);
  const limit = parseInt(searchParams.get('limit') || '2000', 10);
  // Clamp limit to a safe range (1–5000)
  const safeLimit = Math.min(Math.max(1, isNaN(limit) ? 2000 : limit), 5000);

  const customers = await prisma.pppoeUser.findMany({
    where: {
      latitude: { not: null },
      longitude: { not: null },
    },
    take: safeLimit,
    select: {
      id: true,
      username: true,
      name: true,
      status: true,
      latitude: true,
      longitude: true,
      address: true,
      phone: true,
      email: true,
      customerId: true,
      profileId: true,
      routerId: true,
      area: {
        select: {
          id: true,
          name: true,
        },
      },
    },
  });

  return NextResponse.json({
    success: true,
    data: customers,
    count: customers.length,
  });
}
