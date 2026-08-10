#!/bin/bash
DB="salfanet_radius"
USER="salfanet_user"
PASS="9wVyHRbaBxC7ylhBLr0K"

echo "=== radcheck ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT * FROM radcheck" 2>/dev/null
echo
echo "=== radreply ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT * FROM radreply" 2>/dev/null
echo
echo "=== radgroupcheck ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT * FROM radgroupcheck" 2>/dev/null
echo
echo "=== radgroupreply ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT * FROM radgroupreply" 2>/dev/null
echo
echo "=== radusergroup ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT * FROM radusergroup" 2>/dev/null
echo
echo "=== pppoe_profiles ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT id, name, rateLimitRx, rateLimitTx, poolName, syncedToRadius FROM pppoe_profiles" 2>/dev/null
echo
echo "=== pppoe_users (with profile) ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT u.id, u.username, u.status, u.expiredAt, u.profileId, p.name as profile_name FROM pppoe_users u LEFT JOIN pppoe_profiles p ON u.profileId = p.id" 2>/dev/null
echo
echo "=== radippool table exists? ==="
mysql -u $USER -p"$PASS" $DB -e "SHOW TABLES LIKE '%pool%'" 2>/dev/null
echo
echo "=== radippool entries ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT * FROM radippool LIMIT 10" 2>/dev/null
echo
echo "=== FreeRADIUS ippool module enabled? ==="
ls -la /etc/freeradius/3.0/mods-enabled/ippool 2>&1
echo
echo "=== FreeRADIUS sites-enabled/default (full) ==="
cat /etc/freeradius/3.0/sites-enabled/default
