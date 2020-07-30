package ipc

import (
	"errors"
	"fmt"
	"reflect"
	"syscall"
	"unsafe"
)

type SHM struct {
	Key   int
	Size  int
	Data  uintptr
	id    uintptr
	Bytes []byte // It's easy to use a bytes slice in Go because lacking of pointer ops.
}

const (
	// IPC_CREAT create if key is nonexistent
	// Copied from kernel source code.
	IPC_CREAT = 00001000 // create entry if key does not exist
)

// SHMGet gets a shared memory with specified key and size.
func SHMGet(key, size int) (*SHM, error) {
	id, _, err := syscall.Syscall(syscall.SYS_SHMGET, uintptr(key), uintptr(size), IPC_CREAT|0600)
	if int(id) == -1 {
		return nil, errors.New(fmt.Sprintf("shm get failed: %s", err))
	}

	addr, _, err := syscall.Syscall(syscall.SYS_SHMAT, id, 0, 0) // Let OS chooses mem address.
	if int(addr) == -1 {
		return nil, errors.New(fmt.Sprintf("shm attach failed: %s", err))
	}

	bh := reflect.SliceHeader{
		Data: addr,
		Len:  size,
		Cap:  size,
	}

	return &SHM{
		Key:   key,
		id:    id,
		Size:  size,
		Data:  addr,
		Bytes: *(*[]byte)(unsafe.Pointer(&bh)),
	}, nil
}

// Detach detaches shared memory.
func (s *SHM) Detach() error {
	_, _, err := syscall.Syscall(syscall.SYS_SHMDT, s.Data, 0, 0)
	if err == 0 {
		return nil
	}
	return err
}
