// See https://github.com/NetworkBlockDevice/nbd/blob/master/doc/proto.md

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/plockc/disk8s/nbd"
)

func main() {
	clientDevice := flag.String("client", "", "spawns an nbd client on given device path (e.g. /dev/nbd0) to connect to server over unix socket")
	tcp := flag.Bool("tcp", false, "use tcp for client and server, if no client, tcp is automatic")
	port := flag.Int("port", 10809, "port for TCP server on all interfaces")

	ctx, cancel := context.WithCancel(context.Background())

	flag.Parse()

	wg := sync.WaitGroup{}

	routines := []func() (string, error){}

	routines = append(routines, func() (string, error) {
		handleInterrupt(ctx, cancel)
		return "Interrupt handler", nil
	})

	// always run a server
	// check and register a start if we are running server in tcp mode
	if *clientDevice == "" || *tcp {
		routines = append(
			routines,
			func() (string, error) {
				return "TCP Server", nbd.NewTCPSocketServer(ctx, *port)
			},
		)
	}
	// check if we are running a client
	if *clientDevice != "" {
		// either we're running tcp client locally (server already registered to start)
		// or need domain sockets on both client and server
		if *tcp {
			routines = append(routines, func() (string, error) {
				// give the server a moment to come up
				time.Sleep(1 * time.Second)
				return "TCP Client", nbd.NewTcpClient(ctx, *clientDevice, *port)
			})
		} else {
			domainSockets := make(chan uintptr)

			routines = append(
				routines,
				func() (string, error) {
					return "Domain Socket Client", nbd.NewDomainSocketClient(ctx, *clientDevice, domainSockets)
				},
				func() (string, error) {
					return "Domain Socket Server", nbd.NewDomainSocketServer(domainSockets)
				},
			)
		}
	}

	for _, r := range routines {
		f := r // get copy off the heap as r changes and we're spawning usage of r
		wg.Add(1)
		go func() {
			defer wg.Done()
			label, err := f()
			if err != nil {
				fmt.Println(label, "exited with ERROR:", err)
			} else {
				fmt.Println(label, "exited cleanly")
			}
			cancel()
		}()
	}

	// wait for routines to complete before exiting
	wg.Wait()
}

func handleInterrupt(ctx context.Context, cancel func()) {
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt)
	select {
	case <-sigChan:
		fmt.Println("Received Interrupt signal")
		cancel()
	case <-ctx.Done():
	}
}
