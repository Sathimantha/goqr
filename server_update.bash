cp .env .env.bkp
sudo systemctl stop goqr.service
git pull
rm .env
mv .env.bkp .env
go build
sudo systemctl start goqr.service
