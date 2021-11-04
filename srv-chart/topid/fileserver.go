package topid

import (
	"context"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/godevsig/grepo/lib-sys/log"
	"github.com/gorilla/mux"
)

type fileAttr struct {
	Flist []os.FileInfo
	Ftype []string
	Ftag  string
}

type fileServer struct {
	dir      string
	port     string
	listener net.Listener
	lg       *log.Logger
	srv      *http.Server
}

const tmplText = `
<!doctype html>
<html>
    <head>
        <meta charset="UTF-8">
        <title>HTTP File Server</title>
        <style>* { padding:0; margin:0; } body { color: #333; font: 14px Sans-Serif; background: #ccc; margin: 0 auto; } h1 { text-align: center; padding: 20px 0 12px 0; margin: 0; background: #ccc; } #container { box-shadow: 0 5px 10px -5px rgba(0,0,0,0.5); position: relative; background: #ccc; margin: 0 auto; width: 80%; padding-bottom: 150px; } table { background-color: #F3F3F3; border-collapse: collapse; margin: auto; width: 100%; background: #e9e9e9; padding: 50px; } th { background-color: #3d0808; color: #FFF; cursor: pointer; padding: 5px 10px; } th small { font-size: 9px; } td, th { text-align: left; } a { text-decoration: none; } td a { color: #663300; display: block; padding: 5px 10px; }</style></head>
    <body>
        <div id="container">
            <h1>TOPID DATA</h1>
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
                            <a href="{{$.Ftag}}/{{$file.Name}}">{{$file.Name}}</a></td>
                        <td>{{$file.Size}}</td>
                        <td>{{$type}}</td>
                        <td>{{$file.ModTime}}</td></tr>{{end}}</tbody>
            </table>
        </div>
    </body>
</html>
`

func newFileServer(lg *log.Logger, dir string) *fileServer {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		lg.Errorf("file server listen failed: %v", err)
		return nil
	}

	fs := &fileServer{
		lg:       lg,
		dir:      dir,
		listener: listener,
		port:     strconv.Itoa(listener.Addr().(*net.TCPAddr).Port),
	}

	router := mux.NewRouter().StrictSlash(false)
	handler := http.FileServer(http.Dir(fs.dir))
	router.HandleFunc("/", fs.fileIndex)
	router.HandleFunc("/{tag}", fs.fileIndex)
	//router.HandleFunc("/upload", fileUpload)

	router.PathPrefix("/").Handler(http.StripPrefix("/", handler))

	fs.srv = &http.Server{
		Addr:    ":" + fs.port,
		Handler: router,
	}
	return fs
}

func (fs *fileServer) fileIndex(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	tag := params["tag"]

	if tag != "" && (strings.Index(tag, ".") != -1) || tag == "README" {
		file, err := os.Open(fs.dir + "/" + tag)
		defer file.Close()
		if err != nil {
			http.Error(w, "File not found.", 404)
			return
		}
		io.Copy(w, file)
		return
	}

	tmpl := template.Must(template.New("").Parse(tmplText))
	files, err := ioutil.ReadDir(fs.dir + "/" + tag)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	var fileType []string
	for _, file := range files {
		ext := filepath.Ext(file.Name())
		if ext == "" {
			fileType = append(fileType, "dir")
		} else {
			fileType = append(fileType, ext)
		}
	}
	fa := fileAttr{Flist: files, Ftype: fileType, Ftag: tag}
	tmpl.Execute(w, fa)
}

func (fs *fileServer) fileUpload(w http.ResponseWriter, r *http.Request) {
}

func (fs *fileServer) fileDelete(w http.ResponseWriter, r *http.Request) {
}

func (fs *fileServer) start() {
	fs.lg.Infof("start file http server addr %s", fs.srv.Addr)

	if err := fs.srv.Serve(fs.listener); err != http.ErrServerClosed {
		fs.lg.Errorf("file http server ListenAndServe: %v", err)
		return
	}
}

func (fs *fileServer) stop() {
	if err := fs.srv.Shutdown(context.Background()); err != nil {
		fs.lg.Errorf("file http server shutdown failed: %v", err)
	}
	fs.lg.Infoln("file http server shutdown successfully")
}
