#!/bin/bash
DB="salfanet_radius"
USER="salfanet_user"
PASS="9wVyHRbaBxC7ylhBLr0K"

echo "=== Users in radcheck ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT username, attribute, op, value FROM radcheck ORDER BY username LIMIT 20" 2>/dev/null
echo
echo "=== Count radcheck users ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT COUNT(DISTINCT username) as total_users FROM radcheck" 2>/dev/null
echo
echo "=== Count pppoe_users ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT COUNT(*) as total, SUM(CASE WHEN status='active' THEN 1 ELSE 0 END) as active FROM pppoe_users" 2>/dev/null
echo
echo "=== pppoe_users not in radcheck ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT u.username, u.status FROM pppoe_users u LEFT JOIN radcheck r ON u.username = r.username WHERE r.username IS NULL AND u.status='active' LIMIT 20" 2>/dev/null
echo
echo "=== radpostauth: Access-Reject users (last 5 unique) ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT DISTINCT username, reply FROM radpostauth WHERE reply='Access-Reject' ORDER BY authdate DESC LIMIT 10" 2>/dev/null
