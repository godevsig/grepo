# 🚀 topid chart

topid is a cpu/mem stats visualization profiler.

## Set CPU/MEM filter

Appending `?filter=avg CPU,max CPU,avg MEM,max MEM` to the URL outputted by topid can
set the filter which determines which processes you want to show in the chart.
The default value is `?filter=2,10,10,20` which means:

- The processes whose avg CPU usage less than 2% AND max CPU usage less than 10%
  will not show in CPU usage chart.
- The processes whose avg MEM usage less than 10 MB AND max MEM usage less than 20 MB
  will not show in MEM usage chart.

For example, changing `http://10.10.10.10:9998/meaningfultag/20211111-xdtfmvhd` to
`http://10.10.10.10:9998/meaningfultag/20211111-xdtfmvhd?filter=1,10,5,10` will
show more processes in the charts.

Be careful if you set the filter smaller than the default value, since it will slow down
the showing of the charts.

# How to start topid on target device

## Check if gshell daemon is running

Each system that wants to join gshell service network, one gshell daemon should be running on that system.

```shell
cd /path/to/gshell
alias gsh='bin/gshell'
gsh info
```

`gsh info` outputs `service not found: godevsig_gshellDaemon` if `gshell daemon` has not been started.
We need to download gshell and start `gshell daemon` first.

```shell
# Download gshell binary
mkdir -p gshell/bin
cd gshell
wget http://10.10.10.10:8088/gshell/release/latest/gshell.$(uname -a) -O bin/gshell
chmod +x bin/gshell

# Run gshell daemon
alias gsh='bin/gshell'
gsh -loglevel info daemon -registry 10.10.10.10:11985 -bcast 9923 &
```

Then `gsh info` should output below info:

```
~/gshell # alias gsh='bin/gshell'
~/gshell # gsh info
Version: v21.11.rc2
Build tags: stdbase,stdcommon,stdruntime,adaptiveservice,shell,log,pidinfo,topidchartmsg
Commit: 5e77199ae4d3c1579491177cacddb0d80b77bfd8
```

## Run topid app

Once `gshell daemon` has been started, use `gsh run` to run any go apps/services in the central repo:

```
~/gshell # gsh repo
github.com/godevsig/grepo master

~/gshell # gsh run -rt 91 -i app-perf/topid/topid.go -chart -snapshot -sys -i 5 -tag meaningfultag
```

Follow the URL that topid.go outputs to get topid chart.

See

1. [Deploy gshell daemon](https://github.com/godevsig/gshellos/blob/master/docs/daemon.md)
1. [topid usage](https://github.com/godevsig/grepo/tree/master/perf/topid/README.md)

# gshell introduction

gshell is gshellos based service management tool.  
gshellos is a simple pure golang service framework for linux devices that provides:

- Flexible running model
  - Mixed execution mode to run go apps/services
    - interpreted mode for flexibility, compiled mode for performance
    - mix-used in runtime, easy to switch
  - Isolated Gshell Runtime Environment(GRE)
    - one service/app runs in one GRE
    - GRE has separate OS input, output, args
    - GREs share memory by communicating
  - App/service group mechanism
    - GREs can be grouped to run in one Gshell Runtime Group(GRG)
    - applicable real-time scheduling policy on GRG
    - zero communication cost in same GRG: zero data copy, no kernel round trip
    - group/ungroup by gshell command line at runtime
  - Remote deployment
- Simplified and unified communication
  - Name based service publishing/discovery
    - a service is published under the name of {"publisher", "service"}
    - 4 scopes of service visibility: Process, OS, LAN, WAN
    - a service can be published in all the above scopes
    - a service is discovered in the above scope order
  - Message oriented client-server communication
    - servers define message structs, clients import message structs
    - simple Send() Recv() API and RPC alike SendRecv() API
    - data encoding/serializing when necessary
    - messages can be reordered by predefined priority
  - High concurrency model
    - client side multiplexed connection
    - server side auto scale worker pool
    - of course go routines and go channels
- Zero deploy dependency on all CPU arch
  - X86, ARM, MIPS, PPC...
  - embedded boxes, cloud containers, server VMs...
  - only one binary is needed
- Zero cost for service/app migration between different scopes/machines/locations
  - no code change, no recompile, no redeploy
  - gshell command line to move services/apps around at runtime
- Auto update without impacting the running services
- Interactive and native debugging with built-in REPL shell
- P2P network model
  - zero config, self discovered and managed network
  - auto reverse proxy for service behind NAT

See [godevsig/gshellos](https://github.com/godevsig/gshellos) for details.
