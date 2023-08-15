package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	as "github.com/godevsig/adaptiveservice"

	"github.com/godevsig/glib/sys/pidinfo"
	"github.com/godevsig/glib/sys/shell"
	topid "github.com/godevsig/grepo/perf/topidchart/topidchart"
)

type pidValue []int

func (pids *pidValue) String() string {
	return "[-1]"
}

func (pids *pidValue) Set(value string) error {
	pid, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	*pids = append(*pids, pid)
	return nil
}

type showFlag struct {
	detailcpu bool
	tree      bool
	memMB     bool
	verbose   bool
	thread    bool
	child     bool
	pss       bool
}

var (
	version    = "0.2.1"
	sf         showFlag
	pids       pidValue
	interval   uint
	count      int
	recordMode bool
	sessionTag string
	snapshot   bool
	sys        bool
	infocmds   string
)

func isKernelThread(pi *pidinfo.PidInfo) bool {
	cmd := pi.Cmdline()
	rss := pi.Rss()

	return cmd == "" && rss == 0
}

func showPidInfo(sf *showFlag, w io.Writer, pi *pidinfo.PidInfo, level int, prefix string, lastone bool) {
	cpuUtilization := func(ucpu, scpu float64) string {
		var result string
		if sf.detailcpu {
			result = fmt.Sprintf("%7.2f %7.2f", ucpu, scpu)
		} else {
			result = fmt.Sprintf("%7.2f", ucpu+scpu)
		}

		return result
	}

	var s1, s2 string
	if sf.tree {
		s1, s2 = "├─ ", "│  "
		if lastone {
			s1, s2 = "└─ ", "   "
		}
		if level == 0 {
			s1, s2 = "", ""
		}
	}

	ucpu, scpu := pi.CPUpercent()
	var mem uint64
	if sf.pss {
		mem = pi.Pss()
	} else {
		mem = pi.Rss()
	}
	if sf.memMB {
		mem = mem / 1024
	}

	line := fmt.Sprintf("%5d %4d %s %8d", pi.Pid(), pi.Priority(), cpuUtilization(ucpu, scpu), mem)
	if sf.verbose {
		cmd := pi.Cmdline()
		if isKernelThread(pi) {
			cmd = "[" + pi.Comm() + "]"
		}
		line = fmt.Sprintf("%s %s%s", line, prefix+s1, cmd)
	} else {
		cmd := pi.Comm()
		if isKernelThread(pi) {
			cmd = "[" + cmd + "]"
		}
		line = fmt.Sprintf("%s %s", line, cmd)
	}
	if pi.Pid() != -1 {
		fmt.Fprintf(w, "%s\n", line)
	}

	children := pi.Children()

	if sf.thread {
		threads := pi.Threads()
		nthrd := len(threads)
		// check only if the pid has more than 1(itself) threads
		if nthrd > 1 {
			nchld := len(children)
			nzCPUpct, zCPUpct := 0, ""

			for i, ti := range threads {
				ucpu, scpu := ti.CPUpercent()

				if sf.tree {
					s1 = "├> "
					if ucpu+scpu <= 0.01 {
						zCPUpct = zCPUpct + strconv.Itoa(ti.Tid()) + " "
						nzCPUpct++
						continue
					}
					if i == nthrd-1 && nchld == 0 && nzCPUpct == 0 {
						s1 = "└> "
					}
				}

				line := fmt.Sprintf("%5d %4d %s %8d", ti.Tid(), ti.Priority(), cpuUtilization(ucpu, scpu), mem)
				if sf.verbose {
					line = fmt.Sprintf("%s %s<%s>", line, prefix+s2+s1, ti.Comm())
				} else {
					line = fmt.Sprintf("%s <%s>", line, ti.Comm())
				}
				fmt.Fprintf(w, "%s\n", line)
			}

			if nzCPUpct != 0 {
				zCPUpct = zCPUpct[:len(zCPUpct)-1]
				line := fmt.Sprintf("%5s %4d %s %8d", "+", pi.Priority(), cpuUtilization(0, 0), mem)
				s1 := "├> "
				if nchld == 0 {
					s1 = "└> "
				}
				line = fmt.Sprintf("%s %s%d*<%s>: [%s]", line, prefix+s2+s1, nzCPUpct, pi.Comm(), zCPUpct)
				fmt.Fprintf(w, "%s\n", line)
			}
		}
	}

	if sf.child {
		n := len(children)
		for i, child := range children {
			showPidInfo(sf, w, child, level+1, prefix+s2, i == n-1)
		}
	}
}

