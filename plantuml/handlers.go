package plantuml

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"time"

	as "github.com/godevsig/adaptiveservice"
	"github.com/godevsig/glib/sys/log"
)

// Handle handles ChassisModelRequest.
func (msg *PlantRequest) Handle(stream as.ContextStream) (reply any) {
	lg := stream.GetContext().(*log.Logger)

	id := time.Now().Format("20060102") + "-" + randStringRunes(4)

	filepath := fmt.Sprintf("%v/%v", workdir+"/data", msg.Tag)
	if err := os.MkdirAll(filepath, 0777); err != nil {
		return err
	}

	file, err := os.OpenFile(path.Join(filepath, id+".puml"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		lg.Errorln("Error opening file:", err)
		return err
	}
	defer file.Close()

	_, err = file.WriteString(msg.Data)
	if err != nil {
		lg.Errorln("Error writing to file:", err)
		return err
	}

	cmd := exec.Command("java", "-jar", workdir+"/plantuml-1.2023.11.jar", "-tsvg", id+".puml")
	cmd.Dir = filepath
	output, err := cmd.CombinedOutput()
	if err != nil {
		lg.Errorf("Command execution failed:%v, output:%s", err, string(output))
		return err
	}

	return &PlantResponse{fmt.Sprintf("http://%v/%v/%v", hostAddr, msg.Tag, id+".svg")}
}

var knownMsgs = []as.KnownMessage{
	(*PlantRequest)(nil),
}
