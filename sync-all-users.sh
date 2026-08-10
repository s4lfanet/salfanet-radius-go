#!/bin/bash
DB="salfanet_radius"
USER="salfanet_user"
PASS="9wVyHRbaBxC7ylhBLr0K"

echo "=== Sync all active PPPoE users to RADIUS ==="
echo "Step 1: Get all active users with their profiles..."

# Get all active pppoe_users with profile info
mysql -u $USER -p"$PASS" $DB -N -e "
SELECT u.username, u.password, p.groupName, p.rateLimit, p.IPPoolName, u.status
FROM pppoe_users u
LEFT JOIN pppoe_profiles p ON u.profileId = p.id
WHERE u.username != '' AND u.password != '' AND u.status = 'active'
" 2>/dev/null | while IFS=$'\t' read -r username password groupName rateLimit poolName status; do
  
  # Determine group
  group="$groupName"
  if [ -z "$group" ]; then
    group="default"
  fi
  
  # 1. Insert/update radcheck (Cleartext-Password)
  mysql -u $USER -p"$PASS" $DB -e "DELETE FROM radcheck WHERE username='$username' AND attribute='Cleartext-Password'" 2>/dev/null
  mysql -u $USER -p"$PASS" $DB -e "INSERT INTO radcheck (username, attribute, op, value) VALUES ('$username', 'Cleartext-Password', ':=', '$password')" 2>/dev/null
  
  # 2. Insert/update radusergroup
  mysql -u $USER -p"$PASS" $DB -e "DELETE FROM radusergroup WHERE username='$username'" 2>/dev/null
  mysql -u $USER -p"$PASS" $DB -e "INSERT INTO radusergroup (username, groupname, priority) VALUES ('$username', '$group', 1)" 2>/dev/null
  
  # 3. Insert/update radreply (Mikrotik-Rate-Limit) - only if rateLimit is set
  if [ -n "$rateLimit" ] && [ "$rateLimit" != "NULL" ]; then
    mysql -u $USER -p"$PASS" $DB -e "DELETE FROM radreply WHERE username='$username' AND attribute='Mikrotik-Rate-Limit'" 2>/dev/null
    mysql -u $USER -p"$PASS" $DB -e "INSERT INTO radreply (username, attribute, op, value) VALUES ('$username', 'Mikrotik-Rate-Limit', '=', '$rateLimit')" 2>/dev/null
  fi
  
  echo "Synced: $username -> group=$group, rate=$rateLimit"
done

echo
echo "=== Verify: count radcheck users ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT COUNT(DISTINCT username) as total FROM radcheck" 2>/dev/null
echo
echo "=== Verify: count radusergroup ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT COUNT(DISTINCT username) as total FROM radusergroup" 2>/dev/null
echo
echo "=== Verify: sample ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT username, groupname FROM radusergroup LIMIT 10" 2>/dev/null
echo
echo "=== Update syncedToRadius flag ==="
mysql -u $USER -p"$PASS" $DB -e "UPDATE pppoe_users SET syncedToRadius=1 WHERE status='active' AND username != '' AND password != ''" 2>/dev/null
echo "Done"
