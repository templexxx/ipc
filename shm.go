package ipc

import (
	"errors"
	"fmt"
	"reflect"
	"syscall"
	"unsafe"
)

type SHM struct {
	Key   uint
	ID    uintptr
	Size  uint
	Data  uintptr
	Bytes []byte // It's easy to use a bytes slice in Go because lacking of pointer ops.
}

const (
	// IPC_CREAT create if key is nonexistent
	// Copied from kernel source code.
	IPC_CREAT = 00001000 // Create entry if key does not exist
	IPC_RMID  = 0        // Remove resource.
)

// SHMGet gets a shared memory with specified key and size.
func SHMGet(id, size uint) (*SHM, error) {
	key, errf := Ftok("/dev/null", id)
	if errf != nil {
		return nil, errf
	}
	shmid, _, err := syscall.Syscall(syscall.SYS_SHMGET, uintptr(key), uintptr(size), 0)
	if err == 0 && int(shmid) != -1 {
		return &SHM{
			Key:  key,
			ID:   shmid,
			Size: size,
		}, nil
	}

	shmid, _, err = syscall.Syscall(syscall.SYS_SHMGET, uintptr(key), uintptr(size), IPC_CREAT|0666)
	if err != 0 {
		return nil, err
	}
	if int(shmid) == -1 {
		return nil, errors.New(fmt.Sprintf("shm get failed: %s", err))
	}

	return &SHM{
		Key:  key,
		ID:   shmid,
		Size: size,
	}, nil

}

// SHMAttach attaches a shared memory to this process with specified id.
func (s *SHM) Attach() error {
	addr, _, err := syscall.Syscall(syscall.SYS_SHMAT, s.ID, 0, 0) // Let OS chooses mem address.
	if int(addr) == -1 {
		return errors.New(fmt.Sprintf("shm attach failed: %s", err))
	}

	bh := reflect.SliceHeader{
		Data: addr,
		Len:  int(s.Size),
		Cap:  int(s.Size),
	}
	s.Data = addr
	s.Bytes = *(*[]byte)(unsafe.Pointer(&bh))

	_ = s.Remove()

	return nil
}

// Remove removes the resource only after the last process detaches it.
func (s *SHM) Remove() error {
	_, _, err := syscall.Syscall(syscall.SYS_SHMCTL, s.ID, uintptr(IPC_RMID), 0)
	return err
}

// Detach detaches shared memory.
func (s *SHM) Detach() error {
	_, _, err := syscall.Syscall(syscall.SYS_SHMDT, s.Data, 0, 0)
	if err == 0 {
		return nil
	}
	s.Bytes = nil
	return err
}
