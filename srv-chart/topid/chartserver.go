package topid

import (
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	_ "embed" //embed: read file

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/templates"
	"github.com/go-echarts/go-echarts/v2/types"
	"github.com/gorilla/mux"
)

type pair struct {
	key   string
	value float64
}

type list []pair

type processRecords struct {
	time   []string
	cpu    map[string]([]float64)
	mem    map[string]([]float64)
	cpuavg map[string]float64
	memavg map[string]float64
	cpumax map[string]float64
	memmax map[string]float64
}

//go:embed echarts/echarts.min.js
var echarts string

//go:embed echarts/themes/shine.js
var themes string

var info string
var chartShutdown chan struct{}

func newRecords() *processRecords {
	return &processRecords{
		cpu:    make(map[string]([]float64)),
		mem:    make(map[string]([]float64)),
		cpuavg: make(map[string]float64),
		memavg: make(map[string]float64),
		cpumax: make(map[string]float64),
		memmax: make(map[string]float64),
	}
}

func floatConv(value float64) float64 {
	return math.Round(value*100) / 100
}

func maxAndAvg(series []float64) (max, avg float64) {
	if len := len(series); len != 0 {
		sum := 0.0
		for _, v := range series {
			if v > max {
				max = v
			}
			sum += v
		}
		avg = sum / float64(len)
	}
	return
}

func (p list) Len() int           { return len(p) }
func (p list) Less(i, j int) bool { return p[i].value < p[j].value }
func (p list) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func rank(avg map[string]float64) list {
	l := make(list, len(avg))
	i := 0
	for k, v := range avg {
		l[i] = pair{k, v}
		i++
	}
	sort.Sort(sort.Reverse(l))
	return l
}

//Parse parse encode record file
func Parse(filename string) {
	in, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer in.Close()

	out, err := os.Create(strings.Replace(filename, "data", "parsed", -1))
	if err != nil {
		panic(err)
	}

	decoder := gob.NewDecoder(in)
	for err != io.EOF {
		var buf = pRecord{}
		err = decoder.Decode(&buf)
		if err != nil {
			continue
		}
		if len(buf.Processes) != 0 {
			line := fmt.Sprintf(`======%v, processinfo %v\n`, time.Unix(buf.Timestamp, 0).Format("15:04:05"), buf.Processes)
			out.WriteString(line)
		}
	}
}

