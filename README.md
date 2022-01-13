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

### API usage
```
// create terminal
    fetch('/term', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ 
            'host': '127.0.0.1',
            'port': '22',
            'user': 'root',
            'pwd': '123'})
        })
        .then(resp => resp.json())
        .then(term => {
            console.log(term);
            const terminal = new Terminal();
            terminal.loadAddon(new AttachAddon.AttachAddon(
                new WebSocket(`ws://${window.location.host}/term/${term.id}/data`), { bidirectional: true }));
            terminal.open(document.getElementById('terminal'));
            terminal.focus();
        })
        .catch(err => console.error(err));

// change terminal size
    fetch(`/term/${term.id}/windowsize`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                'cols': 80,
                'rows': 40})
        })
        .then(resp => resp.json())
        .then(term => console.log(term))
        .catch(err => console.error(err));
```
