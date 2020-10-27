package ipc

import (
	"bytes"
	"log"
	"math"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"testing"
	"time"
)

func TestSameDataSingleProcess(t *testing.T) {
	s, err := SHMGet(1, 8)
	if err != nil {
		t.Fatal(err)
	}
	err = s.Attach()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Detach()

	bs := s.Bytes
	for i := 0; i < 8; i++ {
		bs[i] = uint8(i)
	}

	s2, err := SHMGet(1, 8)
	if err != nil {
		t.Fatal(err)
	}
	err = s2.Attach()
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Detach()

	if s.ID != s2.ID {
		t.Fatal("shm id mismatch")
	}

	if !bytes.Equal(s.Bytes, s2.Bytes) {
		t.Fatal("shm bytes mismatch")
	}

	bs2 := s2.Bytes
	for i := 0; i < 8; i++ {
		bs2[i] = uint8(8 - i)
	}

	if !bytes.Equal(s.Bytes, s2.Bytes) {
		t.Fatal("shm bytes mismatch")
	}
}

func TestMain(m *testing.M) {

	cmd := exec.Command("go", "build", "cmd/testproc/testproc.go")
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	m.Run()
	os.Remove("testproc")

}

func TestSameDataMultiProcesses(t *testing.T) {
	key := 1
	size := 8
	s, err := SHMGet(uint(key), uint(size))
	if err != nil {
		t.Fatal(err)
	}
	err = s.Attach()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Detach()

	bs := s.Bytes
	for i := 0; i < size; i++ {
		bs[i] = uint8(i)
	}

	cmd := exec.Command("./testproc", "-cmd", "get_same", "-key", "1", "-size", "8")
	cmd.Stdout = os.Stdout
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func TestSHM_Detach(t *testing.T) {
	key := 2
	size := 1 << 30

	startMem := getFreeMem()
	s, err := SHMGet(uint(key), uint(size))
	if err != nil {
		t.Fatal(err)
	}
	err = s.Attach()
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 1<<20)
	for i := range buf {
		buf[i] = uint8(i)
	}
	for i := 0; i < size/len(buf); i++ {
		copy(s.Bytes[i*len(buf):(i+1)*len(buf)], buf)
	}

	afterAttachMem := getFreeMem()

	for i := 0; i < 8; i++ {
		cmd := exec.Command("./testproc", "-cmd", "detach", "-key", "2", "-size", "1073741824")
		cmd.Stdout = os.Stdout
		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	}
	afterMultAttachDeatch := getFreeMem()
	if !bytes.Equal(buf, s.Bytes[:len(buf)]) {
		t.Fatal("data mismatch")
	}

	if startMem-afterAttachMem < 900*(1<<20) {
		t.Fatal("memory usage not match after attach", startMem, afterAttachMem)
	}
	if startMem-afterMultAttachDeatch < 900*(1<<20) {
		t.Fatal("memory usage should bigger: still has attach shm")
	}

	s.Remove()
}

func TestSHM_ProcessesExit(t *testing.T) {
	startMem := getFreeMem()

	for i := 0; i < 8; i++ {
		cmd := exec.Command("./testproc", "-cmd", "exit", "-key", "2", "-size", "1073741824")
		cmd.Stdout = os.Stdout
		err := cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	}

	exitMem := getFreeMem()
	if math.Abs(float64(startMem)-float64(exitMem)) > 256*(1<<20) {
		t.Fatal("memory usage should be almost as same as the beginning")
	}
}

func TestSHM_Kill(t *testing.T) {
	startMem := getFreeMem()

	m := new(sync.Map)

	for i := 0; i < 8; i++ {
		go func(j int) {
			cmd := exec.Command("./testproc", "-cmd", "sleep", "-key", "2", "-size", "1073741824")
			cmd.Stdout = os.Stdout
			err := cmd.Run()
			if err != nil {
				log.Fatal(err)
			}
			m.Store(j, cmd.ProcessState.Pid())
		}(i)
	}
	time.Sleep(time.Second)

	m.Range(func(key, value interface{}) bool {
		syscall.Kill(value.(int), syscall.SIGKILL)
		return true
	})

	exitMem := getFreeMem()
	if math.Abs(float64(startMem)-float64(exitMem)) > 256*(1<<20) {
		t.Fatal("memory usage should be almost as same as the beginning")
	}
}

func getFreeMem() uint64 {

	in := &syscall.Sysinfo_t{}
	err := syscall.Sysinfo(in)
	if err != nil {
		return 0
	}
	return in.Freeram * uint64(in.Unit)

}
