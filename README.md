
# Lipstick Documentation

**Lipstick** is a tool that allows you to share local ports with remote servers immediately, without the need for a VPN. It is currently in an experimental phase and requires authentication. For added security, you can integrate it with a reverse proxy like Nginx.

---

## Overview

**Lipstick** provides a way to expose services running on your local machine to remote servers securely. Below is an explanation of its components and their configurations.

### Server

The **server** refers to the remote host where your local ports are exposed. It is responsible for managing client connections and forwarding traffic to the appropriate destinations.

#### How to Use

The `lipstick-server` application provides the following options:

```text
Usage of lipstick-server:
  -c string
    	Config path (default "config.server.yml")
  -m string
    	Port where your client will connect via WebSocket. You can manage this port in your firewall. (default ":8081")
  -p string
    	Port where you will receive requests from the local network or the internet. (default ":8080")
  -k string
        Secret key to authenticate clients. (default "super_secret_key")
  -cert string
        Path to the TLS certificate. (optional)
  -key string
        Path to the TLS key. (optional)
```

### Client

The **client** is the local machine running the service you want to expose. For example, this could be your local web server or any application listening on a specific port.

#### How to Use

The `lipstick-client` application provides the following options:

```text
Usage of lipstick-client:
  -c string
    	Config path (default "/etc/lipstick/config.client.yml")
  -p string
    	Host/port where you want to connect from the remote server. (default "127.0.0.1:8082")
  -s string
    	WebSocket URL of the server manager where your client will connect. (default "ws://localhost:8081/ws")
  -k string
        Secret key to authenticate with the server. (default "super_secret_key")
```

---

## Running Lipstick in Docker

### Run Client in Docker

You can run the Lipstick client in a Docker container to expose a local service securely:

#### Without SSL:

```bash
docker run --entrypoint lipstick --name lipstick --network host   --restart always -dt jliotorresmoreno/lipstick   -s ws://127.0.0.1:5051/ws   -p 127.0.0.1:8082   -k 123456
```

#### With SSL:

```bash
docker run --entrypoint lipstick --name lipstick --network host   --restart always -dt jliotorresmoreno/lipstick   -s wss://example.com/ws   -p 127.0.0.1:8082   -k 123456
```

### Run Server in Docker

To run the Lipstick server in a Docker container:

#### Without SSL:

```bash
docker run --name lipstickd --network host --restart always -d   jliotorresmoreno/lipstick   -p 8080   -m 8081   -k 123456
```

#### With SSL:

```bash
docker run --entrypoint lipstickd --name lipstickd --network host   -v /etc/letsencrypt:/etc/letsencrypt   --restart always -d jliotorresmoreno/lipstick   -p 8080   -m 8081   -k 123456   -cert /etc/letsencrypt/live/example.com/fullchain.pem   -key /etc/letsencrypt/live/example.com/privkey.pem
```

---

## Authentication

To ensure secure communication between the client and server, a secret key (`-k`) must be provided. This key is used to authenticate the client with the server. Make sure to use a strong, unique key and store it securely.

---

## Examples of Use Cases

1. **Expose a Development Server:** Use the Lipstick client to expose a local development server to a remote team for testing without deploying the application to production.

2. **Securely Share a Local Database:** Run a local database on your machine and use Lipstick to allow remote access for collaborators or applications.

3. **Forward Webhooks to Your Local Machine:** Use Lipstick to expose your local service for receiving webhooks from third-party services, even if you’re behind a NAT or firewall.

---

## Notes

- Lipstick is in an **experimental** phase and may not yet support all production scenarios.
- For enhanced security, it is recommended to integrate Lipstick with a reverse proxy like **Nginx** or **Traefik**.
- Ensure that the ports used by Lipstick are properly secured and monitored to prevent unauthorized access.


---

## Example Configuration File

Here is an example of a configuration file (`config.yml`) for Lipstick:

```yaml
admin_secret_key: "super_secret_key"
proxy:
  address: ":5050"
manager:
  address: ":5051"
admin:
  address: ":5052"
tls:
  certificate_path: "/path/to/cert.pem"
  key_path: "/path/to/key.pem"
database:
  host: "db.example.com"
  port: 5432
  user: "dbuser"
  password: "dbpassword"
  database: "app_db"
  ssl_mode: "disable"
redis:
  host: "localhost"
  port: 6379
  password: "redispassword"
  database: 1
```

This configuration file specifies the secret key, proxy, manager, and admin addresses, TLS certificate paths, database credentials, and Redis configuration.