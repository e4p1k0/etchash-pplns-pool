#!/bin/bash

#ellamain:
screen -dmS etscgeth /path/to/geth --rpc --ws --unlock="./etscuser" --password="./etscpass" --max-peers=500 --extra-data="open-social-pool" --identity="open-social-pool" --cache=128 --ethstats "open-social-pool:NewEthereumSocial@stats.ethereumsocial.kr"

sleep 5

#pool2b:
screen -dmS etscpool2b /var/www/open-social-pool/open-ethereum-pool /var/www/open-social-pool/pool2b.json

sleep 5

#pool4b:
screen -dmS etscpool4b /var/www/open-social-pool/open-ethereum-pool /var/www/open-social-pool/pool4b.json

sleep 5

#pool9b:
screen -dmS etscpool9b /var/www/open-social-pool/open-ethereum-pool /var/www/open-social-pool/pool9b.json

sleep 5

#api:
screen -dmS etscapi /var/www/open-social-pool/open-ethereum-pool /var/www/open-social-pool/api.json

sleep 5

#unlocker:
screen -dmS etscunlocker /var/www/open-social-pool/open-ethereum-pool /var/www/open-social-pool/unlocker.json

sleep 5

#payout:
screen -dmS etscpayout /var/www/open-social-pool/open-ethereum-pool /var/www/open-social-pool/payout.json

sleep 5

# Sample start for stats reporting to stats.ellaism.org
#netintelligence:
#cd /root/ellaism-net-intelligence-api/ && pm2 start /root/ellaism-net-intelligence-api/app.json

exit 0
