-- Migration: Drop FK constraint nas_vpnClientId_fkey
-- vpnClientId on nas table can now reference vpn_clients.id OR vps_peers.id (no FK enforcement)
ALTER TABLE nas DROP FOREIGN KEY IF EXISTS nas_vpnClientId_fkey;
