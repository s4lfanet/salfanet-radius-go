/**
 * Network Feature -- Domain Types
 *
 * @module features/network/types
 */

// Router matches Go Router struct (table: nas)
export type Router = {
  id: string
  name: string
  nasname: string
  shortname: string
  ipAddress: string
  username: string
  password: string
  port: number
  apiPort: number
  secret: string
  ports: number
  type: string
  isActive: boolean
  description: string | null
  latitude: number | null
  longitude: number | null
  vpnClientId: string | null
  createdAt: string
  updatedAt: string
}

// VpnClient matches Go vpnClient struct (table: vpn_clients)
export type VpnClient = {
  id: string
  name: string
  publicKey: string
  allowedIPs: string
  endpoint: string | null
  status: string
  description: string | null
  createdAt: string
  updatedAt: string
  approvedAt: string | null
}

// VpnServer matches Go vpnSite struct (table: vpn_sites)
export type VpnServer = {
  id: string
  name: string
  publicKey: string
  endpoint: string
  allowedIPs: string
  isActive: boolean
  description: string | null
  createdAt: string
  updatedAt: string
}

export type RouterWithStats = Router & {
  userCount: number
  voucherCount: number
}

export type NetworkTopology = {
  routers: RouterWithStats[]
  vpnServers: VpnServer[]
  vpnClients: VpnClient[]
}
