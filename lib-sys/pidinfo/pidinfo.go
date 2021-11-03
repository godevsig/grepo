package pidinfo

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	ef "github.com/godevsig/grepo/lib-base/efuncs"
	et "github.com/godevsig/grepo/lib-base/etypes"
)

// CheckLevles controls if we need to check threads and/or children
const (
	CheckSingle  = 1 << 0 // check the process
	CheckThread  = 1 << 1 // check threads of the process
	CheckChild   = 1 << 2 // check children of the process
	CheckPss     = 1 << 3 // check PSS mem, high CPU overhead, often needs root privilege
	CheckAllTask = CheckSingle | CheckThread | CheckChild
)

var (
	pageSize uint64 = uint64(os.Getpagesize())
	//starts with most normal case: 100
	userHz float64 = 100
)

type cpuTick struct {
	user      uint64
	nice      uint64
	system    uint64
	idle      uint64
	iowait    uint64
	irq       uint64
	softirq   uint64
	steal     uint64
	guest     uint64
	guestnice uint64
	totalTick uint64
}

type memInfo struct {
	total        uint64
	cache        uint64
	sreclaimable uint64
	free         uint64
	buffers      uint64
	shmem        uint64
	used         uint64
	available    uint64
}

type cpuTickInfo struct {
	cpuTick
	perCPUTick []cpuTick
	numCPU     uint
	timestamp  time.Time
}

// SysInfo includes overall system tick information
type SysInfo struct {
	cpuTickLast *cpuTickInfo
	cpuTickCurr *cpuTickInfo
	mem         *memInfo
}

type pidTick struct {
	uTick uint64
	sTick uint64
}

// TidInfo represents a thread
type TidInfo struct {
	isProcess bool
	cnt       int64 // access counter
	tid       int   // tid is the thread's pid
	tgid      int   // tgid is the pid of the process that the thread belongs to
	tidstr    string
	tgidstr   string
	comm      string
	ppid      int
	priority  int
	rss       uint64
	checkPss  bool
	pss       uint64 // pss in smaps, in kB
	tickLast  *pidTick
	tickCurr  *pidTick
	timeLast  time.Time
	timeCurr  time.Time
}

// PidInfo has infomation extracted from corresponding procfs
type PidInfo struct {
	TidInfo
	ppid       int
	pid        int
	pidstr     string
	checkLevel int
	cmdline    string
	tree       *pidTree
	threads    *et.IntMap // threads in /proc/pid/task, map[int]*TidInfo
	children   *et.IntMap // fist level children, map[int]*PidInfo
}

func getppid(pid int) int {
	buf, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return -1
	}

	r := bytes.LastIndex(buf, []byte(")"))
	if r < 0 || len(buf) < r+12 {
		return -1
	}

	fields := bytes.Fields(buf[r+2 : r+12])
	if len(fields) < 2 {
		return -1
	}
	ppid, err := strconv.ParseInt(string(fields[1]), 10, 64)
	if err != nil {
		return -1
	}
	return int(ppid)
}

type pidTree struct {
	oPids      *et.IntMap // hold old allPidMap
	pid, ppid  int
	childTrees *et.IntMap // map[int]*pidTree
}

type treeInfo struct {
	tree      *pidTree
	pidTreeCh chan *pidTree
}

var treeInfoCh chan *treeInfo = make(chan *treeInfo)

