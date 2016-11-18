package newpipe

import (
	"errors"
	"os"
	"syscall"
)

var (
	ErrReadTimeout  = errors.New("Read pipe timeout")
	ErrWriteTimeout = errors.New("Write pipe timeout")
)

const (
	NB_RW = iota
	NB_RO
	NB_WO
)

type Newpipe struct {
	r    *os.File
	w    *os.File
	erfd int
	ewfd int
}

func New(flag int) (pipe *Newpipe, err error) {
	var p [2]int

	if flag == NB_RW {
		if err = syscall.Pipe2(p[0:], syscall.O_CLOEXEC|syscall.O_NONBLOCK); err != nil {
			return nil, os.NewSyscallError("pipe2", err)
		}
	}

	if flag == NB_RO || flag == NB_WO {
		syscall.ForkLock.RLock()
		if err = syscall.Pipe(p[0:]); err != nil {
			syscall.ForkLock.RUnlock()
			return nil, os.NewSyscallError("pipe", err)
		}
		syscall.CloseOnExec(p[0])
		syscall.CloseOnExec(p[1])

		if flag == NB_RO {
			syscall.SetNonblock(p[0], true)
		}
		if flag == NB_WO {
			syscall.SetNonblock(p[1], true)
		}
		syscall.ForkLock.RUnlock()
	}

	var (
		erfd int = -1
		ewfd int = -1
	)

	if flag == NB_RW || flag == NB_RO {
		if erfd, err = syscall.EpollCreate1(0); err != nil {
			return
		}

		event := syscall.EpollEvent{
			Fd:     int32(p[0]),
			Events: syscall.EPOLLIN,
		}
		if err = syscall.EpollCtl(erfd, syscall.EPOLL_CTL_ADD, p[0], &event); err != nil {
			return
		}
	}

	if flag == NB_RW || flag == NB_WO {
		if ewfd, err = syscall.EpollCreate1(0); err != nil {
			return
		}
	}

	return &Newpipe{
		r:    os.NewFile(uintptr(p[0]), "|0"),
		w:    os.NewFile(uintptr(p[1]), "|1"),
		erfd: erfd,
		ewfd: ewfd,
	}, nil
}

func (p *Newpipe) ReadFile() *os.File {
	return p.r
}

func (p *Newpipe) WriteFile() *os.File {
	return p.w
}

// Read read from pipe, will block.
func (p *Newpipe) Read(b []byte) (n int, err error) {
	return p.WaitRead(b, -1)
}

// WaitRead read from pipe with timeout.
func (p *Newpipe) WaitRead(b []byte, msec int) (n int, err error) {
	if p.erfd == -1 {
		return p.r.Read(b)
	}

	var (
		events [1]syscall.EpollEvent
		ready  int
	)

	for {
		if ready, err = syscall.EpollWait(p.erfd, events[:], msec); err != nil {
			if err == syscall.EINTR {
				continue
			}
			return
		}

		if ready == 0 {
			return 0, ErrReadTimeout
		}

		for {
			m := 0
			if m, err = syscall.Read(int(p.r.Fd()), b); err != nil {
				if err == syscall.EAGAIN {
					break
				}
				return
			}

			n += m
			b = b[n:]
			if len(b) == 0 {
				return
			}
		}
	}

	return
}

// Write write data into pipe, will block.
func (p *Newpipe) Write(b []byte) (n int, err error) {
	return p.WaitWrite(b, -1)
}

// WaitWrite write data into pipe with timeout.
func (p *Newpipe) WaitWrite(b []byte, msec int) (n int, err error) {
	if p.ewfd == -1 {
		return p.w.Write(b)
	}

	wait := func() (e error) {
		var (
			events [1]syscall.EpollEvent
			ready  int
		)

		event := syscall.EpollEvent{
			Fd:     int32(p.w.Fd()),
			Events: syscall.EPOLLOUT,
		}

		if e = syscall.EpollCtl(p.ewfd, syscall.EPOLL_CTL_ADD, int(p.w.Fd()), &event); e != nil {
			return
		}

		for {
			if ready, e = syscall.EpollWait(p.ewfd, events[:], msec); e != nil {
				if e == syscall.EINTR {
					continue
				}
			}
			break
		}

		if ready == 0 {
			return ErrWriteTimeout
		}

		return syscall.EpollCtl(p.ewfd, syscall.EPOLL_CTL_DEL, int(p.w.Fd()), &event)
	}

	for {
		m := 0
		if m, err = syscall.Write(int(p.w.Fd()), b); err != nil {
			if err == syscall.EAGAIN {
				if err = wait(); err != nil {
					return
				}
				continue
			}
			return
		}

		n += m
		b = b[n:]
		if len(b) == 0 {
			return
		}
	}

	return
}

func (p *Newpipe) Close() error {
	if p.erfd != -1 {
		syscall.Close(p.erfd)
	}
	if p.ewfd != -1 {
		syscall.Close(p.ewfd)
	}
	return nil
}
