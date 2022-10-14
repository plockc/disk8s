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

	"github.com/plockc/disk8s/nbd/replica"
)

func main() {
	killCtx, cancelKill := context.WithCancel(context.Background())
	go handleKill(killCtx)

	ctx, cancel := context.WithCancel(killCtx)

	flag.Parse()

	wg := sync.WaitGroup{}

	routines := []func() (string, error){}

	routines = append(routines, func() (string, error) {
		handleSignal(ctx, cancel)
		return "Interrupt handler", nil
	})

	routines = append(routines, func() (string, error) {
		replica.HandleRequests(ctx)
		return "Data Disk", nil
	})

	// wait for routines to complete before exiting
	wg.Wait()

	cancelKill()
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
