// See https://github.com/NetworkBlockDevice/nbd/blob/master/doc/proto.md

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/plockc/disk8s/nbd"
)

func main() {
	if err := nbd.Driver(context.Background()); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
