#!/bin/bash

CURRENT_USER=$(whoami)

if [ "$CURRENT_USER" != "root" ]; then
    echo "Error: You must be logged in as root to run this script."
    exit 1
fi

REPO_ID="juliotorresmoreno/kitty"
LATEST="$BASE/latest"
LIPSTICK_PATH="/usr/local/bin"
LIPSTICK_CLIENT_PATH=$LIPSTICK_PATH/lipstick-client
LIPSTICK_SERVER_PATH=$LIPSTICK_PATH/lipstick-server

if [ -e "$LIPSTICK_CLIENT_PATH" ]; then
    rm "$LIPSTICK_CLIENT_PATH"
fi
if [ -e "$LIPSTICK_SERVER_PATH" ]; then
    rm "$LIPSTICK_SERVER_PATH"
fi

cd $LIPSTICK_PATH

RELEASES_JSON=$(curl -s "https://api.github.com/repos/$REPO_ID/releases/latest")
DOWNLOAD_URL=$(echo "$RELEASES_JSON" | jq -r .assets[0].browser_download_url)
wget $DOWNLOAD_URL

RELEASES_JSON=$(curl -s "https://api.github.com/repos/$REPO_ID/releases/latest")
DOWNLOAD_URL=$(echo "$RELEASES_JSON" | jq -r .assets[1].browser_download_url)
wget $DOWNLOAD_URL

chmod +x $LIPSTICK_CLIENT_PATH
chmod +x $LIPSTICK_SERVER_PATH
