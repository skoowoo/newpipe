package newpipe

import (
	"testing"
	"time"
)

func TestWriteBlockWhenRO(t *testing.T) {
	p, err := New(NB_RO)
	if err != nil {
		t.Fatal(err)
	}

	c := make(chan int, 1000)

	go func() {
		for {
			if _, err := p.WaitWrite([]byte("1234567890"), 0); err != nil {
				t.Fatal(err)
			}
			c <- 1
		}
	}()

	for {
		select {
		case <-c:
		case <-time.After(time.Second * 10):
			t.Log("Write blocking")
			return
		}
	}
}

func TestReadNonBlockWhenRO(t *testing.T) {
	p, err := New(NB_RO)
	if err != nil {
		t.Fatal(err)
	}

	var buf [10]byte

	if _, err := p.WaitRead(buf[:], 10000); err != nil {
		if err != ErrReadTimeout {
			t.Error(err)
		}
	}
	t.Log("Read nonblock")
}

func TestReadNonBlockWhenRO2(t *testing.T) {
	p, err := New(NB_RO)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 100; i++ {
		if _, err := p.WriteFile().Write([]byte("A")); err != nil {
			t.Error(err)
		}
	}

	var buf [100]byte
	if n, err := p.WaitRead(buf[:], 1000); err != nil {
		t.Error(err)
	} else {
		if n != 100 {
			t.Error("not 100")
		}
	}

	if _, err := p.WaitRead(buf[:], 1000); err != nil {
		if err != ErrReadTimeout {
			t.Error(err)
		}
	}
}
