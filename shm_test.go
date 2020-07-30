package ipc

import (
	"bytes"
	"testing"
)

func TestSHMGet(t *testing.T) {
	s, err := SHMGet(1, 8)
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
	defer s2.Detach()

	if s.id != s2.id {
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
