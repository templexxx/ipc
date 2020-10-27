# ipc
XSI shared memory in Go.

This lib implements basic APIs of shared memory:

1. shmget
2. shmattach
3. shmdetach
4. shmctl rmid

Users should decide which APIs to use, you could find examples in testing.

Some useful ctls:
1. ipcs 
2. ipcrm
