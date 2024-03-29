# Benchmark for adaptiveservice, mainly for performance test on transport layer.
server/server

scopes="process os lan wan"
nlist="1 2 4 8 16 32 64 128 256"
slist="32 128 512 2048 8192 32768 131072"
for scope in $scopes; do
	for n in $nlist; do
		for s in $slist; do
			cmd="client/client -t 10 -scope $scope -s $s -n $n"
			echo $cmd
			eval $cmd
			sleep 5
		done
		sleep 10
	done
	sleep 15
done > asbench.log

