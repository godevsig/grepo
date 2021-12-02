Benchmark for adaptiveservice, mainly for performance test on transport layer.

How to test:

- start asbench server  
  `gsh run -group asbench benchmark/asbench/server/asbenchserver.go`
- run asbench client in the same grg as the server
  - in default os scope  
    `gsh run -group asbench -i -rm benchmark/asbench/client/asbenchclient.go -t 30`
  - in scope process  
    `gsh run -group asbench -i -rm benchmark/asbench/client/asbenchclient.go -t 30 -scope process`
- run client in another grg in default os scope  
  `gsh run -group another -i -rm benchmark/asbench/client/asbenchclient.go -t 30`

Usage:

```
Usage:
  -logLevel string
        debug/info/warn/error (default "info")
  -n int
        parallel number (default 1)
  -s int
        payload size in byte (default 32)
  -scope string
        process/os/lan/wan (default "os")
  -t int
        test for how long (default 3)
  -type string
        test type: download or upload (default "download")
```
