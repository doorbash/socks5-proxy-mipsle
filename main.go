package main

import (
	"context"
	"log"
	"net"
	"os"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/things-go/go-socks5"
)

type Options struct {
	Dns string `long:"dns" description:"custom dns. Example: 8.8.8.8:53" required:"false"`
}
type DNSResolver struct{}

var opts Options
var server *socks5.Server
var r *net.Resolver

func (d DNSResolver) Resolve(ctx context.Context, name string) (context.Context, net.IP, error) {
	addr, err := r.LookupIP(ctx, "ip", name)
	if err != nil {
		return ctx, nil, err
	}
	return ctx, addr[0], err
}

func main() {
	parser := flags.NewParser(&opts, flags.Default)

	_, err := parser.Parse()

	if err != nil {
		return
	}

	if opts.Dns == "" {
		server = socks5.NewServer(
			socks5.WithLogger(socks5.NewLogger(log.New(os.Stdout, "socks5: ", log.LstdFlags))),
		)
	} else {
		r = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: time.Millisecond * time.Duration(10000),
				}
				return d.DialContext(ctx, network, opts.Dns)
			},
		}
		server = socks5.NewServer(
			socks5.WithResolver(DNSResolver{}),
			socks5.WithLogger(socks5.NewLogger(log.New(os.Stdout, "socks5: ", log.LstdFlags))),
		)
	}

	if err := server.ListenAndServe("tcp", ":10800"); err != nil {
		panic(err)
	}
}
