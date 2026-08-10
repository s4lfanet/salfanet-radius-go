#!/bin/bash
DB="salfanet_radius"
USER="salfanet_user"
PASS="9wVyHRbaBxC7ylhBLr0K"

echo "=== Check nas table structure ==="
mysql -u $USER -p"$PASS" $DB -e "DESCRIBE nas" 2>/dev/null
echo
echo "=== Check for unique constraint on nasname ==="
mysql -u $USER -p"$PASS" $DB -e "SHOW INDEX FROM nas" 2>/dev/null
echo
echo "=== Try insert again with different ID ==="
mysql -u $USER -p"$PASS" $DB -e "INSERT INTO nas (id, nasname, shortname, secret, type, isActive, ipAddress, auth_mode, createdAt, updatedAt) VALUES ('nas-pub-001', '103.191.165.120', 'dst-paska-pub', 'secret123', 'mikrotik', 1, '103.191.165.120', 'radius', NOW(), NOW())" 2>&1
echo
echo "=== Verify ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT id, nasname, shortname, secret, isActive FROM nas" 2>/dev/null
