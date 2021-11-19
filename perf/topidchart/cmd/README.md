On a gshell enabled system, below command starts topid chart service.

`gsh run perf/topidchart/cmd/topidchart.go`

Dependency: It is better to have "docit" service in gshell service network, to start one:

`gsh run render/docit/cmd/docit.go`

See [usage](../README.md)
