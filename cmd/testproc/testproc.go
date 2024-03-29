package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/templexxx/ipc"
	"log"
	"time"
)

var cmd = flag.String("cmd", "", "test cmd")
var key = flag.Uint("key", 0, "shm key")
var size = flag.Uint("size", 0, "shm size")
var sleepSeconds = flag.Int64("sleep", 3, "sleep seconds")

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
		err := testSleep(*key, *size, *sleepSeconds)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func testGetSame(key, size uint) error {
	shm, err := ipc.SHMCreateWithKey(key, size)
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
	shm, err := ipc.SHMCreateWithKey(key, size)
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

func testSleep(key, size uint, sleepSeconds int64) error {
	shm, err := ipc.SHMCreateWithKey(key, size)
	if err != nil {
		return err
	}
	err = shm.Attach()
	if err != nil {
		return err
	}

	time.Sleep(time.Duration(sleepSeconds) * time.Second)
	return nil
}
