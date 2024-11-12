<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Lipstick Documentation</title>
</head>
<body>
    <h1>Lipstick Documentation</h1>
    <p><strong>Lipstick</strong> is a tool that allows you to share local ports with remote servers immediately, without the need for a VPN. It is currently in an experimental phase and requires authentication. For added security, you can integrate it with a reverse proxy like Nginx.</p>

    <h2>Overview</h2>
    <p><strong>Lipstick</strong> provides a way to expose services running on your local machine to remote servers securely. Below is an explanation of its components and their configurations.</p>

    <h3>Server</h3>
    <p>The <strong>server</strong> refers to the remote host where your local ports are exposed. It is responsible for managing client connections and forwarding traffic to the appropriate destinations.</p>

    <h4>How to Use</h4>
    <pre>
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
    </pre>

    <h3>Client</h3>
    <p>The <strong>client</strong> is the local machine running the service you want to expose. For example, this could be your local web server or any application listening on a specific port.</p>

    <h4>How to Use</h4>
    <pre>
Usage of lipstick-client:
  -c string
    	Config path (default "/etc/lipstick/config.client.yml")
  -p string
    	Host/port where you want to connect from the remote server. (default "127.0.0.1:8082")
  -s string
    	WebSocket URL of the server manager where your client will connect. (default "ws://localhost:8081/ws")
  -k string
        Secret key to authenticate with the server. (default "super_secret_key")
    </pre>

    <h2>Running Lipstick in Docker</h2>

    <h3>Run Client in Docker</h3>
    <p>You can run the Lipstick client in a Docker container to expose a local service securely:</p>

    <h4>Without SSL:</h4>
    <pre>
docker run --entrypoint lipstick --name lipstick --network host \
  --restart always -dt jliotorresmoreno/lipstick \
  -s ws://127.0.0.1:5051/ws \
  -p 127.0.0.1:8082 \
  -k 123456
    </pre>

    <h4>With SSL:</h4>
    <pre>
docker run --entrypoint lipstick --name lipstick --network host \
  --restart always -dt jliotorresmoreno/lipstick \
  -s wss://example.com/ws \
  -p 127.0.0.1:8082 \
  -k 123456
    </pre>

    <h3>Run Server in Docker</h3>
    <p>To run the Lipstick server in a Docker container:</p>

    <h4>Without SSL:</h4>
    <pre>
docker run --name lipstickd --network host --restart always -d \
  jliotorresmoreno/lipstick \
  -p 8080 \
  -m 8081 \
  -k 123456
    </pre>

    <h4>With SSL:</h4>
    <pre>
docker run --entrypoint lipstickd --name lipstickd --network host \
  -v /etc/letsencrypt:/etc/letsencrypt \
  --restart always -d jliotorresmoreno/lipstick \
  -p 8080 \
  -m 8081 \
  -k 123456 \
  -cert /etc/letsencrypt/live/example.com/fullchain.pem \
  -key /etc/letsencrypt/live/example.com/privkey.pem
    </pre>

    <h2>Authentication</h2>
    <p>To ensure secure communication between the client and server, a secret key (<code>-k</code>) must be provided. This key is used to authenticate the client with the server. Make sure to use a strong, unique key and store it securely.</p>

    <h2>Examples of Use Cases</h2>
    <ul>
        <li><strong>Expose a Development Server:</strong> Use the Lipstick client to expose a local development server to a remote team for testing without deploying the application to production.</li>
        <li><strong>Securely Share a Local Database:</strong> Run a local database on your machine and use Lipstick to allow remote access for collaborators or applications.</li>
        <li><strong>Forward Webhooks to Your Local Machine:</strong> Use Lipstick to expose your local service for receiving webhooks from third-party services, even if you’re behind a NAT or firewall.</li>
    </ul>

    <h2>Notes</h2>
    <ul>
        <li>Lipstick is in an <strong>experimental</strong> phase and may not yet support all production scenarios.</li>
        <li>For enhanced security, it is recommended to integrate Lipstick with a reverse proxy like <strong>Nginx</strong> or <strong>Traefik</strong>.</li>
        <li>Ensure that the ports used by Lipstick are properly secured and monitored to prevent unauthorized access.</li>
    </ul>
</body>
</html>