func pidTreeUpdater() {
	//map[int]int
	nPids := et.NewIntMap(0)
	oPids := nPids

	allPids := func() []int {
		f, err := os.Open("/proc")
		if err != nil {
			panic(err)
		}
		defer f.Close()

		names, err := f.Readdirnames(-1)
		if err != nil {
			panic(err)
		}

		pids := make([]int, 0, 128)
		for _, name := range names {
			pid, err := strconv.Atoi(name)
			if err != nil {
				continue
			}
			pids = append(pids, pid)
		}

		return pids
	}

	//return map[pid]ppid
	allPidMap := func() *et.IntMap {
		pids := allPids()
		pidmap := et.NewIntMap(len(pids))
		for _, pid := range pids {
			ppid, has := oPids.Get(pid)
			if !has || ppid == -1 {
				ppid = getppid(pid)
			}
			pidmap.Set(pid, ppid)
		}

		return pidmap
	}

	//return the sequence of the hierarchy in tree
	//return nil if pid is not in the tree
	hierarchySeq := func(pt *pidTree, pid int, pids *et.IntMap) []int {
		seq := make([]int, 0, 16)
		test := pid
		for {
			ppid, has := pids.Get(test)
			if !has || ppid == -1 {
				return nil
			}
			seq = append(seq, test)
			if ppid == pt.pid {
				return seq
			}
			test = ppid.(int)
		}
	}

	//insert tree into tree
	treeInsert := func(pt *pidTree, pid int) {
		if pid == pt.pid {
			ppid, _ := nPids.Get(pid)
			pt.ppid = ppid.(int)
			return
		}

		seq := hierarchySeq(pt, pid, nPids)
		//pid is not in the tree
		if seq == nil {
			return
		}

		iterPt := pt
		//in reverse order, oldest ppid first
		for n := len(seq) - 1; n >= 0; n-- {
			pid := seq[n]
			cpt, has := iterPt.childTrees.Get(pid)
			//ppid has not been inserted yet
			if !has {
				cpt = &pidTree{nil, pid, iterPt.pid, et.NewIntMap(0)}
				iterPt.childTrees.Set(pid, cpt)
			}
			iterPt = cpt.(*pidTree)
		}
	}

	//delete tree from tree
	treeDelete := func(pt *pidTree, pid int) {
		if pid == pt.pid {
			pt = nil
			return
		}

		seq := hierarchySeq(pt, pid, pt.oPids)
		//pid is not in the tree
		if seq == nil {
			return
		}

		iterPt := pt
		//in reverse order, oldest ppid first
		for n := len(seq) - 1; n >= 0; n-- {
			pid := seq[n]
			cpt, has := iterPt.childTrees.Get(pid)
			//the tree that the pid belonged to has already been removed
			if !has {
				return
			}
			if n == 0 {
				iterPt.childTrees.Del(pid)
			}
			iterPt = cpt.(*pidTree)
		}

		//declare the func variable first to call it recursively
		var clearTree func(pt *pidTree)
		clearTree = func(pt *pidTree) {
			for _, cpid := range pt.childTrees.Keys() {
				if nPids.Has(cpid) {
					nPids.Set(cpid, -1)
				}
				cpt, _ := pt.childTrees.Get(cpid)
				clearTree(cpt.(*pidTree))
			}
		}

		clearTree(iterPt)
	}

	rebuildPidTree := func(pt *pidTree) *pidTree {
		if pt.pid == -1 {
			pids := allPids()
			childTrees := et.NewIntMap(len(pids))
			for _, pid := range pids {
				cpt := &pidTree{nil, pid, -1, et.NewIntMap(0)}
				childTrees.Set(pid, cpt)
			}
			pt.childTrees = childTrees
			return pt
		}

		nPids = allPidMap()
		//check all old pids
		for _, pid := range pt.oPids.Keys() {
			//a pid was gone since last update
			if !nPids.Has(pid) {
				treeDelete(pt, pid)
			}
		}

		//check all new pids
		for _, pid := range nPids.Keys() {
			has := pt.oPids.Has(pid)
			ppid, _ := nPids.Get(pid)
			if has && ppid != -1 {
				continue
			}
			//a new pid has been created since last update
			//or the pid's hierarchy has been changed
			if ppid == -1 {
				nPids.Set(pid, getppid(pid))
			}
			treeInsert(pt, pid)
		}

		oPids = nPids
		pt.oPids = oPids
		return pt
	}

	for {
		select {
		case treeinfo := <-treeInfoCh:
			treeinfo.pidTreeCh <- rebuildPidTree(treeinfo.tree)
		}
	}
}