func getSysInfo() topid.SysInfo {
	ncpu, _ := shell.Run("nproc")
	cpuinfo, _ := shell.Run("cat /proc/cpuinfo")
	uname, _ := shell.Run("uname -a")

	index := strings.Index(cpuinfo, "\n\n")
	cpuinfo = cpuinfo[:index+1]
	if lines := strings.Split(cpuinfo, "\n"); len(lines) >= 5 {
		cpuinfo = strings.Join(lines[1:5], "\n")
	}
	cpuinfo = "CPU(s)          : " + ncpu + cpuinfo + "\n"

	return topid.SysInfo{
		CPUInfo:    cpuinfo,
		KernelInfo: uname,
	}
}

var running = true

// Start starts the app
func Start(args []string) (err error) {
	// more CPUs cost more
	if runtime.NumCPU() > 2 {
		runtime.GOMAXPROCS(2)
	}

	fmt.Println("Hello 你好 Hola Hallo Bonjour Ciao Χαίρετε こんにちは 여보세요")
	fmt.Println("Version:", version)

	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.SetOutput(os.Stdout)

	flags.Var(&pids, "p", "process id. Multiple -p is supported, but -tree and -child are ignored")
	flags.BoolVar(&sf.verbose, "v", false, "enable verbose output")
	flags.BoolVar(&sf.child, "child", false, "enable child processes")
	flags.BoolVar(&sf.thread, "thread", false, "enable threads")
	flags.BoolVar(&sf.tree, "tree", false, "show all child processes in tree, implies -child -v")
	flags.UintVar(&interval, "i", 1, "wait interval seconds between each run")
	flags.IntVar(&count, "c", 3600, "stop after count times")
	flags.BoolVar(&sf.detailcpu, "detailcpu", false, "show detail CPU utilization")
	flags.BoolVar(&sf.memMB, "MB", false, "show mem in MB (default in KB)")
	flags.BoolVar(&sf.pss, "pss", false, "use pss mem, high overhead, often needs root privilege (default rss mem)")
	flags.BoolVar(&recordMode, "chart", false, "record data to chart server for data parsing")
	flags.StringVar(&sessionTag, "tag", "temp", "tag is part of the URL, used to mark this run, only works with -chart")
	flags.BoolVar(&snapshot, "snapshot", false, "also add snapshot of the pid 1's tree to records, only works with -chart")
	flags.BoolVar(&sys, "sys", false, "collect overall system status data, only works with -chart")
	flags.StringVar(&infocmds, "info", "", "comma separated list of commands to collect system infos")

	if err = flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			err = nil
		}
		return err
	}

	var chartConn as.Connection
	if recordMode {
		c := as.NewClient().SetDiscoverTimeout(3)
		chartConn = <-c.Discover("platform", "topidchart")
		if chartConn == nil {
			return errors.New("connect to chart server failed")
		}
		defer chartConn.Close()
		var infoCmds []string
		if len(infocmds) != 0 {
			infoCmds = strings.Split(infocmds, ",")
		}

		var extrainfo string
		for _, cmd := range infoCmds {
			if len(cmd) != 0 {
				out, _ := shell.Run(cmd)
				extrainfo += fmt.Sprintf("######Command######\n$ %s\n%s\n", cmd, out)
			}
		}

		sessionReq := topid.SessionRequest{
			Tag:       sessionTag,
			SysInfo:   getSysInfo(),
			ExtraInfo: extrainfo,
		}
		var sessionRep topid.SessionResponse
		if err := chartConn.SendRecv(&sessionReq, &sessionRep); err != nil {
			return err
		}
		fmt.Println("Visit below URL to get the chart:")
		if strings.Contains(sessionRep.ChartURL, "0.0.0.0") {
			fmt.Println("(replace 0.0.0.0 with the real IP of the chart server)")
		}
		fmt.Println(sessionRep.ChartURL)
	}

	sysStat := false
	if sys {
		sysStat = true
	}

	if sf.tree {
		sf.child = true
		sf.verbose = true
	}

	var ssf showFlag
	if snapshot {
		ssf.child = true
		ssf.detailcpu = true
		ssf.memMB = true
		ssf.thread = true
		ssf.tree = true
		ssf.verbose = true
	}

	var tgtPids []int
	for _, pid := range pids {
		if pid < 0 {
			tgtPids = nil
			break
		}
		tgtPids = append(tgtPids, pid)
	}

	if len(tgtPids) > 1 {
		sf.tree = false
		sf.child = false
	}
	if len(tgtPids) == 0 {
		tgtPids = []int{-1}
		sf.child = true
	}

	checkLevel := pidinfo.CheckSingle
	if sf.thread {
		checkLevel |= pidinfo.CheckThread
	}
	if sf.child {
		checkLevel |= pidinfo.CheckChild
	}
	if sf.pss {
		checkLevel |= pidinfo.CheckPss
	}

	var si *pidinfo.SysInfo
	piMap := make(map[int]*pidinfo.PidInfo, len(tgtPids))
	psPrepare := func() {
		if sysStat {
			tsi, err := pidinfo.NewSysInfo()
			if err != nil {
				panic(err)
			}
			si = tsi
		}

		for _, pid := range tgtPids {
			pi, err := pidinfo.NewPidInfo(pid, checkLevel)
			if err != nil {
				//fmt.Printf("main: %v\n", err)
				continue
			}
			piMap[pid] = pi
		}
	}

	psCollect := func() {
		if sysStat {
			if err := si.Update(); err != nil {
				panic(err)
			}
		}

		for _, pid := range tgtPids {
			pi := piMap[pid]
			if pi != nil {
				if err := pi.Update(); err != nil {
					//fmt.Printf("main: %v\n", err)
					piMap[pid] = nil
					continue
				}
			}
		}
	}

	show := func() {
		fmt.Fprintln(os.Stdout, "  PID PRIO     CPU      MEM COMM")
		for _, pid := range tgtPids {
			pi := piMap[pid]
			if pi != nil {
				showPidInfo(&sf, os.Stdout, pi, 0, "", true)
			}
		}
	}

	processesRecord2stream := func() error {
		t := time.Now().Unix()
		pspayload := []topid.ProcessInfo{}
		if sysStat {
			sysPerCPU, err := si.SysPercent()
			if err == nil {
				procinfo := topid.ProcessInfo{}
				procinfo.Pid = 0
				procinfo.Ucpu = 0
				procinfo.Mem = 0
				for _, sysCPU := range sysPerCPU {
					if sys {
						for k, v := range sysCPU {
							procinfo.Name = k
							procinfo.Scpu = float32(v)
							pspayload = append(pspayload, procinfo)
						}
					}
				}
			}
			memInfo, err := si.GetMemInfo()
			if err == nil {
				procinfo := topid.ProcessInfo{}
				procinfo.Pid = 0
				procinfo.Ucpu = 0
				procinfo.Scpu = 0
				for k, v := range memInfo {
					procinfo.Name = k
					procinfo.Mem = v
					pspayload = append(pspayload, procinfo)
				}
			}
		}

		for _, pid := range tgtPids {
			pi := piMap[pid]
			if pi != nil {
				for _, pstat := range pi.ProcessStatAll() {
					procInfo := topid.ProcessInfo{
						Pid:  pstat.Pid,
						Name: pstat.Name,
						Ucpu: float32(pstat.Ucpu),
						Scpu: float32(pstat.Scpu),
						Mem:  pstat.Mem,
					}
					pspayload = append(pspayload, procInfo)
				}
			}
		}

		record := topid.Record{Timestamp: t, Processes: pspayload}
		//fmt.Fprintln(os.Stderr, time.Unix(t, 0), record)
		if err := chartConn.Send(&record); err != nil {
			return err
		}
		return nil
	}

	var p1i *pidinfo.PidInfo
	if snapshot {
		pi, err := pidinfo.NewPidInfo(1, pidinfo.CheckAllTask)
		if err != nil {
			panic(err)
		}
		p1i = pi
	}

	doSnapshot := func() error {
		if err := p1i.Update(); err != nil {
			return err
		}

		t := time.Now().Unix()
		var buf strings.Builder
		fmt.Fprintln(&buf, "  PID PRIO    uCPU    sCPU      MEM COMM")
		showPidInfo(&ssf, &buf, p1i, 0, "", true)
		if err := chartConn.Send(&topid.Record{Timestamp: t, Snapshot: buf.String()}); err != nil {
			return err
		}
		return nil
	}

	psPrepare()

	//use uint because we want -1 to be a big number
	for i := uint(0); running && i < uint(count); i++ {
		time.Sleep(time.Second * time.Duration(interval))
		psCollect()
		if recordMode {
			if err := processesRecord2stream(); err != nil {
				return err
			}
		} else {
			show()
		}
		if snapshot && i%30 == 0 {
			if err := doSnapshot(); err != nil {
				return err
			}
		}
	}
	return
}

// Stop stops the app
func Stop() {
	fmt.Println("stopping...")
	running = false
}

func main() {
	if err := Start(os.Args[1:]); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
