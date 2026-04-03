package main

import (
	"ReaperC2/pkg/apiserver"
	"ReaperC2/pkg/dbconnections"
	"ReaperC2/pkg/deploymehere"
	"log"
)

// Execute the program
func main() {

	deployEnv := deploymehere.GetDeploymentEnv()
	switch deployEnv {
	case "AWS":
		dbconnections.InitMongoDB(deployEnv)
		apiserver.StartAPIServer()
	case "AZURE":
		log.Println("AZURE: Coming soon")
	case "GCP":
		log.Println("GCP: Coming soon")
	case "ONPREM":
		dbconnections.InitMongoDB(deployEnv)
		apiserver.StartAPIServer()
	default:
		log.Println("Unknown Environment")
	}
}