func hzUpdater() {
	var tLastSys, tCurrSys *cpuTickInfo
	for {
		var cti cpuTickInfo
		if err := cti.update(); err != nil {
			panic(err)
		}
		tLastSys = tCurrSys
		tCurrSys = &cti

		if tLastSys != nil && tCurrSys != nil {
			tickdiff := float64(tCurrSys.totalTick-tLastSys.totalTick) / float64(tCurrSys.numCPU)
			tdiff := tCurrSys.timestamp.Sub(tLastSys.timestamp).Seconds()
			ef.AtomicStoreFloat64(&userHz, tickdiff/tdiff)
		}

		time.Sleep(time.Second)
	}
}

func (cti *cpuTickInfo) update() error {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return ef.ErrorHere(err)
	}
	defer f.Close()

	var b bytes.Buffer
	_, err = b.ReadFrom(f)
	if err != nil {
		return ef.ErrorHere(err)
	}

	line, err := b.ReadString('\n')
	if err != nil {
		return ef.ErrorHere(err)
	}
	cti.timestamp = time.Now()

	var ignore string
	if _, err := fmt.Sscan(line,
		&ignore,
		&cti.user,
		&cti.nice,
		&cti.system,
		&cti.idle,
		&cti.iowait,
		&cti.irq,
		&cti.softirq,
		&cti.steal,
		&cti.guest,
		&cti.guestnice); err == nil {
		cti.totalTick = cti.user + cti.nice + cti.system + cti.idle + cti.irq + cti.softirq + cti.steal + cti.guest + cti.guestnice
	} else {
		return ef.ErrorHere(err)
	}

	cti.numCPU = 0
	cti.perCPUTick = make([]cpuTick, 0, 4)
	for {
		line, err := b.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return ef.ErrorHere(err)
		}
		if strings.Contains(line, "cpu") {
			cti.numCPU++
			var ct cpuTick
			if _, err := fmt.Sscan(line,
				&ignore,
				&ct.user,
				&ct.nice,
				&ct.system,
				&ct.idle,
				&ct.iowait,
				&ct.irq,
				&ct.softirq,
				&ct.steal,
				&ct.guest,
				&ct.guestnice); err == nil {
				ct.totalTick = ct.user + ct.nice + ct.system + ct.idle + ct.irq + ct.softirq + ct.steal + ct.guest + ct.guestnice
			} else {
				return ef.ErrorHere(err)
			}
			cti.perCPUTick = append(cti.perCPUTick, ct)
		}
	}

	return nil
}

var hasSmapsRollup bool
var pssSkipn1, pssSkipn2 int

func init() {
	pid := os.Getpid()
	path := fmt.Sprintf("/proc/%d/smaps_rollup", pid)

	if _, err := os.Stat(path); err == nil {
		hasSmapsRollup = true
		return
	}

	path = fmt.Sprintf("/proc/%d/smaps", pid)
	buf, err := os.ReadFile(path)
	if err != nil {
		return
	}

	i := bytes.Index(buf, []byte("\nPss:"))
	if i == -1 {
		return
	}

	line1 := bytes.Count(buf[:i+1], []byte("\n"))
	buf = buf[i+1:]
	width := bytes.Index(buf, []byte("\n"))
	if width == -1 {
		return
	}
	j := bytes.Index(buf, []byte("\nVmFlags:"))
	if j == -1 {
		return
	}
	line2 := bytes.Count(buf[:j+1], []byte("\n"))

	width++
	pssSkipn1 = line1 * width
	pssSkipn2 = line2 * width
	return
}

func pss(pid string) (pssSize uint64) {
	if hasSmapsRollup {
		buf, err := os.ReadFile("/proc/" + pid + "/smaps_rollup")
		if err == nil {
			i := bytes.Index(buf, []byte("\nPss:"))
			if i != -1 {
				buf = buf[i+1:]
				size, _ := strconv.ParseUint(string(bytes.TrimSpace(buf[4:24])), 10, 64)
				//fmt.Fprintln(os.Stderr, string(bytes.TrimSpace(buf[4:24])), size)
				pssSize = size
				return
			}
		}
	}

	if pssSkipn1 == 0 || pssSkipn2 == 0 {
		return
	}
	buf, err := os.ReadFile("/proc/" + pid + "/smaps")
	if err != nil {
		return
	}
	for len(buf) >= pssSkipn1 {
		buf = buf[pssSkipn1:]
		i := bytes.Index(buf, []byte("\nPss:"))
		if i == -1 {
			break
		}
		buf = buf[i+1:]
		size, _ := strconv.ParseUint(string(bytes.TrimSpace(buf[4:24])), 10, 64)
		//fmt.Fprintln(os.Stderr, string(bytes.TrimSpace(buf[4:24])), size)
		pssSize += size
		if len(buf) >= pssSkipn2 {
			buf = buf[pssSkipn2:]
		}
	}
	return
}

