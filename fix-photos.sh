#!/bin/bash
DB="salfanet_radius"
USER="salfanet_user"
PASS="9wVyHRbaBxC7ylhBLr0K"

echo "=== Check installationPhotos values ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT id, username, installationPhotos FROM pppoe_users WHERE installationPhotos IS NOT NULL AND installationPhotos != '' AND installationPhotos != '[]'" 2>/dev/null
echo
echo "=== Check for invalid JSON ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT id, username, installationPhotos FROM pppoe_users WHERE installationPhotos IS NOT NULL AND installationPhotos != '' AND installationPhotos != '[]' AND JSON_VALID(installationPhotos) = 0" 2>/dev/null
echo
echo "=== Fix any invalid JSON to [] ==="
mysql -u $USER -p"$PASS" $DB -e "UPDATE pppoe_users SET installationPhotos = '[]' WHERE installationPhotos IS NULL OR installationPhotos = '' OR JSON_VALID(installationPhotos) = 0" 2>/dev/null
echo "Done"
echo
echo "=== Verify all are valid now ==="
mysql -u $USER -p"$PASS" $DB -e "SELECT id, username, installationPhotos, JSON_VALID(installationPhotos) as valid FROM pppoe_users LIMIT 10" 2>/dev/null
