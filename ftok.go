package ipc

import "syscall"

// Ftok returns a probably-unique key that can be used by System V IPC
// syscalls, e.g. msgget(), shmget().
// See ftok(3) or https://code.woboq.org/userspace/glibc/sysvipc/ftok.c.html
// or https://www.ibm.com/support/knowledgecenter/ssw_ibm_i_72/apis/p0zftok.htm
func Ftok(path string, id uint) (uint, error) {
	st := &syscall.Stat_t{}
	if err := syscall.Stat(path, st); err != nil {
		return 0, err
	}
	return uint((uint(st.Ino) & 0xffff) | uint((st.Dev&0xff)<<16) |
		((id & 0xff) << 24)), nil
}