func (prs *processRecords) sortMap(mode string, m map[string]([]float64), f func(k string, v []float64)) {
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

func (prs *processRecords) analysis(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		fmt.Println(err)
		return
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
					for i := 0; i < len(prs.time)-2; i++ {
						prs.cpu[name] = append(prs.cpu[name], 0)
						prs.mem[name] = append(prs.mem[name], 0)
					}
				}
				prs.cpu[name] = append(prs.cpu[name], floatConv(b.Ucpu+b.Scpu))
				prs.mem[name] = append(prs.mem[name], float64(b.Mem/1024))
			}
			for k, v := range prs.cpu {
				if len(v) < len(prs.time) {
					prs.cpu[k] = append(prs.cpu[k], 0)
					prs.mem[k] = append(prs.mem[k], 0)
				}
			}
		}
	}

	for k, v := range prs.cpu {
		prs.cpumax[k], prs.cpuavg[k] = maxAndAvg(v)
		if prs.cpuavg[k] <= 0.5 && prs.cpumax[k] <= 5 {
			delete(prs.cpu, k)
			delete(prs.cpuavg, k)
			delete(prs.cpumax, k)
		}
	}

	for k, v := range prs.mem {
		prs.memmax[k], prs.memavg[k] = maxAndAvg(v)
		if prs.memavg[k] <= 1 && prs.memmax[k] <= 10 {
			delete(prs.mem, k)
			delete(prs.memavg, k)
			delete(prs.memmax, k)
		}
	}
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

	fn := fmt.Sprintf(`document.getElementById("snapshot").onclick=function(){
						location.href=location.href+"/snapshot";
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
					};`, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID)
	line.AddJSFuncs(fn)

	line = line.SetXAxis(prs.time)
	line.SetSeriesOptions(
		charts.WithLineChartOpts(opts.LineChart{
			Smooth: true,
		}))
	prs.sortMap("cpu", prs.cpu, func(k string, v []float64) {
		items := make([]opts.LineData, 0)
		for _, data := range v {
			items = append(items, opts.LineData{Value: data})
		}

		line.AddSeries(k, items).
			SetSeriesOptions(
				charts.WithAreaStyleOpts(
					opts.AreaStyle{
						Opacity: 0.8,
					}),
				charts.WithLineChartOpts(
					opts.LineChart{
						Stack: "stack",
					}),
			)
	})

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
					};`, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID, line.ChartID)
	line.AddJSFuncs(fn)

	line = line.SetXAxis(prs.time)
	line.SetSeriesOptions(
		charts.WithLineChartOpts(opts.LineChart{
			Smooth: true,
		}))
	prs.sortMap("mem", prs.mem, func(k string, v []float64) {
		items := make([]opts.LineData, 0)
		for _, data := range v {
			items = append(items, opts.LineData{Value: data})
		}

		line.AddSeries(k, items).
			SetSeriesOptions(
				charts.WithAreaStyleOpts(
					opts.AreaStyle{
						Opacity: 0.8,
					}),
				charts.WithLineChartOpts(
					opts.LineChart{
						Stack: "stack",
					}),
			)
	})

	return line
}

func lineHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	tag := params["tag"]
	session := "process-" + params["session"]

	if tag != "" && (strings.Index(session, ".") != -1) {
		file, err := os.Open(cfg.dir + tag + "/" + session)
		defer file.Close()
		if err != nil {
			http.Error(w, "File not found.", 404)
			return
		}
		io.Copy(w, file)
		return
	}

	in := fmt.Sprintf("%v/%v/%v.data", cfg.dir, tag, session)

	records := newRecords()
	records.analysis(in)

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

func pieHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	tag := params["tag"]
	session := "process-" + params["session"]

	if tag != "" && (strings.Index(session, ".") != -1) {
		file, err := os.Open(cfg.dir + tag + "/" + session)
		defer file.Close()
		if err != nil {
			http.Error(w, "File not found.", 404)
			return
		}
		io.Copy(w, file)
		return
	}

	in := fmt.Sprintf("%v/%v/%v.data", cfg.dir, tag, session)

	records := newRecords()
	records.analysis(in)

	page := components.NewPage()
	page.PageTitle = "Performance Analysis Tool"
	page.AddCharts(
		records.pieCPU(),
		records.pieMEM(),
	)

	page.SetLayout(components.PageFlexLayout)
	page.Render(w)
}

func infoHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(info))
}

func snapshotHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	tag := params["tag"]
	session := "snapshot-" + params["session"]

	in := fmt.Sprintf("%v/%v/%v.data", cfg.dir, tag, session)
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

func updatePageTpl() {
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
					<a href="http://%s:%s/info"><input type="button" style="width:100px;height:30px;border:5px orange double;margin-top:10px" value="INFO"/></a>
					<a href="http://%s:%s"><input type="button" style="width:100px;height:30px;border:5px orange double;margin-top:10px" value="HISTORY"/></a>
					<input id="pieview" type="button" style="width:100px;height:30px;border:5px blue double;margin-top:10px"value="PIEVIEW"/>
					<input id="snapshot" type="button" style="width:100px;height:30px;border:5px blue double;margin-top:10px"value="SNAPSHOT"/>
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
				`, echarts, themes, cfg.ip, cfg.fileport, cfg.ip, cfg.chartport, cfg.ip, cfg.fileport)
}

//startChartServer to start http chart server
func startChartServer() {
	router := mux.NewRouter().StrictSlash(false)
	router.HandleFunc("/info", infoHandler)
	router.HandleFunc("/{tag}/{session}", lineHandler)
	router.HandleFunc("/{tag}/{session}/pie", pieHandler)
	router.HandleFunc("/{tag}/{session}/snapshot", snapshotHandler)

	chartShutdown = make(chan struct{})
	idleConnsClosed := make(chan struct{})
	srv := &http.Server{
		Addr:    ":" + cfg.chartport,
		Handler: router,
	}

	go func() {
		for {
			if cfg.ip != "" {
				updatePageTpl()
				return
			}
			time.Sleep(time.Duration(2) * time.Second)
		}
	}()

	go func() {
		<-chartShutdown
		if err := srv.Shutdown(context.Background()); err != nil {
			log.Printf("HTTP server Shutdown: %v", err)
		}
		close(idleConnsClosed)
	}()

	log.Printf("start chart http server addr %s", srv.Addr)

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("chart http server ListenAndServe: %v", err)
	}

	<-idleConnsClosed
}

//stopChartServer to stop http chart server
func stopChartServer() {
	close(chartShutdown)
}
