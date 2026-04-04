package main

import (
	"ReaperC2/pkg/adminpanel"
	"ReaperC2/pkg/apiserver"
	"ReaperC2/pkg/dbconnections"
	"ReaperC2/pkg/deploymehere"
	"log"
	"os"
)

// Execute the program
func main() {

	deployEnv := deploymehere.GetDeploymentEnv()
	switch deployEnv {
	case "AWS":
		dbconnections.InitMongoDB(deployEnv)
		adminpanel.BootstrapFirstOperator()
		runDualListeners()
	case "AZURE":
		log.Println("AZURE: Coming soon")
	case "GCP":
		log.Println("GCP: Coming soon")
	case "ONPREM":
		dbconnections.InitMongoDB(deployEnv)
		adminpanel.BootstrapFirstOperator()
		runDualListeners()
	default:
		log.Println("Unknown Environment")
	}
}

func runDualListeners() {
	if os.Getenv("ADMIN_DISABLE") == "1" {
		log.Println("ADMIN_DISABLE=1: admin panel listener skipped")
		if err := apiserver.StartBeaconServer(""); err != nil {
			log.Fatalf("beacon server: %v", err)
		}
		return
	}
	errCh := make(chan error, 2)
	go func() {
		errCh <- apiserver.StartBeaconServer("")
	}()
	go func() {
		errCh <- adminpanel.Start("")
	}()
	log.Fatal(<-errCh)
}
