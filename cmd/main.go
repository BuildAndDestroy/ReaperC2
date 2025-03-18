package main

import (
	"ReaperC2/pkg/apiserver"
	"ReaperC2/pkg/dbconnections"
)

func main() {
	// Need to run this in a go function
	dbconnections.InitMongoDB()
	apiserver.StartAPIServer()

	// Run another go function for the Admin portal
}
