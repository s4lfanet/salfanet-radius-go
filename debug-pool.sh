#!/bin/bash
DB="salfanet_radius"
USER="salfanet_user"
PASS="9wVyHRbaBxC7ylhBLr0K"

echo "=== radgroupreply (Framed-Pool) ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT * FROM radgroupreply WHERE attribute='Framed-Pool'" 2>/dev/null
echo
echo "=== radreply (Framed-Pool per user) ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT * FROM radreply WHERE attribute='Framed-Pool'" 2>/dev/null
echo
echo "=== radippool table exists? ==="
mysql -u $USER -p"$PASS" $DB -e "SHOW TABLES LIKE 'radippool'" 2>/dev/null
echo
echo "=== pppoe_profiles (IPPoolName) ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT id, name, groupName, IPPoolName, rateLimit FROM pppoe_profiles" 2>/dev/null
echo
echo "=== FreeRADIUS ippool module config ==="
cat /etc/freeradius/3.0/mods-available/ippool 2>/dev/null | head -30
echo
echo "=== FreeRADIUS sites-enabled default (ippool section) ==="
grep -A5 -B2 ippool /etc/freeradius/3.0/sites-enabled/default 2>/dev/null
echo
echo "=== FreeRADIUS dictionary check Framed-Pool ==="
grep Framed-Pool /etc/freeradius/3.0/share/dictionary* 2>/dev/null | head -5
echo
echo "=== radtest ==="
radtest server salfanet 127.0.0.1 0 testing123 2>&1 | tail -15
echo
echo "=== Recent radacct ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT radacctid, username, nasipaddress, acctstarttime, acctstoptime, framedipaddress, framedpool FROM radacct ORDER BY acctstarttime DESC LIMIT 10" 2>/dev/null
echo
echo "=== FreeRADIUS log (last 30 lines) ==="
tail -30 /var/log/freeradius/radius.log 2>/dev/null || journalctl -u freeradius -n 30 --no-pager 2>/dev/null