func (ti *TidInfo) updateStat() error {
	ti.cnt++
	var path string
	if ti.isProcess {
		path = "/proc/" + ti.tgidstr + "/stat"
	} else {
		path = "/proc/" + ti.tgidstr + "/task/" + ti.tidstr + "/stat"
	}

	buf, err := os.ReadFile(path)
	if err != nil {
		return ef.ErrorHere(err)
	}

	now := time.Now()
	ti.timeLast = ti.timeCurr
	ti.timeCurr = now

	r := bytes.LastIndex(buf, []byte(")"))
	if r < 0 {
		return ef.ErrorHere(err)
	}
	strs := strings.Fields(string(buf[r+2:]))

	ppid, _ := strconv.ParseInt(strs[1], 10, 64)
	ti.ppid = int(ppid)

	var pt pidTick
	pt.uTick, _ = strconv.ParseUint(strs[11], 10, 64)
	pt.sTick, _ = strconv.ParseUint(strs[12], 10, 64)

	ti.tickLast = ti.tickCurr
	ti.tickCurr = &pt

	priority, _ := strconv.ParseInt(strs[15], 10, 64)
	ti.priority = int(priority)

	if ti.isProcess {
		n, _ := strconv.ParseUint(strs[21], 10, 64)
		ti.rss = n * pageSize
		if ti.checkPss && (ti.cnt-1)&15 == 0 { // every 16 invocations
			ti.pss = pss(ti.tgidstr)
		}
	}

	return nil
}

func (ti *TidInfo) init() error {
	ti.tidstr = strconv.Itoa(ti.tid)
	ti.tgidstr = strconv.Itoa(ti.tgid)
	if buf, err := os.ReadFile("/proc/" + ti.tidstr + "/comm"); err == nil {
		ti.comm = strings.TrimSpace(string(buf))
	} else {
		return ef.ErrorHere(err)
	}

	return nil
}

// CPUpercent calculates latest CPU utilization of the process,
// returns user space and kernel space cpu percent.
func (ti *TidInfo) CPUpercent() (float64, float64) {
	if ti.tickCurr == nil || ti.tickLast == nil {
		return 0, 0
	}

	udiff := float64(ti.tickCurr.uTick-ti.tickLast.uTick) * 100
	sdiff := float64(ti.tickCurr.sTick-ti.tickLast.sTick) * 100
	hz := ef.AtomicLoadFloat64(&userHz)
	sysdiff := ti.timeCurr.Sub(ti.timeLast).Seconds() * hz
	return udiff / sysdiff, sdiff / sysdiff
}

// Comm returns pid's name
func (ti *TidInfo) Comm() string {
	return ti.comm
}

// Priority returns process priority
func (ti *TidInfo) Priority() int {
	return ti.priority
}

// Rss returns process rss in KB
func (ti *TidInfo) Rss() uint64 {
	return ti.rss / 1024
}

// Pss returns process rss in KB
func (ti *TidInfo) Pss() uint64 {
	return ti.pss
}

// Tid returns thread id
func (ti *TidInfo) Tid() int {
	return ti.tid
}

func newTidInfo(tid int, pi *PidInfo) (*TidInfo, error) {
	var ti TidInfo
	ti.tid = tid
	ti.tgid = pi.pid
	ti.isProcess = false

	if err := ti.init(); err != nil {
		return nil, ef.ErrorHere(err)
	}

	if err := ti.updateStat(); err != nil {
		return nil, ef.ErrorHere(err)
	}

	return &ti, nil
}

