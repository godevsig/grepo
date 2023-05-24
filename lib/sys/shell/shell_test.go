package shell

import "fmt"

func ExampleRun() {
	out, _ := Run("echo abc")
	fmt.Print(out)
	out, _ = Run("echo github.com/godevsig/grepo/lib-sys/shell | tr / _")
	fmt.Print(out)
	out, _ = Run("echo starting; echo blocking; sleep 3")
	fmt.Print(out)
	out, _ = Run("echo non-blocking but no output; sleep 3 &")
	fmt.Print(out)

	//Output:
	//abc
	//github.com_godevsig_grepo_lib-sys_shell
	//starting
	//blocking
}
