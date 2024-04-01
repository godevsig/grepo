package fileserver

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/godevsig/glib/sys/log"
	"github.com/gorilla/mux"
)

type attributes struct {
	Flist       []os.FileInfo
	Ftype       []string
	Title       string
	CurrentPath string
}

// FileServer represents data server
type FileServer struct {
	Port     string
	dir      string
	title    string
	lg       *log.Logger
	srv      *http.Server
	listener net.Listener
}

const pageTpl = `
	<!doctype html>
	<html>
		<head>
			<meta charset="UTF-8">
			<title>HTTP File Server</title>
			<style>* { padding:0; margin:0; } body { color: #333; font: 14px Sans-Serif; background: #ccc; margin: 0 auto; } h1 { text-align: center; padding: 20px 0 12px 0; margin: 0; background: #ccc; } #container { box-shadow: 0 5px 10px -5px rgba(0,0,0,0.5); position: relative; background: #ccc; margin: 0 auto; width: 80%; padding-bottom: 150px; } table { background-color: #F3F3F3; border-collapse: collapse; margin: auto; width: 100%; background: #e9e9e9; padding: 50px; } th { background-color: #3d0808; color: #FFF; cursor: pointer; padding: 5px 10px; } th small { font-size: 9px; } td, th { text-align: left; } a { text-decoration: none; } td a { color: #663300; display: block; padding: 5px 10px; }</style></head>
		<body>
			<div id="container">
				<h1>{{$.Title}}</h1>
				<table class="sortable">
					<thead>
						<tr>
							<th>Filename</th>
							<th>Size
								<small>(bytes)</small></th>
							<th>Type</th>
							<th>Date Modified</th></tr>
					</thead>
					<tbody>{{range $i, $file := .Flist}} {{$type := index $.Ftype $i}}
						<tr>
							<td>
								<a href="{{$.CurrentPath}}/{{$file.Name}}">{{$file.Name}}</a></td>
							<td>{{$file.Size}}</td>
							<td>{{$type}}</td>
							<td>{{$file.ModTime}}</td></tr>{{end}}</tbody>
				</table>
			</div>
		</body>
	</html>`

// NewFileServer creates a new file server instance.
// If port = "0", alloc available port by Listen.
// If port != "0", use customized ip port.
func NewFileServer(lg *log.Logger, port, dir, title string) *FileServer {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		lg.Errorf("file server listen failed: %v", err)
		return nil
	}

	if port == "0" {
		port = strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	}

	fs := &FileServer{
		Port:     port,
		dir:      dir,
		lg:       lg,
		title:    title,
		listener: listener,
	}

	router := mux.NewRouter().StrictSlash(false)
	router.PathPrefix("/").HandlerFunc(fs.fileIndex)

	fs.srv = &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}
	return fs
}

func (fs *FileServer) fileIndex(w http.ResponseWriter, r *http.Request) {
	fullPath := filepath.Join(fs.dir, r.URL.Path)
	file, err := os.Open(fullPath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		http.Error(w, "Error reading file", http.StatusInternalServerError)
		return
	}

	if !fileInfo.IsDir() {
		io.Copy(w, file)
		return
	}

	dirEntries, err := file.Readdir(-1)
	if err != nil {
		http.Error(w, "Error reading directory", http.StatusInternalServerError)
		return
	}

	var fileTypes []string
	for _, f := range dirEntries {
		if f.IsDir() {
			fileTypes = append(fileTypes, "Directory")
		} else {
			fileTypes = append(fileTypes, "File")
		}
	}

	currentPath := r.URL.Path
	if !strings.HasPrefix(currentPath, "/") {
		currentPath = "/" + currentPath
	}

	fa := attributes{
		Flist:       dirEntries,
		Ftype:       fileTypes,
		Title:       fs.title,
		CurrentPath: strings.TrimSuffix(currentPath, "/"),
	}

	tmpl := template.Must(template.New("").Parse(pageTpl))
	if err := tmpl.Execute(w, fa); err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
	}
}

func (fs *FileServer) fileUpload(w http.ResponseWriter, r *http.Request) {
}

func (fs *FileServer) fileDelete(w http.ResponseWriter, r *http.Request) {
}

// Start start the file server
func (fs *FileServer) Start() error {
	fs.lg.Infof("start file http server addr %s", fs.srv.Addr)

	err := fs.srv.Serve(fs.listener)
	if err == http.ErrServerClosed {
		err = nil
	}
	if err != nil {
		fs.lg.Errorf("file http server ListenAndServe: %v", err)
	}
	return err
}

// Stop stop the file server
func (fs *FileServer) Stop() {
	if err := fs.srv.Shutdown(context.Background()); err != nil {
		fs.lg.Errorf("file http server shutdown failed: %v", err)
	}
	fs.lg.Infoln("file http server shutdown successfully")
}
