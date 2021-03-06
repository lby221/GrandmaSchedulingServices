// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package websocket

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ErrBadHandshake is returned when the server response to opening handshake is
// invalid.
var ErrBadHandshake = errors.New("websocket: bad handshake")

// NewClient creates a new client connection using the given net connection.
// The URL u specifies the host and request URI. Use requestHeader to specify
// the origin (Origin), subprotocols (Sec-WebSocket-Protocol) and cookies
// (Cookie). Use the response.Header to get the selected subprotocol
// (Sec-WebSocket-Protocol) and cookies (Set-Cookie).
//
// If the WebSocket handshake fails, ErrBadHandshake is returned along with a
// non-nil *http.Response so that callers can handle redirects, authentication,
// etc.
//
// Deprecated: Use Dialer instead.
func NewClient(netConn net.Conn, u *url.URL, requestHeader http.Header, readBufSize, writeBufSize int) (c *Conn, response *http.Response, err error) {
	d := Dialer{
		ReadBufferSize:  readBufSize,
		WriteBufferSize: writeBufSize,
		NetDial: func(net, addr string) (net.Conn, error) {
			return netConn, nil
		},
	}
	return d.Dial(u.String(), requestHeader)
}

// A Dialer contains options for connecting to WebSocket server.
type Dialer struct {
	// NetDial specifies the dial function for creating TCP connections. If
	// NetDial is nil, net.Dial is used.
	NetDial func(network, addr string) (net.Conn, error)

	// Proxy specifies a function to return a proxy for a given
	// Request. If the function returns a non-nil error, the
	// request is aborted with the provided error.
	// If Proxy is nil or returns a nil *URL, no proxy is used.
	Proxy func(*http.Request) (*url.URL, error)

	// TLSClientConfig specifies the TLS configuration to use with tls.Client.
	// If nil, the default configuration is used.
	TLSClientConfig *tls.Config

	// HandshakeTimeout specifies the duration for the handshake to complete.
	HandshakeTimeout time.Duration

	// Input and output buffer sizes. If the buffer size is zero, then a
	// default value of 4096 is used.
	ReadBufferSize, WriteBufferSize int

	// Subprotocols specifies the client's requested subprotocols.
	Subprotocols []string
}

var errMalformedURL = errors.New("malformed ws or wss URL")

// parseURL parses the URL.
//
// This function is a replacement for the standard library url.Parse function.
// In Go 1.4 and earlier, url.Parse loses information from the path.
func parseURL(s string) (*url.URL, error) {
	// From the RFC:
	//
	// ws-URI = "ws:" "//" host [ ":" port ] path [ "?" query ]
	// wss-URI = "wss:" "//" host [ ":" port ] path [ "?" query ]

	var u url.URL
	switch {
	case strings.HasPrefix(s, "ws://"):
		u.Scheme = "ws"
		s = s[len("ws://"):]
	case strings.HasPrefix(s, "wss://"):
		u.Scheme = "wss"
		s = s[len("wss://"):]
	default:
		return nil, errMalformedURL
	}

	if i := strings.Index(s, "?"); i >= 0 {
		u.RawQuery = s[i+1:]
		s = s[:i]
	}

	if i := strings.Index(s, "/"); i >= 0 {
		u.Opaque = s[i:]
		s = s[:i]
	} else {
		u.Opaque = "/"
	}

	u.Host = s

	if strings.Contains(u.Host, "@") {
		// Don't bother parsing user information because user information is
		// not allowed in websocket URIs.
		return nil, errMalformedURL
	}

	return &u, nil
}

func hostPortNoPort(u *url.URL) (hostPort, hostNoPort string) {
	hostPort = u.Host
	hostNoPort = u.Host
	if i := strings.LastIndex(u.Host, ":"); i > strings.LastIndex(u.Host, "]") {
		hostNoPort = hostNoPort[:i]
	} else {
		switch u.Scheme {
		case "wss":
			hostPort += ":443"
		case "https":
			hostPort += ":443"
		default:
			hostPort += ":80"
		}
	}
	return hostPort, hostNoPort
}

// DefaultDialer is a dialer with all fields set to the default zero values.
var DefaultDialer = &Dialer{
	Proxy: http.ProxyFromEnvironment,
}

