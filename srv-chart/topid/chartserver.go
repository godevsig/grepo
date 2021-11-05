package topid

import (
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "embed" //embed: read file

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/templates"
	"github.com/go-echarts/go-echarts/v2/types"
	"github.com/godevsig/grepo/lib-sys/log"
	"github.com/gorilla/mux"
)

type pair struct {
	key   string
	value float32
}

type list []pair

type processRecords struct {
	time   []string
	cpu    map[string]([]float32)
	mem    map[string]([]float32)
	cpuavg map[string]float32
	memavg map[string]float32
	cpumax map[string]float32
	memmax map[string]float32
}

var (
	//go:embed echarts/echarts.min.js
	echarts string
	//go:embed echarts/themes/shine.js
	themes string
)

type filter struct {
	cpuavg float32
	cpumax float32
	memavg float32
	memmax float32
}

type chartServer struct {
	ip        string
	chartport string
	fileport  string
	dir       string
	filter    *filter
	lg        *log.Logger
	srv       *http.Server
}

func newRecords() *processRecords {
	return &processRecords{
		cpu:    make(map[string]([]float32)),
		mem:    make(map[string]([]float32)),
		cpuavg: make(map[string]float32),
		memavg: make(map[string]float32),
		cpumax: make(map[string]float32),
		memmax: make(map[string]float32),
	}
}

func floatConv(value float32) float32 {
	return float32(math.Round(float64(value)*100) / 100)
}

func string2float32(value string) float32 {
	tmp, _ := strconv.ParseFloat(value, 32)
	return float32(tmp)
}

func maxAndAvg(series []float32) (max, avg float32) {
	if len := len(series); len != 0 {
		var sum float32 = 0.0
		for _, v := range series {
			if v > max {
				max = v
			}
			sum += v
		}
		avg = sum / float32(len)
	}
	return
}

func (p list) Len() int           { return len(p) }
func (p list) Less(i, j int) bool { return p[i].value < p[j].value }
func (p list) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func rank(avg map[string]float32) list {
	l := make(list, len(avg))
	i := 0
	for k, v := range avg {
		l[i] = pair{k, v}
		i++
	}
	sort.Sort(sort.Reverse(l))
	return l
}

//ParseFile parse encode record file
func ParseFile(filename string) error {
	in, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(filename + ".parsed")
	if err != nil {
		return err
	}
	defer out.Close()

	decoder := gob.NewDecoder(in)
	for err != io.EOF {
		var buf = pRecord{}
		err = decoder.Decode(&buf)
		if err != nil {
			continue
		}
		if len(buf.Processes) != 0 {
			line := fmt.Sprintf("======%v, processinfo %v\n", time.Unix(buf.Timestamp, 0).Format("15:04:05"), buf.Processes)
			out.WriteString(line)
		}
	}
	return nil
}

func (prs *processRecords) sortMap(mode string, m map[string]([]float32), f func(k string, v []float32)) {
	l := make(list, len(prs.cpuavg))
	switch mode {
	case "cpu":
		l = rank(prs.cpuavg)
	case "mem":
		l = rank(prs.memavg)
	}

	for _, k := range l {
		f(k.key, m[k.key])
	}
}

func (prs *processRecords) analysis(filename string, filter *filter) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	decoder := gob.NewDecoder(f)

	for err != io.EOF {
		var buf = pRecord{}
		err = decoder.Decode(&buf)
		if err != nil {
			continue
		}

		if len(buf.Processes) != 0 {
			prs.time = append(prs.time, time.Unix(buf.Timestamp, 0).Format("15:04:05"))
			for _, b := range buf.Processes {
				var name string
				if strings.Contains(b.Name, "[") {
					name = b.Name
				} else {
					name = fmt.Sprintf("%v-%v", b.Name, b.Pid)
				}
				if _, ok := prs.cpu[name]; !ok {
					reserved := make([]float32, len(prs.time)-1)
					prs.cpu[name] = append(prs.cpu[name], reserved...)
					prs.mem[name] = append(prs.mem[name], reserved...)
				}
				prs.cpu[name] = append(prs.cpu[name], floatConv(b.Ucpu+b.Scpu))
				prs.mem[name] = append(prs.mem[name], float32(b.Mem/1024))
			}
		}
	}

	for k, v := range prs.cpu {
		prs.cpumax[k], prs.cpuavg[k] = maxAndAvg(v)
		if prs.cpuavg[k] <= filter.cpuavg && prs.cpumax[k] <= filter.cpumax {
			delete(prs.cpu, k)
			delete(prs.cpuavg, k)
			delete(prs.cpumax, k)
		} else {
			if len(v) < len(prs.time) {
				prs.cpu[k] = append(prs.cpu[k], make([]float32, len(prs.time)-len(v))...)
			}
		}

	}

	for k, v := range prs.mem {
		prs.memmax[k], prs.memavg[k] = maxAndAvg(v)
		if prs.memavg[k] <= filter.memavg && prs.memmax[k] <= filter.memmax {
			delete(prs.mem, k)
			delete(prs.memavg, k)
			delete(prs.memmax, k)
		} else {
			if len(v) < len(prs.time) {
				prs.mem[k] = append(prs.mem[k], make([]float32, len(prs.time)-len(v))...)
			}
		}
	}

	return nil
}

