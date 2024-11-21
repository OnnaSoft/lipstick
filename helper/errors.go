package helper

import "fmt"

var BadGatewayHeader = `HTTP/1.1 502 Bad Gateway
Content-Type: text/html
Content-Length: `

var BadGatewayBody = `<!DOCTYPE html>
<html>
<head>
    <title>502 Bad Gateway</title>
</head>
<body>
    <h1>Bad Gateway</h1>
    <p>The server encountered a temporary error and could not complete your request.</p>
</body>
</html>`

var BadGatewayResponse = BadGatewayHeader + fmt.Sprint(len(BadGatewayBody)) + "\n\n" + BadGatewayBody