func (pi *PidInfo) init() error {
	pi.tid = pi.pid
	pi.tgid = pi.pid
	pi.isProcess = true
	if err := pi.TidInfo.init(); err != nil {
		return ef.ErrorHere(err)
	}

	if buf, err := os.ReadFile("/proc/" + pi.pidstr + "/cmdline"); err == nil {
		strs := strings.Split(string(bytes.TrimRight(buf, string("\x00"))), string(byte(0)))
		pi.cmdline = strings.Join(strs, " ")
	} else {
		return ef.ErrorHere(err)
	}

	return nil
}

func newPidInfo(pid int, ppi *PidInfo) (*PidInfo, error) {
	//fmt.Println("Creating pidinfo for", pid)
	var pi PidInfo
	pi.pid = pid
	pi.pidstr = strconv.Itoa(pi.pid)
	pi.ppid = ppi.pid
	pi.checkLevel = ppi.checkLevel
	if pi.checkLevel&CheckPss == CheckPss {
		pi.checkPss = true
	}

	if _, err := os.Stat("/proc/" + pi.pidstr); err != nil {
		return nil, ef.ErrorHere(err)
	}

	if err := pi.init(); err != nil {
		return nil, ef.ErrorHere(err)
	}

	// the lead pid
	if ppi.pid == -99 {
		pi.tree = &pidTree{et.NewIntMap(0), pi.pid, pi.ppid, et.NewIntMap(0)}
		if pi.checkLevel&CheckChild == CheckChild {
			pi.rebuildTree()
		}
	} else {
		icpt, _ := ppi.tree.childTrees.Get(pid)
		pi.tree = icpt.(*pidTree)
	}

	if pi.checkLevel&CheckThread == CheckThread {
		pi.threads = et.NewIntMap(0)
	}
	if pi.checkLevel&CheckChild == CheckChild {
		pi.children = et.NewIntMap(0)
	}

	if err := pi.update(); err != nil {
		return nil, ef.ErrorHere(err)
	}

	return &pi, nil
}

var once sync.Once

// NewPidInfo returns a PidInfo of the pid
// If pid = 0, returns self info
// If pid < 0, returns all pids info
// checkLevel is a flag that may contain CheckSingle CheckThread and/or CheckChild
func NewPidInfo(pid int, checkLevel int) (*PidInfo, error) {
	checkLevel &= CheckSingle | CheckThread | CheckChild | CheckPss
	if pid == 0 {
		pid = os.Getpid()
	}
	if pid < 0 {
		pid = -1
	}

	ppi := &PidInfo{}
	ppi.pid = -99
	ppi.checkLevel = checkLevel

	//-1 is a special case in pidTreeUpdater
	if pid == -1 {
		ppi.checkLevel |= CheckChild
	}

	once.Do(func() {
		go hzUpdater()
		if ppi.checkLevel&CheckChild == CheckChild {
			go pidTreeUpdater()
		}
	})

	if pid == -1 {
		pi := &PidInfo{}
		pi.pid = -1
		pi.ppid = ppi.pid
		pi.checkLevel = ppi.checkLevel
		pi.tree = &pidTree{et.NewIntMap(0), pi.pid, pi.ppid, et.NewIntMap(0)}
		pi.rebuildTree()
		pi.children = et.NewIntMap(0)
		pi.updateChildren()
		return pi, nil
	}

	return newPidInfo(pid, ppi)
}

// NewSysInfo returns overall system cpu usage info including idle, and more...
func NewSysInfo() (*SysInfo, error) {
	si := &SysInfo{}
	return si, si.Update()
}

func (mi *memInfo) update() error {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return ef.ErrorHere(err)
	}
	defer f.Close()

	r := bufio.NewReader(f)
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			break
		}

		fields := strings.Split(line, ":")
		if len(fields) != 2 {
			continue
		}

		key := strings.TrimSpace(fields[0])
		value := strings.TrimSpace(fields[1])
		value = strings.Replace(value, " kB", "", -1)

		switch key {
		case "MemTotal":
			t, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return err
			}
			mi.total = t
		case "MemAvailable":
			t, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return err
			}
			mi.available = t
		case "Cached":
			t, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return err
			}
			mi.cache = t
		case "SReclaimable":
			t, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return err
			}
			mi.sreclaimable = t
		case "MemFree":
			t, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return err
			}
			mi.free = t
		case "Buffers":
			t, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return err
			}
			mi.buffers = t
		case "Shmem":
			t, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return err
			}
			mi.shmem = t
		}

	}
	mi.cache = mi.cache + mi.sreclaimable
	mi.used = mi.total - mi.free - mi.cache - mi.buffers
	return nil
}

