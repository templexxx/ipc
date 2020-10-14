package ipc

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
	"testing"
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

// TODO make it choice.
// TODO clean testproc after testing.
// TODO use TestMain function.
func init() {
	cmd := exec.Command("go", "build", "cmd/testproc/testproc.go")
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func TestSameDataMultiProcesses(t *testing.T) {
	key := 1
	size := 8
	s, err := SHMGet(key, size)
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

	startMem := getSysMem()
	s, err := SHMGet(key, size)
	if err != nil {
		t.Fatal(err)
	}
	err = s.Attach()
	if err != nil {
		t.Fatal(err)
	}
	afterAttachMem := getSysMem()

	for i := 0; i < 8; i++ {
		cmd := exec.Command("./testproc", "-cmd", "detach", "-key", "2", "-size", "1073741824")
		cmd.Stdout = os.Stdout
		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	}
	afterMultAttachDeatch := getSysMem()
	fmt.Println(startMem, afterAttachMem, afterMultAttachDeatch)
	s.Detach()

	afterAllDeatch := getSysMem()
	fmt.Println(afterAllDeatch)
}

func getSysMem() uint64 {

	in := &syscall.Sysinfo_t{}
	err := syscall.Sysinfo(in)
	if err != nil {
		return 0
	}
	return in.Totalram * uint64(in.Unit)

}
