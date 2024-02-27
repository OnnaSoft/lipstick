# Lipstick

This project allow share local ports to remote servers inmediatly withow vpn. At moment is experimental and required authentication, but you can add it from nginx.

## Overview
Here a small text explain what exactly is what thing.

### Server
All parts are servers because server content but in this case server is related with the real server in your implementation, i wan to say, server is the remote host where you try to get your local port.

```

**How to use**

Usage of lipstick-server:
```text
  -c string
    	config path (default "config.client.yml")
  -m string
    	Port where your client will connect via websocket. You can manage it in your firewall (default ":8081")
  -p string
    	Port where you will get all requests from local network or internet (default ":8080")
```

### Client
This is the machine where exists your web servere what you want share with others.


**How to use**

Usage of lipstick-client:

```text
  -c string
    	config path (default "/tmp/go-build3372496926/b001/exe/config.client.yml")
  -p string
    	Host/port where you want connect from the remote server (default "127.0.0.1:8082")
  -s string
    	Where you are listening your server manager port (default "ws://localhost:8081/ws")
```

## Run client on docker
```bash
docker run --entrypoint lipstick --name lipstick --network host --restart always -dt jliotorresmoreno/lipstick -s ws://127.0.0.1:5051/ws -k 123456
```

```bash
docker run --entrypoint lipstick --name lipstick --network host --restart always -dt jliotorresmoreno/lipstick -s wss://example.com/ws -k 123456
```

## Run server on docker
```bash
docker run --name lipstickd --network host --restart always -d jliotorresmoreno/lipstick -k 123456
```

```bash
docker run --entrypoint lipstickd --name lipstickd --network host \
  -v /etc/letsencrypt:/etc/letsencrypt --restart always -d jliotorresmoreno/lipstick -k 123456 \
  -cert /etc/letsencrypt/live/example.com/fullchain.pem -key /etc/letsencrypt/live/example.com/privkey.pem
```