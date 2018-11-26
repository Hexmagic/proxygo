package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/libp2p/go-reuseport"

	proxy "proxygo/proxy"
)

type EConn struct {
	net.Conn
}

var proxies []string
var proxiesMap = make(map[string]proxy.ProxyRef)

func checkError(err error) {
	if err != nil {
		fmt.Fprint(os.Stderr, "%v", err)
	}
}

func connectSocks5(addr string, domain string, port uint16) (net.Conn, error) {
	c2, err := net.Dial("tcp", addr)
	fmt.Println(addr)
	if err != nil {
		log.Println("Conn.Dial failed:", err, addr)
		return nil, err
	}
	c2.SetDeadline(time.Now().Add(10 * time.Second))
	c2.Write([]byte{5, 2, 0, 0x81})
	resp := make([]byte, 2)
	n, err := c2.Read(resp)
	if err != nil {
		log.Println("Conn.Read failed:", err)
		return nil, err
	}
	if n != 2 {
		log.Println("socks5 response error:", resp)
		return nil, errors.New("socks5_error")
	}
	method := resp[1]
	if method != 0 && method != 0x81 {
		log.Println("socks5 not support 'NO AUTHENTICATION REQUIRED'")
		return nil, errors.New("socks5_error")
	}
	send := make([]byte, 0, 512)
	send = append(send, []byte{5, 1, 0, 3, byte(len(domain))}...)
	if method == 0 {
		send = append(send, []byte(domain)...)
	} else {
		edomain := []byte(domain)
		for i, c := range edomain {
			edomain[i] = ^c
		}
		send = append(send, edomain...)
	}
	send = append(send, byte(port>>8))
	send = append(send, byte(port&0xff))
	_, err = c2.Write(send)
	if err != nil {
		log.Println("Conn.Write failed:", err)
		return nil, err
	}
	n, err = c2.Read(send[0:10])
	if err != nil {
		log.Println("Conn.Read failed:", err)
		return nil, err
	}
	if send[1] != 0 {
		switch send[1] {
		case 1:
			log.Println("socks5 general SOCKS server failure")
		case 2:
			log.Println("socks5 connection not allowed by ruleset")
		case 3:
			log.Println("socks5 Network unreachable")
		case 4:
			log.Println("socks5 Host unreachable")
		case 5:
			log.Println("socks5 Connection refused")
		case 6:
			log.Println("socks5 TTL expired")
		case 7:
			log.Println("socks5 Command not supported")
		case 8:
			log.Println("socks5 Address type not supported")
		default:
			log.Println("socks5 Unknown eerror:", send[1])
		}
		return nil, errors.New("socks5_error")
	}
	c2.SetDeadline(time.Time{})
	if method == 0 {
		return c2, nil
	} else {
		return &EConn{c2}, nil
	}
}

func getProxyByDomain(domain string, port int) (net.Conn, string, error) {
	address := proxy.FindBestProxy(proxiesMap, proxies, domain)
	proxy.IncRef(proxiesMap, domain, address)
	fmt.Printf("domain %s and address %s\n", domain, address)
	peer, err := connectSocks5(address, domain, uint16(port))
	return peer, address, err
}

func buildConnect(con net.Conn, header []string) (peer net.Conn, domain, address string, err error) {
	head := header[0]
	uri := strings.Split(head, " ")[1]
	parsed, err := url.Parse(uri)
	checkError(err)
	var port string
	if strings.Contains(parsed.Host, ":") {
		lst := strings.Split(parsed.Host, ":")
		domain, port = lst[0], lst[1]
	} else {
		domain = parsed.Scheme
		port = parsed.Opaque
	}
	_port, err := strconv.Atoi(port)
	checkError(err)
	peer, address, err = getProxyByDomain(domain, _port)
	checkError(err)
	_, err = con.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))
	if err != nil {
		peer.Close()
		log.Println("write response failed", err)
	}
	return
}

func HandleProxy(client net.Conn) {
	defer func() { // 必须要先声明defer，否则不能捕获到panic异常
		fmt.Println("c")
		if err := recover(); err != nil {
			fmt.Println(err) // 这里的err其实就是panic传入的内容，55
		}
		fmt.Println("d")
	}()
	var buff [1024]byte
	var peer net.Conn
	_, err := client.Read(buff[:])
	checkError(err)
	peer, domain, address, err := getHttpProxy(client, buff[:])
	checkError(err)
	go func() {
		defer peer.Close()
		defer client.Close()
		proxy.DecRef(proxiesMap, domain, address)
		io.Copy(client, peer)
	}()
	io.Copy(peer, client)
}

func getHttpProxy(con net.Conn, buff []byte) (peer net.Conn, domain, address string, err error) {
	header := string(buff)
	headerList := strings.Split(strings.TrimSpace(header), "\r\n")
	methodAndUrlList := strings.Split(headerList[0], " ")
	method := methodAndUrlList[0]
	if method == "CONNECT" {
		peer, domain, address, err = buildConnect(con, headerList)
	} else {
		/*peer, err = buildHttpProxy(con, headerList)*/
		fmt.Println("okkkk")
	}
	return
}

func main() {
	configs := proxy.Load("config.json")
	for _, v := range configs.Configs {
		proxies = append(proxies, fmt.Sprintf("%s:%d", v.LocalAddr, v.LocalPort))
	}

	socket, err := reuseport.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", configs.LocalPort))
	if err != nil {
		fmt.Fprintf(os.Stdout, "绑定端口%d失败", configs.LocalPort)
		os.Exit(0)
	} else {
		fmt.Printf("Listen on %d\n", configs.LocalPort)
	}
	time.Sleep(2 * 1e9)
	for {
		client, err := socket.Accept()
		if err != nil {
			fmt.Fprintf(os.Stdout, "连接时发生了错误 %v", err)
		}
		go HandleProxy(client)
	}

}
