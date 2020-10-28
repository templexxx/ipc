package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/templexxx/ipc"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var cmd = flag.String("cmd", "", "test cmd")
var key = flag.Uint("key", 0, "shm key")
var size = flag.Uint("size", 0, "shm size")

func init() {
	flag.Usage = func() {
		fmt.Printf("Usage")
	}
}

const (
	cmdGetSame = "get_same"
	cmdDetach  = "detach"
	cmdSleep   = "sleep"
)

func main() {
	flag.Parse()

	switch *cmd {
	case cmdGetSame:
		err := testGetSame(*key, *size)
		if err != nil {
			log.Fatal(err)
		}
	case cmdDetach:
		err := testDetach(*key, *size)
		if err != nil {
			log.Fatal(err)
		}
	case cmdSleep:
		err := testSleep(*key, *size)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func testGetSame(key, size uint) error {
	shm, err := ipc.SHMGet(key, size)
	if err != nil {
		return err
	}
	err = shm.Attach()
	if err != nil {
		return err
	}
	defer shm.Detach()

	for i, v := range shm.Bytes {
		if v != uint8(i) {
			return errors.New("value mismatch")
		}
	}
	_ = shm.Remove()
	return nil
}

func testDetach(key, size uint) error {
	shm, err := ipc.SHMGet(key, size)
	if err != nil {
		return err
	}
	err = shm.Attach()
	if err != nil {
		return err
	}
	for i, v := range shm.Bytes[:1<<20] {
		if v != uint8(i) {
			return errors.New("value mismatch")
		}
	}
	return shm.Detach()
}

func testSleep(key, size uint) error {
	shm, err := ipc.SHMGet(key, size)
	if err != nil {
		return err
	}
	err = shm.Attach()
	if err != nil {
		return err
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	var sig os.Signal
	go func() {
		sig = <-sc
		os.Exit(0)
	}()

	time.Sleep(30 * time.Second)
	return nil
}
