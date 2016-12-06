# TCP Checker :heartbeat:
[![Go Report Card](https://goreportcard.com/badge/github.com/tevino/tcp-shaker)](https://goreportcard.com/report/github.com/tevino/tcp-shaker)
[![GoDoc](https://godoc.org/github.com/tevino/tcp-shaker?status.svg)](https://godoc.org/github.com/tevino/tcp-shaker)

Performing TCP handshake without ACK, useful for health checking.

HAProxy does this exactly the same, which is:

- SYN
- SYN-ACK
- RST

## Why do I have to do this?
Usually when you establish a TCP connection(e.g. `net.Dial`), these are the first three packets (TCP three-way handshake):

- Client -> Server: SYN
- Server -> Client: SYN-ACK
- Client -> Server: ACK

**This package tries to avoid the last ACK when doing handshakes.**

By sending the last ACK, the connection is considered established.

However as for TCP health checking the last ACK may not necessary.

The Server could be considered alive after it sends back SYN-ACK.

### Benefits of avoiding the last ACK:
1. Less packets better efficiency
2. The health checking is less obvious

The second one is essential, because it bothers server less.

Usually this means the server will not notice the health checking traffic at all, **thus the act of health checking will not be
considered as some misbehaviour of client.**

## Requirements:
- Linux 2.4 or newer

There is a **fake implementation** for **non-Linux** platform which is equivalent to:
```go
conn, err := net.DialTimeout("tcp", addr, timeout)
conn.Close()
```

## Usage
```go
	import "github.com/tevino/tcp-shaker"

	c := tcp.NewChecker(true)
	if err := c.InitChecker(); err != nil {
		log.Fatal("Checker init failed:", err)
	}

	timeout := time.Second * 1
	err := c.CheckAddr("google.com:80", timeout)
	switch err {
	case tcp.ErrTimeout:
		fmt.Println("Connect to Google timed out")
	case nil:
		fmt.Println("Connect to Google succeeded")
	default:
		if e, ok := err.(*tcp.ErrConnect); ok {
			fmt.Println("Connect to Google failed:", e)
		} else {
			fmt.Println("Error occurred while connecting:", err)
		}
	}
```

## TODO:

- [ ] IPv6 support (Test environment needed, PRs are welcome)
