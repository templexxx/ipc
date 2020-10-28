package ipc

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"testing"
	"time"
)

func isSHMClean(t *testing.T, start uint64) {
	if getShm()-start != 0 {
		t.Fatal("shm leak")
	}
}

func TestSameDataSingleProcess(t *testing.T) {
	start := getShm()

	s, err := SHMGet(1, 8)
	if err != nil {
		t.Fatal(err)
	}
	err = s.Attach()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Detach()
	defer s.Remove()

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
	defer s2.Remove()

	defer isSHMClean(t, start)

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
	start := getShm()

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
	defer s.Remove()
	defer isSHMClean(t, start)

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

// Test multi-process attach same shm, one process will call detach and remove, others will only detach but not remove,
// see the shm will leak or not.
// Expect: not leak.
func TestSHM_Detach(t *testing.T) {
	key := 2
	size := 1 << 30

	start := getShm()
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

	afterAttachMem := getShm()

	for i := 0; i < 8; i++ {
		cmd := exec.Command("./testproc", "-cmd", "detach", "-key", "2", "-size", "1073741824")
		cmd.Stdout = os.Stdout
		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	}
	afterMultAttachDeatch := getShm()
	if !bytes.Equal(buf, s.Bytes[:len(buf)]) {
		t.Fatal("data mismatch")
	}

	if afterAttachMem-start != 1<<30 {
		t.Fatal("new shm size mismatch")
	}

	if afterAttachMem != afterMultAttachDeatch {
		t.Fatal("shm should still survive")
	}

	err = s.Detach()
	if err != nil {
		t.Fatal(err)
	}

	err = s.Detach()
	if err != nil {
		t.Fatal(err)
	}
	err = s.Remove()
	if err != nil {
		t.Fatal(err)
	}
	isSHMClean(t, start)
}

// Test kill all processes, none of these processes will call detach or remove.
// Expect: not leak.
func TestSHM_Kill(t *testing.T) {
	start := getShm()

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

	isSHMClean(t, start)
}

func getShm() uint64 {

	in := &syscall.Sysinfo_t{}
	err := syscall.Sysinfo(in)
	if err != nil {
		return 0
	}
	return in.Sharedram * uint64(in.Unit)
}
