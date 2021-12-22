# Introduction

Topid is a gshell app that collects system/processes performance statistics data
and sends the data through network to a micro service named "topidchart" which
visualizes the data into charts.

# Usage

1. Get gshell binary and start gshell daemon if it has not been started.
2. Run topid.go:

```
cd /path/to/gshell
alias gsh='bin/gshell'
gsh run -i perf/topid/topid.go -chart -snapshot -sys -i 5 -tag meaningfulTag
```

Will output:

```
Hello 你好 Hola Hallo Bonjour Ciao Χαίρετε こんにちは 여보세요
Version: 0.2.1
Visit below URL to get the chart:
http://10.10.10.10:9998/meaningfultag/20211111-ntivpjtp
```

## Real time priority

It is seen sometimes especially in very high CPU load condition that there are abnormal
high spikes showed in CPU usage chart, it is because the grg(gshell runtime group)
in which topid.go is running has low scheduling priority, causing delayed colletion of
performance data and then CPU usage is incorrectly calculated.

That is why we use `gsh run -rt 91` to start topid.go, which sets the grg to SCHED_RR 91
priority:

```
~/gshell # gsh run -h
Usage of run [options] <file.go> [args...]
        Look for file.go in local file system or else in `gshell repo`,
        run it in a new GRE in specified GRG on local/remote system:
  -group string
        create new or use existing GRG (default "<random>")
  -i    enter interactive mode
  -rm
        automatically remove the GRE when it exits
  -rt string
        Set the GRG to SCHED_RR min/max priority 1/99 on new GRG creation
        Caution: gshell daemon must be started as root to set realtime attributes
```

Note: root privilege is required to be able to set real time priority.

## Interactive output mode

`gsh run -i` enters interactive output mode, you can `Ctrl+C` after URL shows up,
topid.go will continue to run in background.

If you `gsh run` without `-i`, e.g. `gsh run path/file.go` will prints its GRE ID
after the named `file.go` was successfully started. The output can then be checked
by `gsh log` with that GRE ID. You can also use `gsh ps` to find that ID.

```
~/gshell # gsh ps
GRE ID        IN GROUP            NAME                START AT             STATUS
af794a6cceaa  tfkenk.v21.11.rc3   topid               1970/10/23 21:15:12  running    2m8.097382229s

~/gshell # gsh log af794a6cceaa
Hello 你好 Hola Hallo Bonjour Ciao Χαίρετε こんにちは 여보세요
Version: 0.2.1
Visit below URL to get the chart:
http://10.10.10.10:9998/meaningfultag/20211111-ntivpjtp
```

## Stop and restart

You can stop the previous `gsh run` instance by its GRE ID.

```
~/gshell # gsh ps
GRE ID        IN GROUP            NAME                START AT             STATUS
62485b4fc5d3  qyvuzp.v21.11.rc3   hello               1970/10/23 21:19:12  exited:OK  11.10736ms
af794a6cceaa  tfkenk.v21.11.rc3   topid               1970/10/23 21:15:12  running    4m24.982602464s

~/gshell # gsh stop af794a6cceaa
af794a6cceaa
stopped
```

If you want to start topid again, just do `gsh restart` the stopped GRE is enough.
The following output shows GRE ID `af794a6cceaa` was stopped and restared, it generates a new URL:

```
~/gshell # gsh restart af794a6cceaa
af794a6cceaa
restarted

~/gshell # gsh log af794a6cceaa
Hello 你好 Hola Hallo Bonjour Ciao Χαίρετε こんにちは 여보세요
Version: 0.2.1
Visit below URL to get the chart:
http://10.10.10.10:9998/meaningfultag/20211111-xdtfmvhd

~/gshell # gsh ps
GRE ID        IN GROUP            NAME                START AT             STATUS
62485b4fc5d3  qyvuzp.v21.11.rc3   hello               1970/10/23 21:19:12  exited:OK  11.10736ms
af794a6cceaa  tfkenk.v21.11.rc3   topid               1970/10/23 21:19:50  running    7.420186219s
```

## Parameters of topid.go

- topid by default collect all process performance data, use `-p pid -child` to specify
  pid if only the pid and its child are interested.
- `-chart`: send data to chart server
- `-snapshot`: optional, periodically send process data to chart server
- `-sys`: also collect system CPU and mem data
- `-i 5`: collect data every 5 seconds
- `-tag name`: mark this run as "name", this tag will be part of the generated URL
- `-info "cmd1,cmd2,cmd3..."`: extra info of the system, will be shown in the web

```
~/gshell # gsh run -i -rm perf/topid/topid.go -h
Hello 你好 Hola Hallo Bonjour Ciao Χαίρετε こんにちは 여보세요
Version: 0.2.1
Usage:
  -MB
        show mem in MB (default in KB)
  -c int
        stop after count times (default 3600)
  -chart
        record data to chart server for data parsing
  -child
        enable child processes
  -detailcpu
        show detail CPU utilization
  -i uint
        wait interval seconds between each run (default 1)
  -info string
        comma separated list of commands to collect system infos
  -p value
        process id. Multiple -p is supported, but -tree and -child are ignored (default -1) (default [-1])
  -pss
        use pss mem, high overhead, often needs root privilege (default rss mem)
  -snapshot
        also add snapshot of the pid 1's tree to records, only works with -chart
  -sys
        collect overall system status data, only works with -chart
  -tag string
        tag is part of the URL, used to mark this run, only works with -chart (default "temp")
  -thread
        enable threads
  -tree
        show all child processes in tree, implies -child -v
  -v    enable verbose output

```

## RSS mem vs PSS mem

Memory usage data per process is collected in RSS by default, linux top command also uses RSS.

- RSS: resident set size, number of pages the process has in real memory.
- PSS: proportional share of the process memory mappings.

topid also supports PSS, but this usually uses more CPU to do the calculation.

# More info

- one system should start one gshell daemon, gshell daemon is responsible to run gshell
  apps(.go files) locally and remotely. Each daemon in the network is identified by
  provider ID, see below output, `00e3df230009` is the provider ID that can be used to
  run gshell apps on that system, from another gshell enabled system.

- "topidchart" service by publisher "platform" should be up and running in the network.
  Use `gshell list` to check all the available micro services, the last line is "topidchart".

```
$ gshell  list
PUBLISHER                 SERVICE                   PROVIDER      WLOP(SCOPE)
builtin                   IPObserver                self          1000
builtin                   LANRegistry               self            11
builtin                   providerInfo              self            11
builtin                   registryInfo              self            11
builtin                   reverseProxy              fa163ecfb434   100
builtin                   reverseProxy              self          1100
builtin                   serviceLister             self            11
godevsig                  codeRepo                  self          1111
godevsig                  grg-owcaxi.v21.11.rc2     self            10
godevsig                  gshellDaemon              00198f936ea2  1000
godevsig                  gshellDaemon              00e3df230009  1000
godevsig                  gshellDaemon              184a6fefbbba  1000
godevsig                  gshellDaemon              fa163ecfb434  1100
godevsig                  gshellDaemon              self          1111
godevsig                  updater                   self          1111
platform                  topidchart                fa163ecfb434  1000
```