func (prs *processRecords) lineCPU() *charts.Line {
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "CPU Usage",
			Left:  "560",
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: "Percent",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:    true,
			Trigger: "axis",
		}),
		charts.WithInitializationOpts(opts.Initialization{
			Theme:  types.ThemeShine,
			Width:  "1400px",
			Height: "350px",
		}),
		charts.WithDataZoomOpts(opts.DataZoom{
			Start:      0,
			End:        100,
			XAxisIndex: []int{0},
		}),
		charts.WithLegendOpts(opts.Legend{
			Show:   true,
			Type:   "scroll",
			Orient: "vertical",
			Left:   "83%",
		}),
	)

	fn := fmt.Sprintf(`var obj = {};
					for(var key in option_%s.series){
						if(option_%s.series[key].name.indexOf("[") != -1){
							obj[option_%s.series[key].name] = false;
						}
					}
					option_%s.legend.selected = obj;
					goecharts_%s.setOption(option_%s);
					document.getElementById("snapshot").onclick=function(){
						location.href=location.href+"/snapshot";
					};
					document.getElementById("info").onclick=function(){
						location.href=location.href+"/info";
					};
					document.getElementById("pieview").onclick=function(){
						this.value="PIEVIEW";
						location.href=location.href+"/pie";
					};
					document.getElementById("cpuselectall").onclick=function(){
						var flag=this.getAttribute("flag");
						var val=false;
						if(flag==1){
							val=false;
							this.setAttribute("flag",0);
							this.value="CPUON";
						}else{
							val=true;
							this.setAttribute("flag",1);
							this.value="CPUOFF";
						}
						var obj = {};
						for(var key in option_%s.series){
							obj[option_%s.series[key].name] = val;
						}
						option_%s.legend.selected = obj;
						goecharts_%s.setOption(option_%s);
					};
					document.getElementById("syscpu").onclick=function(){
						var flag=this.getAttribute("flag");
						var val=false;
						var obj = {};
						if(flag==1){
							val=false;
							this.setAttribute("flag",0);
							this.value="P-CPU";
						}else{
							val=true;
							this.setAttribute("flag",1);
							this.value="SYSCPU";
						}
						for(var key in option_%s.series){
							if(option_%s.series[key].name.indexOf("[") != -1){
								obj[option_%s.series[key].name] = !val;
							}else{
								obj[option_%s.series[key].name] = val;
							}
						}
						option_%s.legend.selected = obj;
						goecharts_%s.setOption(option_%s);
					};`, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID)
	line.AddJSFuncs(fn)

	line = line.SetXAxis(prs.time)
	prs.sortMap("cpu", prs.cpu, func(k string, v []float32) {
		items := make([]opts.LineData, 0, len(prs.time))
		for _, data := range v {
			items = append(items, opts.LineData{Value: data})
		}

		line.AddSeries(k, items)
	})
	line.SetSeriesOptions(
		charts.WithAreaStyleOpts(
			opts.AreaStyle{
				Opacity: 0.8,
			}),
		charts.WithLineChartOpts(
			opts.LineChart{
				Stack:    "stack",
				Sampling: "lttb",
			}),
	)

	return line
}

