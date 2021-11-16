package shell

import "fmt"

func ExampleRun() {
	fmt.Print(Run("echo abc"))
	fmt.Print(Run("echo github.com/godevsig/grepo/lib-sys/shell | tr / _"))
	fmt.Print(Run("echo starting; echo blocking; sleep 3"))
	fmt.Print(Run("echo non-blocking but no output; sleep 3 &"))

	//Output:
	//abc
	//github.com_godevsig_grepo_lib-sys_shell
	//starting
	//blocking
}
