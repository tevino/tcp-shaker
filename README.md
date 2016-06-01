# TCP Shaker :heartbeat:
[![GoDoc](https://godoc.org/github.com/tevino/tcp-shaker?status.svg)](https://godoc.org/github.com/tevino/tcp-shaker)

Performing TCP handshake without ACK, useful for health checking.

HAProxy do this exactly the same, which is:

- SYN
- SYN-ACK
- RST

## Why do I have to do this?
Usually when you establish a TCP connection(e.g. net.Dial), these are the first three packets (TCP three-way handshake):

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

Usually this means the server will not notice the health checking traffic at all, **thus the act of health chekcing will not be
considered as some misbehaviour of client.**

## Requirements:
- Linux 2.4 or newer

## TODO:

- [ ] IPv6 support (Test environment needed, PRs are welcomed)