func (prs *processRecords) lineMEM() *charts.Line {
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "MEM Usage",
			Left:  "560",
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: "MB",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:    true,
			Trigger: "axis",
		}),
		charts.WithInitializationOpts(opts.Initialization{
			Theme:  types.ThemeShine,
			Width:  "1400px",
			Height: "350px",
		}),
		charts.WithDataZoomOpts(opts.DataZoom{
			Start:      0,
			End:        100,
			XAxisIndex: []int{0},
		}),
		charts.WithLegendOpts(opts.Legend{
			Show:   true,
			Type:   "scroll",
			Orient: "vertical",
			Left:   "83%",
		}),
	)

	fn := fmt.Sprintf(`var obj = {};
					for(var key in option_%s.series){
						if(option_%s.series[key].name.indexOf("[") != -1){
							obj[option_%s.series[key].name] = false;
						}
					}
					option_%s.legend.selected = obj;
					goecharts_%s.setOption(option_%s);
					document.getElementById("memselectall").onclick=function(){
						var flag=this.getAttribute("flag");
						var val=false;
						if(flag==1){
							val=false;
							this.setAttribute("flag",0);
							this.value="MEMON";
						}else{
							val=true;
							this.setAttribute("flag",1);
							this.value="MEMOFF";
						}
						var obj = {};
						for(var key in option_%s.series){
							obj[option_%s.series[key].name] = val;
						}
						option_%s.legend.selected = obj;
						goecharts_%s.setOption(option_%s);
					};
					document.getElementById("sysmem").onclick=function(){
						var flag=this.getAttribute("flag");
						var val=false;
						var obj = {};
						if(flag==1){
							val=false;
							this.setAttribute("flag",0);
							this.value="P-MEM";
						}else{
							val=true;
							this.setAttribute("flag",1);
							this.value="SYSMEM";
						}
						for(var key in option_%s.series){
							if(option_%s.series[key].name.indexOf("[") != -1){
								obj[option_%s.series[key].name] = !val;
							}else{
								obj[option_%s.series[key].name] = val;
							}
						}
						option_%s.legend.selected = obj;
						goecharts_%s.setOption(option_%s);
					};`, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID)
	line.AddJSFuncs(fn)

	line = line.SetXAxis(prs.time)
	prs.sortMap("mem", prs.mem, func(k string, v []float32) {
		items := make([]opts.LineData, 0, len(prs.time))
		for _, data := range v {
			items = append(items, opts.LineData{Value: data})
		}

		line.AddSeries(k, items)
	})
	line.SetSeriesOptions(
		charts.WithAreaStyleOpts(
			opts.AreaStyle{
				Opacity: 0.8,
			}),
		charts.WithLineChartOpts(
			opts.LineChart{
				Stack:    "stack",
				Sampling: "lttb",
			}),
	)

	return line
}

func (cs *chartServer) lineHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	tag := params["tag"]
	session := "process-" + params["session"]

	vars := r.URL.Query()
	if filterVar, ok := vars["filter"]; ok {
		filterVar = strings.Split(filterVar[0], ",")
		if len(filterVar) == 4 {
			cs.filter.cpuavg = string2float32(filterVar[0])
			cs.filter.cpumax = string2float32(filterVar[1])
			cs.filter.memavg = string2float32(filterVar[2])
			cs.filter.memmax = string2float32(filterVar[3])
		}
	}

	if tag != "" && (strings.Index(session, ".") != -1) {
		file, err := os.Open(cs.dir + tag + "/" + session)
		defer file.Close()
		if err != nil {
			http.Error(w, "File not found.", 404)
			return
		}
		io.Copy(w, file)
		return
	}

	in := fmt.Sprintf("%v/%v/%v.data", cs.dir, tag, session)

	records := newRecords()
	if err := records.analysis(in, cs.filter); err != nil {
		cs.lg.Errorln(err)
		return
	}

	page := components.NewPage()
	page.PageTitle = "Performance Analysis Tool"
	page.AddCharts(
		records.lineCPU(),
		records.lineMEM(),
	)

	page.SetLayout(components.PageFlexLayout)
	page.Render(w)
}

func (prs *processRecords) pieCPU() *charts.Pie {
	pie := charts.NewPie()
	pie.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "CPU Usage",
			Left:  "560",
		}),
		charts.WithInitializationOpts(opts.Initialization{
			Theme: types.ThemeShine,
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:    true,
			Trigger: "axis",
		}),
		charts.WithLegendOpts(opts.Legend{
			Show:   true,
			Type:   "scroll",
			Orient: "vertical",
			Right:  "0",
		}),
	)

	fn := fmt.Sprintf(`document.getElementById("snapshot").onclick=function(){
							location.href=location.href.replace("pie","snapshot");
						};
						var btn = document.getElementById("pieview");
						btn.value="LINEVIEW";
						btn.onclick=function(){
							location.href=location.href.replace("/pie","");
						};
						document.getElementById("cpuselectall").onclick=function(){
							var flag=this.getAttribute("flag");
							var val=false;
							if(flag==1){
								val=false;
								this.setAttribute("flag",0);
								this.value="CPUON";
							}else{
								val=true;
								this.setAttribute("flag",1);
								this.value="CPUOFF";
							}
							var obj = {};
							for(var key in option_%s.series){
								obj[option_%s.series[key].name] = val;
						}
						option_%s.legend.selected = obj;
						goecharts_%s.setOption(option_%s);
						}`, pie.ChartID, pie.ChartID, pie.ChartID, pie.ChartID, pie.ChartID)
	pie.AddJSFuncs(fn)

	items := make([]opts.PieData, 0)
	for k, v := range prs.cpuavg {
		items = append(items, opts.PieData{Name: k, Value: v})
	}
	pie = pie.AddSeries("cpu", items)

	return pie
}

