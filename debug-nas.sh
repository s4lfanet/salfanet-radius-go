#!/bin/bash
DB="salfanet_radius"
USER="salfanet_user"
PASS="9wVyHRbaBxC7ylhBLr0K"

echo "=== Current NAS ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT id, nasname, shortname, secret, type FROM nas" 2>/dev/null
echo
echo "=== FreeRADIUS clients.d ==="
ls -la /etc/freeradius/3.0/clients.d/ 2>/dev/null
echo
cat /etc/freeradius/3.0/clients.d/*.conf 2>/dev/null
echo
echo "=== FreeRADIUS clients.conf ==="
grep -v '^\s*#' /etc/freeradius/3.0/clients.conf 2>/dev/null | grep -v '^\s*$' | head -30
echo
echo "=== Check if 192.168.54.1 is registered ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT * FROM nas WHERE nasname='192.168.54.1'" 2>/dev/null
