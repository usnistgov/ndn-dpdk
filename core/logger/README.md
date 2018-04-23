# ndn-dpdk/core/logger

This package provides a wrapper of [Logrus](https://github.com/sirupsen/logrus/) structured logger for Go.
C code in this project uses [zf\_log](https://github.com/wonder-mice/zf_log/) library.

Loggers in both C and Go support log level configuration through environment variables.
For log module "Foo", the initialization code first looks for "LOG\_Foo" and, if not found, looks for "LOG" environment variable.
The value of this environment variable shall be one of:

* **V**: VERBOSE level (C only)
* **D**: DEBUG level
* **I**: INFO level (default)
* **W**: WARNING level
* **E**: ERROR level
* **F**: FATAL level
* **N**: disabled (in C), PANIC level (in Go)
