# Description of UML logging

Adapted from 
[this documentation](http://user-mode-linux.sourceforge.net/old/tty_logging.html).

User Mode Linux has/had a logging format that could be used for TTY
logging of systems running as honeypots.

Logs were captured in files with records that had headers of the following form,
followed by `len` bytes of data.

```c
struct tty_log_buf {
	int what;
	unsigned long tty;
	int len;
	int direction;
	unsigned long sec;
	unsigned long usec;
};
```

* `what` specified which action the log is about. It is one of the following:
    * `1` - open a TTY
    * `2` - close a TTY
    * `3` - write to a TTY
* `tty` is an opaque identifier for the TTY.
* `len` number of bytes following the header corresponding to the data associated with the header.
    * If the `what` was `1` (open), the bytes following are the name of the parent TTY.
    * If the `what` was `3` (write), the bytes following are the data written to the TTY.
* `direction` indicates whether the dat awas read or written from the TTY.
    * `1` is read
    * `2` is write
* `sec` is the seconds part of a UNIX timestamp.
* `usec` is the microseconds part of a UNIX timestamp.
              

