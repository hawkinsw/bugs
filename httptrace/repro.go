package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"sync"

	"golang.org/x/net/http2"
)

type TracedHttp2Client struct {
	client   *http.Client
	clientId uint64
	tracer   *httptrace.ClientTrace
}

func main() {
	demonstrateBug := false
	flag.BoolVar(&demonstrateBug, "bug", false, "If set, the program will demonstrate the bug. Otherwise, the program will demonstrate expected behavior.")
	flag.Parse()

	var client *http.Client = nil

	if demonstrateBug {
		fmt.Printf("Demonstrating buggy behavior.\n")
		// All httptrace methods will be invoked during the subsequent request *except*
		// TLSHandshakeStart and TLSHandshakeDone.

		// To force net/http to give me a new TCP connection for this,
		// client, I must set a transport. Because I definitely want HTTP/2,
		// I will use a x/net/http2 transport.
		transport := http2.Transport{}
		// To mimic my actual setup, I will set a TLSConfig. In my actual
		// code, I (conditionally) set a KeyLogWriter, but that does not
		// change the ultimate outcome of this code!
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		client = &http.Client{Transport: &transport}
	} else {
		fmt.Printf("Demonstrating expected behavior.\n")
		// All httptrace methods will be invoked during the subsequent request *except*
		// TLSHandshakeStart and TLSHandshakeDone.
		client = &http.Client{}
	}

	tracer := &httptrace.ClientTrace{
		DNSStart: func(dnsStartInfo httptrace.DNSStartInfo) {
			fmt.Printf(
				"DNS Start: %v\n",
				dnsStartInfo,
			)
		},
		DNSDone: func(dnsDoneInfo httptrace.DNSDoneInfo) {
			fmt.Printf("DNS Done: %v\n", dnsDoneInfo)
		},
		ConnectStart: func(network, address string) {
			fmt.Printf("ConnectStart: network: %v; address: %v\n", network, address)
		},
		ConnectDone: func(network, address string, err error) {
			fmt.Printf(
				"ConnectDone: %v: %v (%v)\n",
				network,
				address,
				err,
			)
		},
		GetConn: func(hostPort string) {
			fmt.Printf(
				"GetConn host port: %v\n",
				hostPort,
			)
		},
		GotConn: func(connInfo httptrace.GotConnInfo) {
			fmt.Printf(
				"GetConn host port: %v\n",
				connInfo,
			)
		},
		TLSHandshakeStart: func() {
			fmt.Printf("TLSHandshakeStart.\n")
		},
		TLSHandshakeDone: func(tlsConnState tls.ConnectionState, err error) {
			fmt.Printf(
				"TLSHandshakeDone: %v\n",
				tlsConnState,
			)
		},
	}

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		req, err := http.NewRequestWithContext(httptrace.WithClientTrace(context.Background(), tracer), "GET", "https://www.google.com", nil)
		if err != nil {
			fmt.Printf("Could not initialize NewRequestWithContext(): %v\n", err)
			wg.Done()
			return
		}

		get, err := client.Do(req)

		if err == nil {
			// Stringify `get` so that we can verify it is using HTTP2 in both cases.
			fmt.Printf("Response header: \n")
			fmt.Printf("%v\n", get)
			fmt.Printf("\n")
			get.Body.Close()
		} else {
			fmt.Printf("Failure: %v\n", err)
		}
		wg.Done()
	}()

	fmt.Printf("The waiting is the hardest part.\n")
	wg.Wait()

	fmt.Printf("All done!\n")
}
