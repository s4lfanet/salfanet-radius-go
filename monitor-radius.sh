#!/bin/bash
DB="salfanet_radius"
USER="salfanet_user"
PASS="9wVyHRbaBxC7ylhBLr0K"

echo "=== Waiting 30s for MikroTik RADIUS requests ==="
sleep 30
echo
echo "=== FreeRADIUS log (last 30 lines) ==="
tail -30 /var/log/freeradius/radius.log 2>/dev/null
echo
echo "=== radacct (recent sessions) ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT radacctid, username, nasipaddress, acctstarttime, acctstoptime, framedipaddress FROM radacct ORDER BY acctstarttime DESC LIMIT 10" 2>/dev/null
echo
echo "=== radpostauth (recent auth attempts) ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT id, username, reply, authdate FROM radpostauth ORDER BY authdate DESC LIMIT 10" 2>/dev/null
