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
we can use this ctl `numactl`

It's a useful ctl to control numa binds.

e.g.: Run program “myapp” on cpu node 0, using memory on node 0.

```
numactl --cpunodebind=0 --membind=0 myapp 
```

e.g.: Run program "myapp" on physical cpu 0,1,2,3,4.

```
numactl --physcpubind=+0-4 myapp
```

**Warning*:*

1. Ensure your applications won't use too much memory.

2. If you want bind node after starting, please using cgroup.