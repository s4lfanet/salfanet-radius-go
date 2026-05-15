import { NextResponse } from 'next/server';
import { getServerSession } from 'next-auth';
import { authOptions } from '@/server/auth/config';
import { spawn } from 'child_process';
import { existsSync, readFileSync, openSync, writeFileSync } from 'fs';
import path from 'path';

export const dynamic = 'force-dynamic';

const LOG_FILE = '/tmp/salfanet-update.log';
const PID_FILE = '/tmp/salfanet-update.pid';

function getAppDir(): string {
  const candidates = [
    process.env.SALFANET_APP_DIR,
    '/var/www/salfanet-radius',
    process.cwd(),
  ].filter(Boolean) as string[];
  for (const dir of candidates) {
    if (existsSync(path.join(dir, 'package.json'))) return dir;
  }
  return '/var/www/salfanet-radius';
}

function isUpdateRunning(): boolean {
  if (!existsSync(PID_FILE)) return false;
  try {
    const pid = parseInt(readFileSync(PID_FILE, 'utf-8').trim());
    if (!Number.isInteger(pid) || pid <= 0) return false;
    process.kill(pid, 0);
    return true;
  } catch {
    return false;
  }
}

/** POST /api/admin/system/update — Trigger updater.sh in background */
export async function POST() {
  const session = await getServerSession(authOptions);
  if (!session) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
  if (session.user.role !== 'SUPER_ADMIN') {
    return NextResponse.json({ error: 'Forbidden' }, { status: 403 });
  }

  if (isUpdateRunning()) {
    const pid = readFileSync(PID_FILE, 'utf-8').trim();
    return NextResponse.json({ error: 'Update sedang berjalan', pid }, { status: 409 });
  }

  const appDir     = getAppDir();
  const updaterPath = path.join(appDir, 'vps-install', 'updater.sh');

  if (!existsSync(updaterPath)) {
    return NextResponse.json({ error: 'updater.sh tidak ditemukan di: ' + updaterPath }, { status: 404 });
  }

  // Clear previous log
  writeFileSync(LOG_FILE, `[${new Date().toISOString()}] Memulai update...\n`);

  const outFd = openSync(LOG_FILE, 'a');

  const env: Record<string, string> = { ...process.env as Record<string, string>, PATH: '/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin' };

  const child = spawn('bash', [updaterPath, '--branch', 'master', '--skip-backup'], {
    detached: true,
    stdio: ['ignore', outFd, outFd],
    env,
  });

  writeFileSync(PID_FILE, String(child.pid));
  child.unref();

  return NextResponse.json({ started: true, pid: child.pid }, { status: 202 });
}

/** GET /api/admin/system/update — Return current update log */
export async function GET() {
  const session = await getServerSession(authOptions);
  if (!session) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
  if (session.user.role !== 'SUPER_ADMIN') {
    return NextResponse.json({ error: 'Forbidden' }, { status: 403 });
  }

  const log     = existsSync(LOG_FILE) ? readFileSync(LOG_FILE, 'utf-8') : '';
  const running = isUpdateRunning();

  return NextResponse.json({ log, running });
}
