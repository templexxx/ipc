package ipc

import (
	"bufio"
	"bytes"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

func isSHMClean(t *testing.T, start int) {

	cnt, _, err := getSHMStatus()
	if err != nil {
		t.Fatal(err)
	}

	if cnt-start != 0 {
		t.Fatalf("shm leak: %d", cnt-start)
	}
}

func isSHMLeakN(t *testing.T, start, n int) {

	cnt, _, err := getSHMStatus()
	if err != nil {
		t.Fatal(err)
	}

	if cnt-start != n {
		t.Fatalf("shm leak: exp: %d, but: %d", n, cnt-start)
	}
}

func TestSameDataSingleProcess(t *testing.T) {
	start, _, err := getSHMStatus()
	if err != nil {
		t.Fatal(err)
	}
	defer isSHMClean(t, start)

	s, err := SHMGet(1, 8192) // Using 8192 for avoiding get sys_info ignore too small size (because of the unit maybe KB?).
	if err != nil {
		t.Fatal(err)
	}
	err = s.Attach()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Remove()
	defer s.Detach()

	bs := s.Bytes
	for i := 0; i < 8192; i++ {
		bs[i] = uint8(i)
	}

	s2, err := SHMGet(1, 8192)
	if err != nil {
		t.Fatal(err)
	}
	err = s2.Attach()
	if err != nil {
		t.Fatal(err)
	}

	defer s2.Remove()
	defer s2.Detach()

	if s.ID != s2.ID {
		t.Fatal("shm id mismatch")
	}

	if !bytes.Equal(s.Bytes, s2.Bytes) {
		t.Fatal("shm bytes mismatch")
	}

	bs2 := s2.Bytes
	for i := 0; i < 8192; i++ {
		bs2[i] = uint8(8192 - i)
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
	start, _, err := getSHMStatus()
	if err != nil {
		t.Fatal(err)
	}
	defer isSHMClean(t, start)

	key := 2
	size := 8192
	s, err := SHMGet(uint(key), uint(size))
	if err != nil {
		t.Fatal(err)
	}
	err = s.Attach()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Remove()
	defer s.Detach()

	bs := s.Bytes
	for i := 0; i < size; i++ {
		bs[i] = uint8(i)
	}

	cmd := exec.Command("./testproc", "-cmd", "get_same", "-key", "2", "-size", "8192")
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
	key := 3
	size := 1 << 30

	startCnt, startAlloc, err := getSHMStatus()
	if err != nil {
		t.Fatal(err)
	}
	s, err := SHMGet(uint(key), uint(size))
	if err != nil {
		t.Fatal(err)
	}
	err = s.Attach()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Remove()
	defer s.Detach()
	buf := make([]byte, 1<<20)
	for i := range buf {
		buf[i] = uint8(i)
	}
	for i := 0; i < size/len(buf); i++ {
		copy(s.Bytes[i*len(buf):(i+1)*len(buf)], buf)
	}

	afterAttachCnt, afterAttachAlloc, err := getSHMStatus()
	if err != nil {
		t.Fatal(err)
	}

	if afterAttachCnt-startCnt != 1 {
		t.Fatal("new shm cnt mismatch", afterAttachCnt, startCnt)
	}

	if afterAttachAlloc-startAlloc != size/(1<<12) {
		t.Fatal("shm size mismatch")
	}

	for i := 0; i < 8; i++ {
		cmd := exec.Command("./testproc", "-cmd", "detach", "-key", "3", "-size", "1073741824")
		cmd.Stdout = os.Stdout
		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	}
	afterDetachCnt, afterDetachAlloc, err := getSHMStatus()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf, s.Bytes[:len(buf)]) {
		t.Fatal("data mismatch")
	}
	if afterDetachCnt-startCnt != 1 {
		t.Fatal("new shm cnt mismatch")
	}

	if afterDetachAlloc-startAlloc != size/(1<<12) {
		t.Fatal("shm size mismatch")
	}

	err = s.Detach()
	if err != nil {
		t.Fatal(err)
	}
	err = s.Remove()
	if err != nil {
		t.Fatal(err)
	}
	isSHMClean(t, startCnt)
}

// Test kill all processes, none of these processes will call detach or remove.
// Expect: 1 leak.
func TestSHM_Kill(t *testing.T) {

	startCnt, _, err := getSHMStatus()
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		s, err := SHMGet(4, 1073741824)
		if err != nil {
			t.Fatal(err)
		}
		err = s.Remove()
		if err != nil {
			t.Fatal(err)
		}
		isSHMClean(t, startCnt)
	}()

	m := new(sync.Map)

	for i := 0; i < 8; i++ {
		go func(j int) {
			cmd := exec.Command("./testproc", "-cmd", "sleep", "-key", "4", "-size", "1073741824")
			cmd.Stdout = os.Stdout
			err := cmd.Start()
			if err != nil {
				log.Fatal(err)
			}
			m.Store(j, cmd.ProcessState.Pid())
		}(i)
	}
	time.Sleep(time.Second)

	m.Range(func(key, value interface{}) bool {
		err = syscall.Kill(value.(int), syscall.SIGKILL)
		if err != nil {
			t.Fatal(err)
		}
		return true
	})

	isSHMLeakN(t, startCnt, 1)

}

func getSHMStatus() (cnt int, allocated int, err error) {
	//defer fmt.Println(cnt, allocated)

	output, err := exec.Command("ipcs", "-m", "-u").Output()
	if err != nil {
		return 0, 0, err
	}

	out := bytes.NewBuffer(output)
	r := bufio.NewScanner(out)
	for r.Scan() {
		text := r.Text()
		if text == " " || len(text) == 0 {
			continue
		}

		if strings.Contains(text, "segments") {
			s := strings.TrimLeft(text, "segments allocated ")
			s = strings.TrimSpace(s)
			if s == "" {
				break // No segments.
			}
			cnt, err = strconv.Atoi(s)
			if err != nil {
				return
			}
			continue
		}

		if strings.Contains(text, "pages allocated") {
			s := strings.TrimLeft(text, "pages allocated ")
			s = strings.TrimSpace(s)
			allocated, err = strconv.Atoi(s)
			if err != nil {
				return
			}
		}
	}
	return
}
