#!/bin/bash
mysql -u salfanet_user -phBH0ReZP9WzmG9iE81Cg salfanet_radius -e "SELECT email,username,role FROM admin_users LIMIT 5" 2>/dev/null
echo "---"
mysql -u salfanet_user -phBH0ReZP9WzmG9iE81Cg salfanet_radius -e "DESCRIBE admin_users" 2>/dev/null
