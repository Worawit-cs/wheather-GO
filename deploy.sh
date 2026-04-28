#!/bin/bash
set -e
git pull
go build -o server .
sudo systemctl restart weather-server
echo "Deployed successfully."
