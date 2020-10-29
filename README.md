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

## Performance

### NUMA

I think it's important to know some important details about NUMA before using shared memory.

In NUMA, different CPUs may cross nodes to access the same area of memory, it will
cost 10x(10ns -> 100ns) than normal memory random access. The default NUMA policy in kernel is:

**local allocation**

Which means kernel will allocate memory in same node for a process. If the node's memory
is full, kernel will try to swap, it may cause "swap insanity". When a process which doesn't
need much memory, it works great.

For shared memory, we may want processes which shared memory each other run on same node,
we can use this command `numactl`

e.g.: Runs program “myapp” on cpu 0, using memory on nodes 0 and 1.
```
numactl --cpubind=0 --membind=0,1 myapp 
```

**Warning*:*

Ensure your applications won't use too much memory.