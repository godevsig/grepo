package main

import (
	"fmt"

	as "github.com/godevsig/adaptiveservice"
	"github.com/godevsig/grepo/plantuml/plantuml"
)

const data = `@startuml
	!define RECTANGLE class 

	participant "APIGateway" as APIGateway
	participant "Orchestrator" as Orchestrator
	participant "VoltMgmt" as VoltMgmt

	APIGateway -> Orchestrator : GetVoltStatus Request  (with parameters: VoltID)
	activate Orchestrator

	Orchestrator -> VoltMgmt : GetVoltStatus Request to the "dindInVM" voltmgmt instance\n(with parameters: VoltID)
	activate VoltMgmt

	VoltMgmt -> VoltMgmt :
	note right: go voltmgmt.GetVoltStatus(VoltID, callback)


	VoltMgmt --> Orchestrator : Response\n(VoltID, Status)
	deactivate VoltMgmt

	Orchestrator --> APIGateway : Final Response\n(VoltID, Status)
	deactivate Orchestrator
	@enduml`

func main() {
	c := as.NewClient().SetDiscoverTimeout(0)
	conn := <-c.Discover("platform", "plantuml")

	if conn != nil {
		plantRequest := plantuml.PlantRequest{Tag: "test", Data: data}
		var plantResponse plantuml.PlantResponse
		if err := conn.SendRecv(&plantRequest, &plantResponse); err != nil {
			fmt.Println(err)
		}
		fmt.Printf("visit this web page to get the image --> %v\n", plantResponse.URL)

		conn.Close()
	}
}
