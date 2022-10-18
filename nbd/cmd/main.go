// See https://github.com/NetworkBlockDevice/nbd/blob/master/doc/proto.md

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/plockc/disk8s/nbd"
	"github.com/plockc/disk8s/nbd/internal/store"
)

func usage() {
	for _, s := range []string{
		"Usage of " + os.Args[0] + ":",
		"A Network Block Device (NBD).",
		"Set environment variable REMOTE_STORAGE=host:port to connect via grpc.",
	} {
		fmt.Fprintf(flag.CommandLine.Output(), s)
	}
	flag.PrintDefaults()
}

func main() {
	killCtx, cancelKill := context.WithCancel(context.Background())
	go handleKill(killCtx)
	defer cancelKill()

	clientDevice := flag.String("client", "", "spawns an nbd client on given device path (e.g. /dev/nbd0) to connect to server over unix socket")
	tcp := flag.Bool("tcp", false, "use tcp for client and server, if no client, tcp is automatic")
	port := flag.Int("port", 10809, "port for TCP server on all interfaces")
	flag.Usage = usage

	remote := os.Getenv("REMOTE_STORAGE")

	ctx, cancel := context.WithCancel(context.Background())

	flag.Parse()

	var storage store.Storage
	if remote != "" {
		var err error
		storage, err = store.NewRemote(remote)
		if err != nil {
			fmt.Println("Failed to Set up Remote Store:", err)
			os.Exit(1)
		}
	} else {
		storage = store.NewMemory()
	}

	wg := sync.WaitGroup{}

	routines := []func() (string, error){}

	routines = append(routines, func() (string, error) {
		handleSignal(ctx, cancel)
		return "Interrupt handler", nil
	})

	// always run a server
	// check and register a start if we are running server in tcp mode
	if *clientDevice == "" || *tcp {
		routines = append(
			routines,
			func() (string, error) {
				return "TCP Server", nbd.NewTCPSocketServer(ctx, storage, *port)
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
					return "Domain Socket Server", nbd.NewDomainSocketServer(ctx, storage, domainSockets)
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

func handleSignal(ctx context.Context, cancel func()) {
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	select {
	case <-sigChan:
		fmt.Println("Received signal")
		cancel()
	case <-ctx.Done():
	}
}

func handleKill(ctx context.Context) {
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Kill)
	select {
	case <-sigChan:
		fmt.Println("Received Kill signal")
		os.Exit(1)
	case <-ctx.Done():
	}
}