func (prs *processRecords) pieMEM() *charts.Pie {
	pie := charts.NewPie()
	pie.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "MEMORY Usage",
			Left:  "560",
		}),
		charts.WithInitializationOpts(opts.Initialization{
			Theme: types.ThemeShine,
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:    true,
			Trigger: "axis",
		}),
		charts.WithLegendOpts(opts.Legend{
			Show:   true,
			Type:   "scroll",
			Orient: "vertical",
			Right:  "0",
		}),
	)

	fn := fmt.Sprintf(`document.getElementById("memselectall").onclick=function(){
						var flag=this.getAttribute("flag");
						var val=false;
						if(flag==1){
							val=false;
							this.setAttribute("flag",0);
							this.value="MEMON";
						}else{
							val=true;
							this.setAttribute("flag",1);
							this.value="MEMOFF";
						}
						var obj = {};
						for(var key in option_%s.series){
							obj[option_%s.series[key].name] = val;
						}
						option_%s.legend.selected = obj;
						goecharts_%s.setOption(option_%s);
					}`, pie.ChartID, pie.ChartID, pie.ChartID, pie.ChartID, pie.ChartID)
	pie.AddJSFuncs(fn)

	items := make([]opts.PieData, 0)
	for k, v := range prs.memavg {
		items = append(items, opts.PieData{Name: k, Value: v})
	}
	pie = pie.AddSeries("mem", items)

	return pie
}

func (cs *chartServer) pieHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	tag := params["tag"]
	session := "process-" + params["session"]

	vars := r.URL.Query()
	if filterVar, ok := vars["filter"]; ok {
		filterVar = strings.Split(filterVar[0], ",")
		if len(filterVar) == 4 {
			cs.filter.cpuavg = string2float32(filterVar[0])
			cs.filter.cpumax = string2float32(filterVar[1])
			cs.filter.memavg = string2float32(filterVar[2])
			cs.filter.memmax = string2float32(filterVar[3])
		}
	}

	if tag != "" && (strings.Index(session, ".") != -1) {
		file, err := os.Open(cs.dir + tag + "/" + session)
		defer file.Close()
		if err != nil {
			http.Error(w, "File not found.", 404)
			return
		}
		io.Copy(w, file)
		return
	}

	in := fmt.Sprintf("%v/%v/%v.data", cs.dir, tag, session)

	records := newRecords()
	if err := records.analysis(in, cs.filter); err != nil {
		cs.lg.Errorln(err)
		return
	}

	page := components.NewPage()
	page.PageTitle = "Performance Analysis Tool"
	page.AddCharts(
		records.pieCPU(),
		records.pieMEM(),
	)

	page.SetLayout(components.PageFlexLayout)
	page.Render(w)
}

func (cs *chartServer) infoHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	tag := params["tag"]
	session := "info-" + params["session"]

	in := fmt.Sprintf("%v/%v/%v.data", cs.dir, tag, session)
	f, err := os.Open(in)
	if err != nil {
		http.Error(w, "File not found.", 404)
		return
	}
	defer f.Close()

	info, _ := ioutil.ReadAll(f)
	w.Write([]byte(info))
}

func (cs *chartServer) snapshotHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	tag := params["tag"]
	session := "snapshot-" + params["session"]

	in := fmt.Sprintf("%v/%v/%v.data", cs.dir, tag, session)
	f, err := os.Open(in)
	if err != nil {
		http.Error(w, "File not found.", 404)
		return
	}
	defer f.Close()

	decoder := gob.NewDecoder(f)
	var data string
	for err != io.EOF {
		var buf = sRecord{}
		err = decoder.Decode(&buf)
		if err != nil {
			continue
		}
		if len(buf.Snapshot) != 0 {
			line := fmt.Sprintf("======%v, snapshot======\n%v\n", time.Unix(buf.Timestamp, 0).Format("15:04:05"), buf.Snapshot)
			data = data + line
		}
	}

	w.Write([]byte(data))
}

