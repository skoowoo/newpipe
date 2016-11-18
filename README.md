# newpipe
newpipe is a nonblocking pipe.


### Usage:

```
    // NB_RW set nonblock on the two ends (read and write) of the pipe.
    // NB_RO set nonblock on the read end of the pipe only.
    // NB_WO set nonblock on the write end of the pipe only.
	p, err := newpipe.New(newpipe.NB_RW)
	if err != nil {
        panic(err)
	}

    // WaitWrite write data into pipe, the sencond argument is timeout, millisecond.
    if _, err := p.WaitWrite([]byte("Hello, world"), 1000); err != nil {
        if err != newpipe.ErrWriteTimeout {
            panic(err)
        }

        // TODO: handle write timeout here.

    }

	var buf [100]byte

    // WaitRead read data from pipe, the second argument is timeout, millisecond.
	if _, err := p.WaitRead(buf[:], 1000); err != nil {
        if err != newpipe.ErrReadTimeout {
            panic(err)
        }

        // TODO: handle read timeout here.

	}
```
