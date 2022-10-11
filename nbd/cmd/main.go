// See https://github.com/NetworkBlockDevice/nbd/blob/master/doc/proto.md

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"

	"github.com/plockc/disk8s/nbd"
)

func main() {
	clientDevice := flag.String("local", "", "spawns an nbd client on given device path (e.g. /dev/nbd0) to connect to server over unix socket")
	port := flag.Uint("port", 10809, "port for TCP server on all interfaces")

	ctx, cancel := context.WithCancel(context.Background())

	flag.Parse()

	wg := sync.WaitGroup{}

	routines := []func() (string, error){}

	routines = append(routines, func() (string, error) {
		handleInterrupt(ctx, cancel)
		return "Interrupt handler", nil
	})

	if *clientDevice == "" {
		routines = append(
			routines,
			func() (string, error) {
				return "server", nbd.NewTCPSocketServer(ctx, *port)
			},
		)
	} else {
		domainSockets := make(chan uintptr)

		routines = append(
			routines,
			func() (string, error) {
				return "Client", nbd.Client(ctx, *clientDevice, domainSockets)
			},
			func() (string, error) {
				return "server", nbd.NewDomainSocketServer(domainSockets)
			},
		)
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