func (cs *chartServer) updatePageTpl() {
	templates.BaseTpl = `
				{{- define "base" }}
				<div class="container">
					<div class="item" id="{{ .ChartID }}" style="width:{{ .Initialization.Width }};height:{{ .Initialization.Height }};"></div>
				</div>

				<script type="text/javascript">
					"use strict";
					let goecharts_{{ .ChartID | safeJS }} = echarts.init(document.getElementById('{{ .ChartID | safeJS }}'), "{{ .Theme }}");
					let option_{{ .ChartID | safeJS }} = {{ .JSONNotEscaped | safeJS }};
					option_{{ .ChartID | safeJS }}.grid = {"left":50, "right":250};
					goecharts_{{ .ChartID | safeJS }}.setOption(option_{{ .ChartID | safeJS }});
					{{- range .JSFunctions.Fns }}
					{{ . | safeJS }}
					{{- end }}
				</script>
				{{ end }}
				`
	templates.PageTpl = fmt.Sprintf(`
				{{- define "page" }}
				<!DOCTYPE html>
				<html>
					{{- template "header" . }}
				<body>
				<p>&nbsp;&nbsp;🚀 <em>Performance Analysis Tool</em></p>
				<script type="text/javascript">%s</script>
				<script type="text/javascript">%s</script>
				<style> .btn { justify-content:space-around; padding-left:50px; float:left; width:150px } </style>
				<div class="btn">
					<a href="http://%s:%s/README"><input type="button" style="width:100px;height:30px;border:5px orange double;margin-top:10px" value="README"/></a>
					<a href="http://%s:%s"><input type="button" style="width:100px;height:30px;border:5px orange double;margin-top:10px" value="HISTORY"/></a>
					<input id="info" type="button" style="width:100px;height:30px;border:5px blue double;margin-top:10px"value="INFO"/>
					<input id="snapshot" type="button" style="width:100px;height:30px;border:5px blue double;margin-top:10px"value="SNAPSHOT"/>
					<input id="pieview" type="button" style="width:100px;height:30px;border:5px blue double;margin-top:10px"value="PIEVIEW"/>
					<input id="cpuselectall" type="button" style="width:100px;height:30px;border:5px blue double;margin-top:10px"value="CPUOFF" flag="1"/>
					<input id="memselectall" type="button" style="width:100px;height:30px;border:5px blue double;margin-top:10px"value="MEMOFF" flag="1"/>
					<input id="syscpu" type="button" style="width:100px;height:30px;border:5px blue double;margin-top:10px"value="SYSCPU" flag="1"/>
					<input id="sysmem" type="button" style="width:100px;height:30px;border:5px blue double;margin-top:10px"value="SYSMEM" flag="1"/>
				</div>
				<style> .box { justify-content:center; flex-wrap:wrap; float:left } </style>
				<div class="box"> {{- range .Charts }} {{ template "base" . }} {{- end }} </div>
				</body>
				</html>
				{{ end }}
				`, echarts, themes, cs.ip, cs.fileport, cs.ip, cs.chartport)
}

func newChartServer(lg *log.Logger, ip, chartport, fileport, dir string) *chartServer {
	cs := &chartServer{
		ip:        ip,
		chartport: chartport,
		fileport:  fileport,
		dir:       dir,
		lg:        lg,
		filter:    &filter{cpuavg: 1, cpumax: 10, memavg: 5, memmax: 10},
	}

	router := mux.NewRouter().StrictSlash(false)
	router.HandleFunc("/{tag}/{session}", cs.lineHandler)
	router.HandleFunc("/{tag}/{session}/info", cs.infoHandler)
	router.HandleFunc("/{tag}/{session}/pie", cs.pieHandler)
	router.HandleFunc("/{tag}/{session}/snapshot", cs.snapshotHandler)

	cs.updatePageTpl()

	cs.srv = &http.Server{
		Addr:    ":" + cs.chartport,
		Handler: router,
	}

	return cs
}

func (cs *chartServer) start() {
	cs.lg.Infof("start chart http server addr %s", cs.srv.Addr)

	if err := cs.srv.ListenAndServe(); err != http.ErrServerClosed {
		cs.lg.Errorf("chart http server ListenAndServe: %v", err)
		return
	}
}

func (cs *chartServer) stop() {
	if err := cs.srv.Shutdown(context.Background()); err != nil {
		cs.lg.Errorf("chart http server shutdown: %v", err)
	}
	cs.lg.Infoln("chart http server shutdown successfully")
}
