# MiniWebSSHServer
A mini Web SSH server.

### Build
```
go build .
```

### Run
```
miniwebsshserver -bind <ip_addr:port>
```

### Open a term from url
```
http://<ip_addr:port>/term?host=127.0.0.1&port=22&user=root&pwd=123
```
