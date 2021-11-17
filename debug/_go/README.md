# Examples

- `gshell run -group namex -i debug/_go/setmaxprocs.go 4`: set GOMAXPROCS to 4 for target GRG namex, return the previous value.
- `gshell run -group namex -i debug/_go/setgcpercent.go 120`: set the garbage collection target percentage to 120 for target GRG namex, return the previous value.
- `gshell run -group namex -i debug/_go/showstack.go`: dump all stack of the target GRG namex
- `gshell run -group namex -i debug/_go/showmetric.go`: show runtime statistics of the target GRG namex
- `gshell run -group namex -i debug/_go/freemem.go`: in target GRG namex, forces a garbage collection followed by an attempt to return as much memory to the operating system as possible.
