
## Build
```
GOOS=linux GOARCH=mipsle GOMIPS=softfloat go build -trimpath -ldflags="-s -w" -o socks5-proxy-mipsle
```

## Usage
```
  socks5-proxy-mipsle [OPTIONS] address

Application Options:
      --dns=  custom dns. Example: 8.8.8.8:53

Help Options:
  -h, --help  Show this help message
```

## Example
```
./socks5-proxy-mipsle :1080
./socks5-proxy-mipsle localhost:1080
./socks5-proxy-mipsle --dns=8.8.8.8:53 :1080
```