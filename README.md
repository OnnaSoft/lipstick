
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

## Environment Variables

Lipstick supports environment variables for configuration. These variables can be used as an alternative to the configuration file or command-line arguments:

### General Configuration

| Variable          | Description                                         | Default         |
|--------------------|-----------------------------------------------------|-----------------|
| `ADMIN_ADDR`       | Address for the admin API                          | `:5052`         |
| `MANAGER_ADDR`     | Address for WebSocket manager connections          | `:5051`         |
| `PROXY_ADDR`       | Address for the proxy                              | `:5050`         |
| `ADMIN_SECRET_KEY` | Secret key for admin API authorization             | `""`            |

### TLS Configuration

| Variable         | Description                  | Default       |
|-------------------|------------------------------|---------------|
| `TLS_CERT`        | Path to the TLS certificate | `""`          |
| `TLS_KEY`         | Path to the TLS key         | `""`          |

### Redis Configuration

| Variable            | Description                                         | Default   |
|----------------------|-----------------------------------------------------|-----------|
| `REDIS_HOST`         | Redis host address                                 | `localhost` |
| `REDIS_PORT`         | Redis port                                         | `6379`    |
| `REDIS_PASSWORD`     | Redis password (leave empty if not set)            | `""`      |
| `REDIS_DB`           | Redis database index                               | `0`       |
| `REDIS_POOL_SIZE`    | Maximum number of connections in the Redis pool    | `10`      |
| `REDIS_MIN_IDLE_CONNS` | Minimum number of idle connections in the pool    | `3`       |
| `REDIS_POOL_TIMEOUT` | Timeout (in seconds) for getting a connection      | `30`      |

### Database Configuration

| Variable        | Description                    | Default      |
|------------------|--------------------------------|--------------|
| `DB_HOST`        | Database host address         | `localhost`  |
| `DB_PORT`        | Database port                 | `5432`       |
| `DB_USER`        | Database user                 | `postgres`   |
| `DB_PASSWORD`    | Database password             | `""`         |
| `DB_NAME`        | Database name                 | `app_db`     |
| `DB_SSL_MODE`    | Database SSL mode             | `disable`    |

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
  pool_size: 20
  min_idle_conns: 5
  pool_timeout: 30
```

---

## Notes

- Lipstick is in an **experimental** phase and may not yet support all production scenarios.
- For enhanced security, it is recommended to integrate Lipstick with a reverse proxy like **Nginx** or **Traefik**.
- Ensure that the ports used by Lipstick are properly secured and monitored to prevent unauthorized access.