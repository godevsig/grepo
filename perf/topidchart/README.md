# 🚀 topid chart

topid is a cpu/mem stats visualization profiler.
See [godevsig/grepo](https://github.com/godevsig/grepo) for details.

## Set CPU/MEM filter

Appending `?filter=avg CPU,max CPU,avg MEM,max MEM` to the URL outputted by topid can
set the filter which determines which processes you want to show in the chart.
The default value is `?filter=2,10,10,20` which means:

- The processes whose avg CPU usage less than 2% AND max CPU usage less than 10%
  will not show in CPU usage chart.
- The processes whose avg MEM usage less than 10 MB AND max MEM usage less than 20 MB
  will not show in MEM usage chart.

For example, changing `http://10.10.10.179:9998/meaningfultag/20211111-xdtfmvhd` to
`http://10.10.10.179:9998/meaningfultag/20211111-xdtfmvhd?filter=1,10,5,10` will
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
wget http://10.10.10.179:8088/gshell/release/latest/gshell.$(uname -m) -O bin/gshell
chmod +x bin/gshell

# Run gshell daemon
alias gsh='bin/gshell'
gsh -loglevel info daemon -registry 10.10.10.179:11985 -bcast 9923 &
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

~/gshell # gsh run -rt 91 -i perf/topid/topid.go -chart -snapshot -sys -i 5 -tag meaningfultag -info "free,uname -a"
```

Follow the URL that topid.go outputs to get topid chart.

See

1. [gshell daemon quick start](https://github.com/godevsig/gshellos/-/wikis/home)
1. [Deploy gshell daemon](https://github.com/godevsig/gshellos/-/blob/master/docs/daemon.md)
1. [topid usage](https://github.com/godevsig/grepo/-/blob/master/perf/topid/README.md)
