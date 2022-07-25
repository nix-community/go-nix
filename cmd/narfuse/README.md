# narfuse

Mount a single NAR file as a FUSE filesystem.

This is mainly an experiment to play with go-fuse. A more interesting version (for the future) would probably be a "nixstorefuse".

## Example usage

```console
# Create folder that will be used as a mount point
$ mkdir /tmp/mountpoint
# Mount the NAR on /tmp/mountpoint
$ ./narfuse /tmp/mountpoint ../../test/testdata/nar_1094wph9z4nwlgvsd53abfz8i117ykiv5dwnq9nnhz846s7xqd7d.nar &
# Check that the content is accessible
$ man /tmp/mountpoint/share/man/man1/hostname.1.gz 
$ ls -la /tmp/mountpoint/
total 0
drwxr-xr-x 0 root root 0 Jan  1  1970 bin
-rwxrwxrwx 1 root root 0 Jan  1  1970 sbin
drwxr-xr-x 0 root root 0 Jan  1  1970 share
# Unmount the filesystem
$ fusermount -u /tmp/mountpoint
```

## Known issues

* symlinks are not supported yet
* lots of TODOs and FIXMEs to address
* add unit and integration tests
