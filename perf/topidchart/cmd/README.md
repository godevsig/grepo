On a gshell enabled system, below command starts topid chart service.

`gsh run perf/topidchart/cmd/topidchart.go`

Dependency: topidchart needs "docit" service in gshell service network, to start one:

`gsh run render/docit/cmd/docit.go`

See [usage](../README.md)
