package plantuml

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/gorilla/mux"
	"github.com/swaggo/http-swagger"
	_ "github.com/godevsig/grepo/plantuml/docs"
)

// RestAPIServer is the REST API server
type RestAPIServer struct {
	port   string
	server *http.Server
}

// NewRestAPIServer creates a new instance of the REST API server
func NewRestAPIServer(port string) *RestAPIServer {
	return &RestAPIServer{port: port}
}

// Start starts the REST API server
// @title PlantUML Server API
// @version 1.0
// @description This is a sample server for PlantUML.
// @BasePath /
func (s *RestAPIServer) Start() {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/create", s.createHandler).Methods("POST")
	router.PathPrefix("/docs/").Handler(httpSwagger.WrapHandler)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%s", s.port),
		Handler: router,
	}

	if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
		fmt.Printf("HTTP server ListenAndServe: %v\n", err)
	}
}

// createHandler handles the POST requests to the /create endpoint.
// @Summary Create UML diagram
// @Description Create UML diagram from PlantUML data.
// @Accept  json
// @Produce  json
// @Param   request body PlantRequest true "UML Data"
// @Success 200 {object} PlantResponse
// @Failure 400 {string} string "Bad Request"
// @Failure 500 {string} string "Internal Server Error"
// @Router /create [post]
func (s *RestAPIServer) createHandler(w http.ResponseWriter, r *http.Request) {
	var request PlantRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("Error decoding request: %s", err), http.StatusBadRequest)
		return
	}

	// Check if request.Data is empty
	if request.Data == "" {
		http.Error(w, "Request data is empty", http.StatusBadRequest)
		return
	}

	id := time.Now().Format("20060102") + "-" + randStringRunes(4)

	filepath := fmt.Sprintf("%v/%v", workdir+"/data", request.Tag)
	if err := os.MkdirAll(filepath, 0777); err != nil {
		http.Error(w, fmt.Sprintf("Error creating directory: %s", err), http.StatusInternalServerError)
		return
	}

	file, err := os.OpenFile(path.Join(filepath, id+".puml"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error opening file: %s", err), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	_, err = file.WriteString(request.Data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error writing to file: %s", err), http.StatusInternalServerError)
		return
	}

	cmd := exec.Command("java", "-jar", workdir+"/plantuml-1.2023.11.jar", "-tsvg", id+".puml")
	cmd.Dir = filepath
	output, err := cmd.CombinedOutput()
	if err != nil {
		http.Error(w, fmt.Sprintf("Command execution failed: %v, output: %s", err, string(output)), http.StatusInternalServerError)
		return
	}

	url := fmt.Sprintf("http://%v/%v/%v", hostAddr, request.Tag, id+".svg")
	response := PlantResponse{URL: url}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding response: %s", err), http.StatusInternalServerError)
		return
	}
}

// Stop stops the REST API server
func (s *RestAPIServer) Stop() {
	if s.server == nil {
		fmt.Println("Server not started")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.server.Shutdown(ctx); err != nil {
		fmt.Printf("HTTP server Shutdown: %v\n", err)
	}
}
