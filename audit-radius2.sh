#!/bin/bash
DB="salfanet_radius"
USER="salfanet_user"
PASS="9wVyHRbaBxC7ylhBLr0K"

echo "=== pppoe_profiles FULL ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT * FROM pppoe_profiles\G" 2>/dev/null
echo
echo "=== pppoe_profiles table structure ==="
mysql -u $USER -p"$PASS" $DB -e "DESCRIBE pppoe_profiles" 2>/dev/null
echo
echo "=== All tables with 'pool' in name ==="
mysql -u $USER -p"$PASS" $DB -e "SHOW TABLES LIKE '%pool%'" 2>/dev/null
echo
echo "=== All tables with 'ip' in name ==="
mysql -u $USER -p"$PASS" $DB -e "SHOW TABLES LIKE '%ip%'" 2>/dev/null
echo
echo "=== nas table ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT id, nasname, shortname, secret, type, isActive FROM nas" 2>/dev/null
echo
echo "=== FreeRADIUS sql module config (ippool section) ==="
grep -A20 'ippool\|pool_name\|ip_pool' /etc/freeradius/3.0/mods-enabled/sql 2>/dev/null
echo
echo "=== FreeRADIUS available modules for ippool ==="
ls /etc/freeradius/3.0/mods-available/ | grep -i pool
echo
echo "=== MikroTik dictionary check ==="
grep -r 'Mikrotik\|mikrotik' /usr/share/freeradius/dictionary* 2>/dev/null | head -5
grep -r 'Mikrotik\|mikrotik' /etc/freeradius/3.0/dictionary* 2>/dev/null | head -5
echo
echo "=== radreply duplicates check ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT username, attribute, COUNT(*) as cnt FROM radreply GROUP BY username, attribute HAVING cnt > 1" 2>/dev/null
echo
echo "=== radacct (recent sessions) ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT radacctid, username, nasipaddress, acctstarttime, acctstoptime, framedipaddress FROM radacct ORDER BY acctstarttime DESC LIMIT 5" 2>/dev/null