// GetMemInfo returns a slice of map[string]uint64, representing system memory info in KB.
func (si *SysInfo) GetMemInfo() (map[string]uint64, error) {
	mem := make(map[string]uint64)
	mem["[available]"] = si.mem.available
	mem["[cache]"] = si.mem.cache
	mem["[free]"] = si.mem.free
	mem["[buffers]"] = si.mem.buffers
	mem["[shmem]"] = si.mem.shmem
	mem["[used]"] = si.mem.used

	return mem, nil
}

// Update updates SysInfo with the latest status
func (si *SysInfo) Update() error {
	var cti cpuTickInfo
	var mem memInfo
	if err := cti.update(); err != nil {
		return ef.ErrorHere(err)
	}
	if err := mem.update(); err != nil {
		return ef.ErrorHere(err)
	}
	si.cpuTickLast = si.cpuTickCurr
	si.cpuTickCurr = &cti
	si.mem = &mem
	return nil
}

// NumCPU returns the number of CPU cores
func (si *SysInfo) NumCPU() uint {
	return si.cpuTickCurr.numCPU
}

// SysPercent returns a slice of map[string]float64, representing overall per CPU's percentage data,
// which includes user, nice, system, idle, iowait, irq, softirq, steal, guest and guest_nice.
func (si *SysInfo) SysPercent() ([]map[string]float64, error) {
	if si.cpuTickCurr.numCPU != si.cpuTickLast.numCPU {
		return nil, errors.New("CPU offline detected")
	}

	sysPerCPU := make([]map[string]float64, si.cpuTickCurr.numCPU)
	sysdiff := si.cpuTickCurr.timestamp.Sub(si.cpuTickLast.timestamp).Seconds() * float64(userHz)
	for i := uint(0); i < si.cpuTickCurr.numCPU; i++ {
		currCt := si.cpuTickCurr.perCPUTick[i]
		lastCt := si.cpuTickLast.perCPUTick[i]

		sys := make(map[string]float64, 15)
		prefix := fmt.Sprintf("[CPU%d-", i)
		sys[prefix+"user]"] = float64(currCt.user-lastCt.user) * 100 / sysdiff
		sys[prefix+"nice]"] = float64(currCt.nice-lastCt.nice) * 100 / sysdiff
		sys[prefix+"system]"] = float64(currCt.system-lastCt.system) * 100 / sysdiff
		sys[prefix+"idle]"] = float64(currCt.idle-lastCt.idle) * 100 / sysdiff
		iowaitTick := 0.0
		if diff := currCt.iowait - lastCt.iowait; diff > 0 {
			iowaitTick = float64(diff)
		}
		sys[prefix+"iowait]"] = iowaitTick * 100 / sysdiff
		sys[prefix+"irq]"] = float64(currCt.irq-lastCt.irq) * 100 / sysdiff
		sys[prefix+"softirq]"] = float64(currCt.softirq-lastCt.softirq) * 100 / sysdiff
		sys[prefix+"steal]"] = float64(currCt.steal-lastCt.steal) * 100 / sysdiff
		sys[prefix+"guest]"] = float64(currCt.guest-lastCt.guest) * 100 / sysdiff
		sys[prefix+"guestnice]"] = float64(currCt.guestnice-lastCt.guestnice) * 100 / sysdiff

		sysPerCPU[i] = sys
	}

	return sysPerCPU, nil
}

