import { NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/server/auth/config'
import { readFile, writeFile } from 'fs/promises'
import { exec as execCb } from 'child_process'
import { promisify } from 'util'

const exec = promisify(execCb)

const L2TP_INFO_FILE = '/etc/salfanet/l2tp/l2tp-server-info.json'
const L2TP_CONF_DIR = '/etc/salfanet/l2tp'

export async function GET() {
  const session = await getServerSession(authOptions)
  if (!session) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })

  // 1. Try the JSON info file first (fastest path)
  try {
    const raw = await readFile(L2TP_INFO_FILE, 'utf8')
    const info = JSON.parse(raw)
    return NextResponse.json({ installed: true, ...info })
  } catch { /* fall through to fallback detection */ }

  // 2. Fallback: detect from xl2tpd.conf if info file missing
  // (e.g. L2TP was installed manually or info file was deleted)
  try {
    const xl2tpdConf = await readFile('/etc/xl2tpd/xl2tpd.conf', 'utf8')

    // Parse ip range = start-end
    let poolStart = '10.201.0.10'
    let poolEnd = '10.201.0.254'
    const rangeMatch = xl2tpdConf.match(/ip range\s*=\s*([\d.]+)-([\d.]+)/)
    if (rangeMatch) { poolStart = rangeMatch[1]; poolEnd = rangeMatch[2] }

    // Parse local ip
    let localIp = '10.201.0.1'
    const localMatch = xl2tpdConf.match(/local ip\s*=\s*([\d.]+)/)
    if (localMatch) localIp = localMatch[1]

    // Derive subnet from localIp (/24 assumption)
    const octets = localIp.split('.')
    octets[3] = '0'
    const subnet = `${octets.join('.')}/24`

    // Read PSK from salfanet conf dir first, then ipsec.secrets
    let ipsecPsk = ''
    try { ipsecPsk = (await readFile(`${L2TP_CONF_DIR}/ipsec.psk`, 'utf8')).trim() } catch { /* ignore */ }
    if (!ipsecPsk) {
      try {
        const secrets = await readFile('/etc/ipsec.secrets', 'utf8')
        const pskMatch = secrets.match(/PSK\s+"([^"]+)"/)
        if (pskMatch) ipsecPsk = pskMatch[1]
      } catch { /* ignore */ }
    }

    // Get public IP
    let publicIp = ''
    try {
      const { stdout } = await exec('curl -4 -s --connect-timeout 5 ifconfig.me 2>/dev/null', { shell: '/bin/bash' })
      publicIp = stdout.trim()
    } catch { /* ignore */ }

    const infoData = {
      type: 'l2tp-ipsec',
      localIp,
      subnet,
      poolStart,
      poolEnd,
      ipsecPsk,
      publicIp,
      recoveredAt: new Date().toISOString(),
    }

    // Re-write JSON file so next load is instant (fire-and-forget)
    writeFile(L2TP_INFO_FILE, JSON.stringify(infoData, null, 2) + '\n', 'utf8').catch(() => {/* ignore */})

    return NextResponse.json({ installed: true, ...infoData })
  } catch { /* xl2tpd.conf not readable → L2TP truly not installed */ }

  return NextResponse.json({
    installed: false,
    message: 'L2TP/IPsec server belum di-install di VPS ini. Jalankan install-l2tp-server.sh dulu.',
  })
}
