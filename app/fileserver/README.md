# ndn-dpdk/app/fileserver

This package is the file server, implemented as a [traffic generator](../tg) producer module.
It speaks the [ndn6-file-server protocol](https://github.com/yoursunny/ndn6-tools/blob/main/file-server.md).
It requires at least one thread, running the `FileServer_Run` function.

The file server is configured with one or more **mountpoints**.
Each mountpoint maps from an NDN name prefix to a path on the filesystem.
The top directory of each mountpoint is `open`ed during file server initialization, and the user must not delete them while the file server is running.

## Request Processing Workflow

Upon receiving an Interest, the name prefix consisting of only GenericNameComponents is used to lookup the list of mountpoints and determine a filesystem path, while the suffix consisting of non-GenericNameComponents classifies the request into one of the following kinds:

* request for file or directory metadata
* request for a directory listing segment
* request for a file segment
* unrecognized request - dropped

The file server invokes `openat` to open the file or directory (or `dup` in case of a request to the mountpoint directory itself), and then gathers information about file size, etc, via `statx` syscall.
Metadata and directory listing requests are responded right away.

File segment requests are enqueued onto **io\_uring** as READV operations.
If multiple incoming Interests are requesting consecutive segments of the same file, they may be batched into the same READV operation; however, this batching logic is currently disabled because preliminary benchmark indicates it worsens performance.

## File Descriptor Caching

The file server maintains a hashtable of open file descriptors.
If a request refers to a file or directory that already has an open file descriptor, the same file descriptor is reused instead of calling `openat` again.
Each open file descriptor is associated with a reference count, which indicates how many inflight READV operations are using this file descriptor.

As soon as the reference count reaches zero, i.e. the file descriptor becomes unused, it is placed in a cleanup queue (doubly linked list).
This cleanup queue has a limited capacity (configurable through `keepFds` option); if it's full, the oldest unused file descriptor is `close`d.
If a new request locates a file descriptor in the cleanup queue (i.e. its reference count is zero), the file descriptor is removed from the cleanup queue.
In short, the hashtable contains both active and unused file descriptors, while the cleanup queue forms a FIFO cache of unused file descriptors.

The result of `statx` syscall is stored together with each file descriptor, to avoid invoking `statx` for every request.
It is refreshed every few seconds (configurable through `statValidity` option) to keep the information up-to-date.

If the file server is configured to have multiple threads, each thread has its own file descriptor hashtable.
`InputDemux` for incoming Interests can dispatch Interests based on their name prefixes consisting of only GenericNameComponents, so that requests for the same file go to the same thread, eliminating the overhead of opening the same file in multiple threads.

## Limitations

The file server does not perform Data signing.
Each Data packet has a Null signature, which provides no integrity or authenticity protection.

Directory listing is truncated to one Data segment.
