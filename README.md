
## Build
```
go build
```

## Usage
```
socks5-server [OPTIONS] address

Application Options:
      --dns=  custom dns. Example: 8.8.8.8:53

Help Options:
  -h, --help  Show this help message
```

## Example
```
./socks5-server 0.0.0.0:1080
./socks5-server localhost:1080
./socks5-server --dns=8.8.8.8:53 0.0.0.0:1080
```
