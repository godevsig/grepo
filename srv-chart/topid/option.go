package topid

type config struct {
	ip        string
	chartport string
	fileport  string
	dir       string
}

var cfg config

// WithPort sets the port.
func WithPort(port string) {
	cfg.chartport = port
}

// WithDir sets the dir.
func WithDir(dir string) {
	cfg.dir = dir
}

// SetGlobalOptions sets port&dir
func SetGlobalOptions(port, dir string) {
	cfg.chartport = port
	cfg.dir = dir
}
