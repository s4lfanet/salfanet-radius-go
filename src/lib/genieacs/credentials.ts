import 'server-only';

export interface GenieACSCredentials {
  host: string;
  username?: string;
  password?: string;
}

/**
 * Load GenieACS credentials from the `genieacs_settings` table.
 * Returns null if no active record is found.
 */
export async function getGenieACSCredentials(): Promise<GenieACSCredentials | null> {
  try {
    const { prisma } = await import('@/server/db/client');
    const row = await (prisma as any).genieacsSettings.findFirst({
      where: { isActive: true },
    });
    if (!row) return null;
    return { host: row.host, username: row.username, password: row.password };
  } catch {
    return null;
  }
}
