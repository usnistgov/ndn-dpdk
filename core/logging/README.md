# ndn-dpdk/core/logging

NDN-DPDK implements **structured logging**.
Log entries are written to stderr in JSON format.

## Loggers

NDN-DPDK components are organized into named loggers.
You can find all logger names in the codebase with this command:

```bash
git grep -wE 'N_LOG_INIT|logging\.New'
```

In addition:

* "DPDK" refers to DPDK libraries and drivers.
* "SPDK" refers to SPDK libraries and drivers.

## Log Level Configuration

Log level of each logger can be configured through environment variables.
For logger "Foo", the initialization code first looks for "NDNDPDK\_LOG\_Foo" and, if not found, looks for the generic "NDNDPDK\_LOG" environment variable.
The value of this environment variable should be one of the values in "env" column:

env | Go level | C level | DPDK level | SPDK level
----|----------|---------|------------|-----------
V   | DEBUG    | VERBOSE | DEBUG      | DEBUG
D   | DEBUG    | DEBUG   | INFO       | INFO
I   | INFO     | INFO    | NOTICE     | NOTICE
W   | WARNING  | WARNING | WARNING    | WARNING
E   | ERROR    | ERROR   | ERR        | ERROR
F   | FATAL    | (none)  | CRIT       | (none)
N   | FATAL    | (none)  | ALERT      | (none)

When the NDN-DPDK service is running, you can retrieve and change log levels via GraphQL (implemented in [package logginggql](logginggql)).

## Internals

```text
|--------|
|NDN-DPDK|  |------|  |------|
| C code |  | DPDK |  | SPDK |
|---+----|  |--+---|  |--+---|
    |          |         |
  |-v----------v---------v-|
  | DPDK rte_log.h library |
  |----------+-------------|
             |                   |--------|
|------------v---------------|   |NDN-DPDK|
| ealinit.processLogStream() |   |Go code |
|-------------------------+--|   |--+-----|
                           \       /
                          |-v-----v-|
                          |   zap   |
                          |---------|
```

Go code uses [zap](https://pkg.go.dev/go.uber.org/zap) structured logging library.
It is initialized in this package.

C code logs to DPDK logging library.
It generally uses a semi-structured format, where each line starts with a freeform message, followed by zero or more key-value pairs.

Package [spdkenv](../../dpdk/spdkenv) configures SPDK to log to DPDK logging library.

Package [ealinit](../../dpdk/ealinit) redirects DPDK log messages (including messages from NDN-DPDK C code and SPDK) to zap.
It creates a Unix pipe, and configures DPDK logging library to write to the pipe.
Then, it creates a goroutine that reads from the pipe, parses messages, and emits as zap log entries.