// SysPercentBy returns the CPU statistics specified by core and name:
// core is the core number, -1 means all; name can be one of
// user, nice, system, idle, iowait, irq, softirq, steal, guest or guestnice.
// Return 0 if the name is not in above list.
// Panic if core is out of range.
func (si *SysInfo) SysPercentBy(core int, name string) float64 {
	var ctCurr, ctLast *cpuTick
	if core < 0 {
		ctCurr = &si.cpuTickCurr.cpuTick
		ctLast = &si.cpuTickLast.cpuTick
	} else {
		ctCurr = &si.cpuTickCurr.perCPUTick[core]
		ctLast = &si.cpuTickLast.perCPUTick[core]
	}
	tickDiff := float64(ctCurr.totalTick - ctLast.totalTick)

	switch name {
	case "user":
		return float64(ctCurr.user-ctLast.user) * 100 / tickDiff
	case "nice":
		return float64(ctCurr.nice-ctLast.nice) * 100 / tickDiff
	case "system":
		return float64(ctCurr.system-ctLast.system) * 100 / tickDiff
	case "idle":
		return float64(ctCurr.idle-ctLast.idle) * 100 / tickDiff
	case "iowait":
		return float64(ctCurr.iowait-ctLast.iowait) * 100 / tickDiff
	case "irq":
		return float64(ctCurr.irq-ctLast.irq) * 100 / tickDiff
	case "softirq":
		return float64(ctCurr.softirq-ctLast.softirq) * 100 / tickDiff
	case "steal":
		return float64(ctCurr.steal-ctLast.steal) * 100 / tickDiff
	case "guest":
		return float64(ctCurr.guest-ctLast.guest) * 100 / tickDiff
	case "guestnice":
		return float64(ctCurr.guestnice-ctLast.guestnice) * 100 / tickDiff
	default:
		return 0
	}
}

func (pi *PidInfo) updateThreads() error {
	f, err := os.Open("/proc/" + pi.pidstr + "/task")
	if err != nil {
		return ef.ErrorHere(err)
	}
	defer f.Close()

	tidstrs, err := f.Readdirnames(-1)
	if err != nil {
		return ef.ErrorHere(err)
	}

	nthreads := len(tidstrs)
	// only if the process has more than 1 threads
	if nthreads > 1 {
		threads := et.NewIntMap(nthreads)
		//use reverse order of tidstrs
		for i := nthreads - 1; i >= 0; i-- {
			tidstr := tidstrs[i]
			tid, _ := strconv.Atoi(tidstr)
			iti, has := pi.threads.Get(tid)
			var err error
			if !has {
				iti, err = newTidInfo(tid, pi)
			} else {
				ti := iti.(*TidInfo)
				err = ti.updateStat()
			}
			if err == nil {
				//refill to the new map
				threads.Set(tid, iti)
			}
		}
		//use new refilled map
		pi.threads = threads
	}

	return nil
}

func (pi *PidInfo) updateChildren() error {
	cpids := pi.tree.childTrees.Keys()
	children := et.NewIntMap(len(cpids))
	for _, cpid := range cpids {
		icpi, has := pi.children.Get(cpid)
		var err error
		if !has {
			icpi, err = newPidInfo(cpid, pi)
		} else {
			cpi := icpi.(*PidInfo)
			itree, _ := pi.tree.childTrees.Get(cpid)
			cpi.tree = itree.(*pidTree)
			//next level check
			err = cpi.update()
		}
		if err == nil {
			//refill to the new map
			children.Set(cpid, icpi)
		}
	}
	//use new refilled map
	pi.children = children

	return nil
}

func (pi *PidInfo) update() error {
	if err := pi.updateStat(); err != nil {
		return ef.ErrorHere(err)
	}

	if pi.checkLevel&CheckThread == CheckThread {
		if err := pi.updateThreads(); err != nil {
			return ef.ErrorHere(err)
		}
	}

	if pi.checkLevel&CheckChild == CheckChild {
		if err := pi.updateChildren(); err != nil {
			return ef.ErrorHere(err)
		}
	}

	return nil
}

func (pi *PidInfo) rebuildTree() {
	treeinfo := &treeInfo{pi.tree, make(chan *pidTree)}
	treeInfoCh <- treeinfo
	pi.tree = <-treeinfo.pidTreeCh
}

