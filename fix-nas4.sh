#!/bin/bash
DB="salfanet_radius"
USER="salfanet_user"
PASS="9wVyHRbaBxC7ylhBLr0K"

echo "=== Insert second NAS with all required fields ==="
mysql -u $USER -p"$PASS" $DB -e "INSERT INTO nas (id, name, nasname, shortname, type, isActive, ipAddress, username, password, port, apiPort, secret, ports, auth_mode, createdAt, updatedAt) VALUES ('nas-pub-001', 'dst paska public', '103.191.165.120', 'dst-paska-pub', 'mikrotik', 1, '103.191.165.120', '', '', 8728, 8729, 'secret123', 1812, 'radius', NOW(), NOW())" 2>&1
echo
echo "=== Verify ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT id, name, nasname, shortname, secret, isActive FROM nas" 2>/dev/null
echo
echo "=== Restart FreeRADIUS ==="
systemctl restart freeradius
sleep 2
echo
echo "=== Check log for errors ==="
tail -5 /var/log/freeradius/radius.log 2>/dev/null
echo
echo "=== Wait 10s and check if MikroTik requests are accepted ==="
sleep 10
tail -20 /var/log/freeradius/radius.log 2>/dev/null
