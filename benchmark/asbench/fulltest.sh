gsh run -group asbench benchmark/asbench/server/asbenchserver.go

scopes="process os lan wan"
nlist="1 2 4 8 16 32 64 128 256"
slist="32 128 512 2048 8192 32768 131072"
for scope in $scopes; do
	for n in $nlist; do
		for s in $slist; do
			cmd="gsh run -group asbench -i -rm benchmark/asbench/client/asbenchclient.go -t 30 -scope $scope -s $s -n $n"
			echo $cmd
			eval $cmd
			sleep 10
		done
		sleep 20
	done
	sleep 30
done > asbench.log