// Dial creates a new client connection. Use requestHeader to specify the
// origin (Origin), subprotocols (Sec-WebSocket-Protocol) and cookies (Cookie).
// Use the response.Header to get the selected subprotocol
// (Sec-WebSocket-Protocol) and cookies (Set-Cookie).
//
// If the WebSocket handshake fails, ErrBadHandshake is returned along with a
// non-nil *http.Response so that callers can handle redirects, authentication,
// etcetera. The response body may not contain the entire response and does not
// need to be closed by the application.
func (d *Dialer) Dial(urlStr string, requestHeader http.Header) (*Conn, *http.Response, error) {

	if d == nil {
		d = &Dialer{
			Proxy: http.ProxyFromEnvironment,
		}
	}

	challengeKey, err := generateChallengeKey()
	if err != nil {
		return nil, nil, err
	}

	u, err := parseURL(urlStr)
	if err != nil {
		return nil, nil, err
	}

	switch u.Scheme {
	case "ws":
		u.Scheme = "http"
	case "wss":
		u.Scheme = "https"
	default:
		return nil, nil, errMalformedURL
	}

	if u.User != nil {
		// User name and password are not allowed in websocket URIs.
		return nil, nil, errMalformedURL
	}

	req := &http.Request{
		Method:     "GET",
		URL:        u,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Host:       u.Host,
	}

	// Set the request headers using the capitalization for names and values in
	// RFC examples. Although the capitalization shouldn't matter, there are
	// servers that depend on it. The Header.Set method is not used because the
	// method canonicalizes the header names.
	req.Header["Upgrade"] = []string{"websocket"}
	req.Header["Connection"] = []string{"Upgrade"}
	req.Header["Sec-WebSocket-Key"] = []string{challengeKey}
	req.Header["Sec-WebSocket-Version"] = []string{"13"}
	if len(d.Subprotocols) > 0 {
		req.Header["Sec-WebSocket-Protocol"] = []string{strings.Join(d.Subprotocols, ", ")}
	}
	for k, vs := range requestHeader {
		switch {
		case k == "Host":
			if len(vs) > 0 {
				req.Host = vs[0]
			}
		case k == "Upgrade" ||
			k == "Connection" ||
			k == "Sec-Websocket-Key" ||
			k == "Sec-Websocket-Version" ||
			(k == "Sec-Websocket-Protocol" && len(d.Subprotocols) > 0):
			return nil, nil, errors.New("websocket: duplicate header not allowed: " + k)
		default:
			req.Header[k] = vs
		}
	}

	hostPort, hostNoPort := hostPortNoPort(u)

	var proxyURL *url.URL
	// Check wether the proxy method has been configured
	if d.Proxy != nil {
		proxyURL, err = d.Proxy(req)
	}
	if err != nil {
		return nil, nil, err
	}

	var targetHostPort string
	if proxyURL != nil {
		targetHostPort, _ = hostPortNoPort(proxyURL)
	} else {
		targetHostPort = hostPort
	}

	var deadline time.Time
	if d.HandshakeTimeout != 0 {
		deadline = time.Now().Add(d.HandshakeTimeout)
	}

	netDial := d.NetDial
	if netDial == nil {
		netDialer := &net.Dialer{Deadline: deadline}
		netDial = netDialer.Dial
	}

	netConn, err := netDial("tcp", targetHostPort)
	if err != nil {
		return nil, nil, err
	}

	defer func() {
		if netConn != nil {
			netConn.Close()
		}
	}()

	if err := netConn.SetDeadline(deadline); err != nil {
		return nil, nil, err
	}

	if proxyURL != nil {
		connectReq := &http.Request{
			Method: "CONNECT",
			URL:    &url.URL{Opaque: hostPort},
			Host:   hostPort,
			Header: make(http.Header),
		}

		connectReq.Write(netConn)

		// Read response.
		// Okay to use and discard buffered reader here, because
		// TLS server will not speak until spoken to.
		br := bufio.NewReader(netConn)
		resp, err := http.ReadResponse(br, connectReq)
		if err != nil {
			return nil, nil, err
		}
		if resp.StatusCode != 200 {
			f := strings.SplitN(resp.Status, " ", 2)
			return nil, nil, errors.New(f[1])
		}
	}

	if u.Scheme == "https" {
		cfg := d.TLSClientConfig
		if cfg == nil {
			cfg = &tls.Config{ServerName: hostNoPort}
		} else if cfg.ServerName == "" {
			shallowCopy := *cfg
			cfg = &shallowCopy
			cfg.ServerName = hostNoPort
		}
		tlsConn := tls.Client(netConn, cfg)
		netConn = tlsConn
		if err := tlsConn.Handshake(); err != nil {
			return nil, nil, err
		}
		if !cfg.InsecureSkipVerify {
			if err := tlsConn.VerifyHostname(cfg.ServerName); err != nil {
				return nil, nil, err
			}
		}
	}

	conn := newConn(netConn, false, d.ReadBufferSize, d.WriteBufferSize)

	if err := req.Write(netConn); err != nil {
		return nil, nil, err
	}

	resp, err := http.ReadResponse(conn.br, req)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != 101 ||
		!strings.EqualFold(resp.Header.Get("Upgrade"), "websocket") ||
		!strings.EqualFold(resp.Header.Get("Connection"), "upgrade") ||
		resp.Header.Get("Sec-Websocket-Accept") != computeAcceptKey(challengeKey) {
		// Before closing the network connection on return from this
		// function, slurp up some of the response to aid application
		// debugging.
		buf := make([]byte, 1024)
		n, _ := io.ReadFull(resp.Body, buf)
		resp.Body = ioutil.NopCloser(bytes.NewReader(buf[:n]))
		return nil, resp, ErrBadHandshake
	}

	resp.Body = ioutil.NopCloser(bytes.NewReader([]byte{}))
	conn.subprotocol = resp.Header.Get("Sec-Websocket-Protocol")

	netConn.SetDeadline(time.Time{})
	netConn = nil // to avoid close in defer.
	return conn, resp, nil
}
