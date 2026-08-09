#!/bin/bash
DBPASS=$(grep DB_PASSWORD /var/www/salfanet-radius/.env | head -1 | cut -d= -f2 | tr -d '"')
mysql -u salfanet_user -p"$DBPASS" salfanet_radius -e "SELECT isolationServerIp, baseUrl FROM companies LIMIT 1;" 2>/dev/null