// Update check the tree for rebuilding and updates the stat of the pid tree
func (pi *PidInfo) Update() error {
	if pi.checkLevel&CheckChild == CheckChild {
		pi.rebuildTree()
	}

	if pi.pid == -1 {
		pi.updateChildren()
		return nil
	}

	if err := pi.update(); err != nil {
		return ef.ErrorHere(err)
	}
	return nil
}

// Pid returns process id
func (pi *PidInfo) Pid() int {
	return pi.pid
}

// CheckLevel returns process check level
func (pi *PidInfo) CheckLevel() int {
	return pi.checkLevel
}

// Cmdline returns process cmdline
func (pi *PidInfo) Cmdline() string {
	return pi.cmdline
}

// Children returns first level children nodes
func (pi *PidInfo) Children() []*PidInfo {
	if pi.checkLevel&CheckChild != CheckChild {
		return nil
	}
	children := make([]*PidInfo, 0, pi.children.Len())

	for _, pid := range pi.children.Keys() {
		childpi, _ := pi.children.Get(pid)
		children = append(children, childpi.(*PidInfo))
	}

	return children
}

// Threads returns threads of the process
func (pi *PidInfo) Threads() []*TidInfo {
	if pi.checkLevel&CheckThread != CheckThread {
		return nil
	}
	if pi.pid == -1 {
		return nil
	}
	threads := make([]*TidInfo, 0, pi.threads.Len())

	for _, tid := range pi.threads.Keys() {
		thrdi, _ := pi.threads.Get(tid)
		threads = append(threads, thrdi.(*TidInfo))
	}

	return threads
}

// ProcessStat is process statistics.
type ProcessStat struct {
	Pid  int
	Name string
	Ucpu float64
	Scpu float64
	Mem  uint64 // in KB
}

// ProcessStatAll returns the statistics of the pid and all its children/threads
// if the corresponding check level enabled.
func (pi *PidInfo) ProcessStatAll() (all []ProcessStat) {
	var appendProcessStat func(pi *PidInfo)
	appendProcessStat = func(pi *PidInfo) {
		if pi.pid != -1 {
			var mem uint64
			if pi.checkPss {
				mem = pi.Pss()
			} else {
				mem = pi.Rss()
			}

			if pi.checkLevel&CheckThread == CheckThread {
				threads := pi.Threads()
				for _, ti := range threads {
					procstat := ProcessStat{}
					procstat.Pid = ti.Tid()
					procstat.Name = ti.Comm()
					procstat.Ucpu, procstat.Scpu = ti.CPUpercent()
					procstat.Mem = mem
					all = append(all, procstat)
				}
			} else {
				procstat := ProcessStat{}
				procstat.Pid = pi.Pid()
				procstat.Name = pi.Comm()
				procstat.Ucpu, procstat.Scpu = pi.CPUpercent()
				procstat.Mem = mem
				all = append(all, procstat)
			}
		}

		if pi.checkLevel&CheckChild == CheckChild {
			for _, cpi := range pi.Children() {
				appendProcessStat(cpi)
			}
		}
	}

	appendProcessStat(pi)
	return
}

// Pidof finds the process id(s) of the named programs
func Pidof(pNames ...string) ([]int, error) {
	if len(pNames) == 0 {
		return nil, errors.New("wrong args number")
	}

	var foundPids []int
	foundPidsSet := make(map[string][]int)

	f, err := os.Open("/proc")
	if err != nil {
		return nil, ef.ErrorHere(err)
	}
	defer f.Close()

	names, err := f.Readdirnames(-1)
	if err != nil {
		return nil, ef.ErrorHere(err)
	}

	for _, name := range names {
		pid, err := strconv.Atoi(name)
		if err != nil {
			continue
		}

		buf, err := os.ReadFile("/proc/" + name + "/comm")
		if err != nil {
			continue
		}

		str := string(bytes.TrimSpace(buf))
		for _, pname := range pNames {
			if pname == str {
				foundPidsSet[pname] = append(foundPidsSet[pname], pid)
			}
		}
	}

	for _, pname := range pNames {
		foundPids = append(foundPids, foundPidsSet[pname]...)
	}

	return foundPids, nil
}
