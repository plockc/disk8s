## Testing

I used [lima](https://github.com/lima-vm/lima) (used by Rancher Desktop) running alpine on M1 Mac

```
brew install lima
lima start --name=alpine template://alpine
lima shell alpine
apk add go
modprobe nbd
```

### Local client and server
```
go run ./cmd
```

### Local client and server with TCP
```
go run ./cmd -client /dev/nbd0 -tcp
```

### Use with nbd-client
```
go run ./cmd -tcp
```

```
nbdclient -p localhost /dev/nbd0
```

To disconnect
```
nbdclient -d /dev/nbd0
```

## References

busybox implementation [src](https://git.busybox.net/busybox/tree/networking/nbd-client.c)

[NBD Protocol](https://github.com/NetworkBlockDevice/nbd/blob/master/doc/proto.md), and implementations of client and server are there as well

[buse-go](https://github.com/samalba/buse-go)

[go-nbd](https://github.com/derlaft/go-nbd/blob/master/nbd.go) has a minimal implementation of an in memory client and server

[gondbserver](https://github.com/abligh/gonbdserver)
