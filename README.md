# Introduction

grepo is the central repo for gshell based apps/services, see godevsig/gshellos
for details.

# Code organization

There are 3 prefixes based on the execution mode:

- app-\* : the code will be interpreted by `gshell run`,
  this is where the client/service entry points should be put in.
- lib-\* : the code is compiled into gshell binary,
  providing relatively stable API.
- srv-\* : the code is compiled into gshell binary,
  this is where the code doing the actual job for services should be put in.

# Examples

To be able to run grepo apps, a `gshell daemon` should already be started on the system.

## topid and topid chart

topid is a tool that collects linux process statistic data like top, periodically sends
the data to topid chart server which then draws the CPU usage and MEM usage chart that
can be viewed in web.

- [topid usage](app-perf/topid/README.md)
- [topid chart usage](app-perf/topidchart/README.md)
- ![topid demo](app-perf/topid/gshell-topid.gif)
- ![topid chart demo](app-perf/topidchart/gshell-topidchart.gif)
