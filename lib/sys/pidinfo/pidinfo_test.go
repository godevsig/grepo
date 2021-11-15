package pidinfo

import (
	"fmt"
	"testing"
	"time"
)

func showPidInfo(pi *PidInfo, level int, prefix string, lastone bool) {
	ucpu, scpu := pi.CPUpercent()

	var s1, s2 string
	if level == 0 {
		s1, s2 = "", ""
	} else {
		s1, s2 = "├─ ", "│  "
		if lastone {
			s1, s2 = "└─ ", "   "
		}
	}

	line := fmt.Sprintf("%5d %7.2f %7.2f %8d", pi.Pid(), ucpu, scpu, pi.Rss())
	line = fmt.Sprintf("%s %s%s", line, prefix+s1, pi.Cmdline())
	fmt.Printf("%s\n", line)

	threads := pi.Threads()
	nthrd := len(threads)
	children := pi.Children()
	nchld := len(children)

	for i, ti := range threads {
		ucpu, scpu := ti.CPUpercent()

		s1, s2 := "├> ", "│  "
		if i == nthrd-1 && nchld == 0 {
			s1 = "└> "
		}
		if lastone {
			s2 = "   "
		}
		if level == 0 {
			s2 = ""
		}

		line := fmt.Sprintf("%5d %7.2f %7.2f %8d", ti.Tid(), ucpu, scpu, ti.Rss())
		line = fmt.Sprintf("%s %s%s", line, prefix+s2+s1, ti.Comm())
		fmt.Printf("%s\n", line)
	}

	for i, child := range children {
		showPidInfo(child, level+1, prefix+s2, i == nchld-1)
	}
}

func testPi(pid int, t *testing.T) {
	done := make(chan bool)
	go func() {
		time.Sleep(3 * time.Second)
		done <- true
	}()

	pi, err := NewPidInfo(pid, CheckAllTask)
	if err != nil {
		t.Errorf("main: %v\n", err)
		return
	}

	t.Log("Check Level:", pi.CheckLevel())
	t.Log("root priority:", pi.Priority())

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if err := pi.Update(); err != nil {
				t.Errorf("main: %v\n", err)
				return
			}
			showPidInfo(pi, 0, "", true)
		}
	}
}

func TestP1Info(t *testing.T) {
	testPi(1, t)
}

func TestAllInfo(t *testing.T) {
	testPi(-1, t)
}

func TestSysInfo(t *testing.T) {
	done := make(chan bool)
	go func() {
		time.Sleep(10 * time.Second)
		done <- true
	}()

	si, err := NewSysInfo()
	if err != nil {
		t.Errorf("main: %v\n", err)
		return
	}

	t.Log("CPU number:", si.NumCPU())

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if err := si.Update(); err != nil {
				t.Errorf("main: %v\n", err)
				return
			}

			names := []string{"user", "nice", "system", "idle", "iowait", "irq", "softirq", "steal", "guest", "guestnice"}
			for _, name := range names {
				v := si.SysPercentBy(-1, name)
				t.Log("all", name, v)
				v = si.SysPercentBy(0, name)
				t.Log("core0", name, v)
			}
		}
	}
}

func BenchmarkP1InfoUpdate(b *testing.B) {
	pi, err := NewPidInfo(1, CheckAllTask)
	if err != nil {
		b.Errorf("main: %v\n", err)
		return
	}
	time.Sleep(time.Second)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := pi.Update(); err != nil {
			b.Errorf("main: %v\n", err)
			return
		}
	}
}

func BenchmarkMapInter(b *testing.B) {
	pi, err := NewPidInfo(1, CheckAllTask)
	if err != nil {
		b.Errorf("main: %v\n", err)
		return
	}
	if err := pi.Update(); err != nil {
		b.Errorf("main: %v\n", err)
		return
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pi.Children()
	}
}
