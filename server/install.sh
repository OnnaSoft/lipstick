#!/bin/sh

app_name=lipstick
bin_path=/tmp/$app_name-6415415

if [ $USER != "root" ]; then
  echo "No eres root"
  exit 1
fi

mkdir -p bin
rm -rf bin/*
GOOS=linux GOARCH=amd64 go build -o $bin_path

systemctl stop $app_name > /dev/null
cp $bin_path /usr/bin/$app_name
mkdir -p /etc/$app_name

cp config.yml /etc/$app_name &> /dev/null
cp config.yml.example /etc/$app_name &> /dev/null
cp $app_name.service /etc/systemd/system
adduser $app_name \
    --gecos "" \
    --system \
    --no-create-home \
    --disabled-password \
    --disabled-login > /dev/null
systemctl daemon-reload
systemctl start $app_name
systemctl status $app_name

