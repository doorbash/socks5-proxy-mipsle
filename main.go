package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/nadoo/glider/proxy"
	"github.com/nadoo/glider/proxy/socks5"
)

var opts Options
var r *net.Resolver

type Options struct {
	Dns string `long:"dns" description:"custom dns. Example: 8.8.8.8:53" required:"false"`
}

type UdpConn struct {
	net.PacketConn
	addr *net.UDPAddr
}

func (u UdpConn) Read(b []byte) (int, error) {
	n, _, err := u.ReadFrom(b)
	return n, err
}

func (u UdpConn) Write(b []byte) (int, error) {
	return u.WriteTo(b, u.RemoteAddr())
}

func (u UdpConn) RemoteAddr() net.Addr {
	return u.addr
}

type UdpPktConn struct {
	net.Conn
	addr *net.UDPAddr
}

func (p UdpPktConn) ReadFrom(b []byte) (int, net.Addr, error) {
	n, err := p.Read(b)
	return n, p.addr, err
}

func (p UdpPktConn) WriteTo(b []byte, addr net.Addr) (int, error) {
	return p.Write(b)
}

type ProxyDialer struct {
	dialer *net.Dialer
}

func (p *ProxyDialer) Dial(network, addr string) (c net.Conn, err error) {
	log.Printf("Dial: network: %s, addr: %s\n", network, addr)

	colonIndex := strings.LastIndex(addr, ":")

	if colonIndex == -1 {
		return nil, errors.New("bad address")
	}

	address := addr[:colonIndex]
	port := addr[colonIndex+1:]
	ip := net.ParseIP(address)

	if ip == nil {
		ctx := context.Background()
		ips, err := r.LookupIP(ctx, "ip", address)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		ip = ips[0]
	}

	ipv4 := ip.To4()

	if ipv4 != nil {
		ip = ipv4
	}

	_, err = strconv.Atoi(port)

	if err != nil {
		return nil, err
	}

	conn, err := p.dialer.Dial(network, fmt.Sprintf("%s:%s", ip.String(), port))
	return conn, err
}

func (p *ProxyDialer) DialUDP(network, addr string) (pc net.PacketConn, writeTo net.Addr, err error) {
	log.Printf("DialUDP: network: %s, addr: %s\n", network, addr)

	colonIndex := strings.LastIndex(addr, ":")

	if colonIndex == -1 {
		return nil, nil, errors.New("bad address")
	}

	address := addr[:colonIndex]
	port := addr[colonIndex+1:]
	ip := net.ParseIP(address)

	if ip == nil {
		ctx := context.Background()
		ips, err := r.LookupIP(ctx, "ip", address)
		if err != nil {
			return nil, nil, err
		}
		ip = ips[0]
	}

	ipv4 := ip.To4()

	if ipv4 != nil {
		ip = ipv4
	}

	prt, err := strconv.Atoi(port)

	if err != nil {
		return nil, nil, err
	}

	conn, err := p.dialer.Dial(network, fmt.Sprintf("%s:%s", ip.String(), port))

	uaddr := &net.UDPAddr{
		IP:   ip,
		Port: prt,
	}

	return UdpPktConn{conn, uaddr}, uaddr, err
}

func (p *ProxyDialer) Addr() string {
	return ""
}

type SSRProxy struct {
	dialer *ProxyDialer
}

func (p SSRProxy) Dial(network, addr string) (net.Conn, proxy.Dialer, error) {
	conn, err := p.dialer.Dial(network, addr)
	return conn, p.dialer, err
}

func (p SSRProxy) DialUDP(network, addr string) (net.PacketConn, proxy.UDPDialer, net.Addr, error) {
	conn, ad, err := p.dialer.DialUDP(network, addr)
	return conn, p.dialer, ad, err
}

func (p SSRProxy) NextDialer(dstAddr string) proxy.Dialer {
	log.Printf("NextDialer: dstAddr: %s\n", dstAddr)
	return p.dialer
}

func (p SSRProxy) Record(dialer proxy.Dialer, success bool) {
	// log.Printf("Record: success: %v\n", success)
}

func main() {
	parser := flags.NewParser(&opts, flags.Default)

	parser.Usage = "[OPTIONS] address"

	args, err := parser.Parse()

	if err != nil {
		return
	}

	if len(args) == 0 {
		parser.WriteHelp(os.Stdout)
		return
	}

	proxy := SSRProxy{
		dialer: &ProxyDialer{
			dialer: &net.Dialer{
				Timeout: time.Millisecond * time.Duration(10000),
			},
		},
	}

	if opts.Dns == "" {
		r = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				pk, addr, err := proxy.dialer.DialUDP(network, address)
				return UdpConn{pk, addr.(*net.UDPAddr)}, err
			},
		}
	} else {
		c := strings.LastIndex(opts.Dns, ":")
		if c == -1 {
			log.Fatalln("bad dns address")
		}
		dnsAddr := opts.Dns[:c]
		dp := opts.Dns[c+1:]
		_, err := strconv.Atoi(dp)

		dnsIp := net.ParseIP(dnsAddr)

		if dnsIp == nil {
			log.Fatalln("bad dns ip")
		}

		if err != nil {
			log.Fatalln("bad dns port")
		}

		r = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				pk, addr, err := proxy.dialer.DialUDP(network, opts.Dns)
				return UdpConn{pk, addr.(*net.UDPAddr)}, err
			},
		}
	}

	server, _ := socks5.NewSocks5Server(fmt.Sprintf("socks://%s", args[0]), proxy)

	server.ListenAndServe()

}
