import { NextResponse } from 'next/server';
import { getServerSession } from 'next-auth';
import { authOptions } from '@/server/auth/config';
import { execSync, execFileSync } from 'child_process';
import { readFileSync, existsSync } from 'fs';
import path from 'path';

export const dynamic = 'force-dynamic';

function getAppDir(): string {
  const candidates = [
    process.env.SALFANET_APP_DIR,
    '/var/www/salfanet-frontend',
    '/var/www/salfanet-radius',
    path.resolve(process.cwd(), '../..'),
    process.cwd(),
  ].filter(Boolean) as string[];

  for (const dir of candidates) {
    if (existsSync(path.join(dir, 'package.json'))) {
      return dir;
    }
  }

  return '/var/www/salfanet-radius';
}

/** Try git at appDir, then SOURCE_DIR candidates (when installed from zip, git lives in /root/salfanet-radius-go) */
function getGitDir(appDir: string): string {
  if (existsSync(path.join(appDir, '.git'))) return appDir;
  const sourceCandidates = ['/root/salfanet-radius-go', '/root/salfanet-radius'];
  for (const d of sourceCandidates) {
    if (existsSync(path.join(d, '.git'))) return d;
  }
  return appDir; // fallback even if no .git
}

function git(cmd: string, dir: string): string {
  try {
    return execSync(cmd, { cwd: dir, timeout: 5000, stdio: 'pipe' }).toString().trim();
  } catch {
    return 'unknown';
  }
}

function readCommitFile(appDir: string, name: string): string {
  const p = path.join(appDir, name);
  try {
    return existsSync(p) ? readFileSync(p, 'utf-8').trim() : '';
  } catch {
    return '';
  }
}

/** Check GitHub API for latest commit on master. Works for private repos when GITHUB_TOKEN is set. */
async function getRemoteCommit(githubRepo: string): Promise<string> {
  const token = process.env.GITHUB_TOKEN;
  const headers: Record<string, string> = {
    'Accept': 'application/vnd.github.v3+json',
    'User-Agent': 'salfanet-radius',
  };
  if (token) headers['Authorization'] = `Bearer ${token}`;

  try {
    const res = await fetch(
      `https://api.github.com/repos/${githubRepo}/commits/master`,
      { headers, signal: AbortSignal.timeout(8000) }
    );
    if (!res.ok) return 'unknown';
    const data = await res.json() as { sha?: string };
    return data.sha ?? 'unknown';
  } catch {
    return 'unknown';
  }
}

export async function GET() {
  const session = await getServerSession(authOptions);
  if (!session) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
  if (session.user.role !== 'SUPER_ADMIN') {
    return NextResponse.json({ error: 'Forbidden' }, { status: 403 });
  }

  const appDir  = getAppDir();
  const gitDir  = getGitDir(appDir);
  const GITHUB_REPO = 's4lfanet/salfanet-radius-go';

  const pkgPath = path.join(appDir, 'package.json');
  const pkg = JSON.parse(readFileSync(pkgPath, 'utf-8'));

  // Prefer VERSION file (written by release CI)
  const versionFilePath = path.join(appDir, 'VERSION');
  const appVersion = existsSync(versionFilePath)
    ? readFileSync(versionFilePath, 'utf-8').trim().replace(/^v/, '')
    : pkg.version;

  // Try git commands first, fall back to COMMIT_HASH/COMMIT_DATE/COMMIT_MSG files
  // (files are written by updater.sh after each git pull)
  let localCommit = git('git rev-parse HEAD', gitDir);
  let commitDate  = git('git log -1 --format="%ci"', gitDir);
  let commitMsg   = git('git log -1 --format="%s"', gitDir);

  if (localCommit === 'unknown') localCommit = readCommitFile(appDir, 'COMMIT_HASH');
  if (!commitDate || commitDate === 'unknown') commitDate = readCommitFile(appDir, 'COMMIT_DATE');
  if (!commitMsg  || commitMsg  === 'unknown') commitMsg  = readCommitFile(appDir, 'COMMIT_MSG');

  const shortCommit = localCommit && localCommit !== 'unknown'
    ? localCommit.slice(0, 7)
    : 'unknown';

  // Check remote commit — try git fetch first, then GitHub API
  let remoteCommit = 'unknown';
  let hasUpdate    = false;
  try {
    execSync('git fetch origin master --quiet', { cwd: gitDir, timeout: 10000, stdio: 'pipe' });
    remoteCommit = git('git rev-parse origin/master', gitDir);
  } catch { /* network unavailable or no .git */ }

  if (remoteCommit === 'unknown') {
    remoteCommit = await getRemoteCommit(GITHUB_REPO);
  }

  if (localCommit && localCommit !== 'unknown' && remoteCommit !== 'unknown') {
    hasUpdate = localCommit !== remoteCommit;
  }

  const logExists = existsSync('/tmp/salfanet-update.log');
  const pidExists = existsSync('/tmp/salfanet-update.pid');
  let updateRunning = false;
  if (pidExists) {
    try {
      const pid = parseInt(readFileSync('/tmp/salfanet-update.pid', 'utf-8').trim());
      if (Number.isInteger(pid) && pid > 0) {
        execFileSync('kill', ['-0', pid.toString()], { timeout: 2000, stdio: 'pipe' });
        updateRunning = true;
      } else {
        updateRunning = false;
      }
    } catch { updateRunning = false; }
  }

  return NextResponse.json({
    version:       appVersion,
    commit:        shortCommit,
    commitFull:    localCommit || 'unknown',
    commitDate:    commitDate  || '',
    commitMessage: commitMsg   || '',
    remoteCommit:  remoteCommit !== 'unknown' ? remoteCommit.slice(0, 7) : 'unknown',
    hasUpdate,
    updateRunning,
    logExists,
    nodeVersion:   process.version,
    platform:      process.platform,
    uptime:        Math.floor(process.uptime()),
  });
}
