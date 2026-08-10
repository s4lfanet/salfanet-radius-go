#!/bin/bash
DB="salfanet_radius"
USER="salfanet_user"
PASS="9wVyHRbaBxC7ylhBLr0K"

echo "=== pppoe_users by status ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT status, COUNT(*) as count FROM pppoe_users GROUP BY status" 2>/dev/null
echo
echo "=== pppoe_users with empty username/password ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT COUNT(*) as empty_creds FROM pppoe_users WHERE username='' OR password='' OR username IS NULL OR password IS NULL" 2>/dev/null
echo
echo "=== All pppoe_users (username, status, profileId) ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT username, status, profileId, syncedToRadius FROM pppoe_users LIMIT 20" 2>/dev/null
echo
echo "=== radgroupreply for all groups ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT groupname, attribute, op, value FROM radgroupreply ORDER BY groupname, attribute" 2>/dev/null
echo
echo "=== Recent radacct (check session duration) ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT radacctid, username, nasipaddress, acctstarttime, acctstoptime, framedipaddress, TIMESTAMPDIFF(SECOND, acctstarttime, acctstoptime) as duration_sec FROM radacct ORDER BY acctstarttime DESC LIMIT 10" 2>/dev/null
