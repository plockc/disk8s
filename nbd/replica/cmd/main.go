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

	"github.com/plockc/disk8s/nbd/internal/store"
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

	var serviceErr error
	routines = append(routines, func() (string, error) {
		fileStore, err := store.NewFile()
		if err == nil {
			err = replica.NewDataDiskServer(fileStore).HandleRequests(ctx)
		}
		return "Data Disk", err
	})

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

	cancelKill()

	if serviceErr != nil {
		fmt.Println("Exiting due to Error:", serviceErr)
		os.Exit(1)
	}
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
