# 1. Start the server
./goqr

# 2. Generate a certificate
./goqr generate-cert -id S123
or for a range
./goqr generate-cert -id S123-S223

# 3. Clean up old files
./goqr cleanup -days 20 



## Cronjob to automate cleanups
crontab -e

0 2 * * * /home/bitnami/work/goqr/./goqr cleanup -days 20 >> /home/bitnami/work/goqr/cleanup.log 2>&1