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

	"go.mongodb.org/mongo-driver/bson"
)

// Setting endpoints as a constant
// this helps the user declare their own endpoints
const (
	endpointRoot          = "/"
	endpointStatus        = "/status"
	endpointComingSoon    = "/coming-soon"
	endpointData          = "/data"
	endpointDataHeartbeat = "/heartbeat"
	endpointReceive       = "/receive"
)

// Start the API server
func StartAPIServer() {
	// Welcome message to know the API exists
	http.HandleFunc(endpointRoot, func(w http.ResponseWriter, r *http.Request) {
		JsonResponse(w, map[string]string{"message": "Welcome"})
	})

	// Status check to make sure the API is running
	http.HandleFunc(endpointStatus, func(w http.ResponseWriter, r *http.Request) {
		JsonResponse(w, map[string]string{"message": "API is running"})
	})

	// Redirector, use this to redirect unauthenticated clients
	http.HandleFunc(endpointComingSoon, func(w http.ResponseWriter, r *http.Request) {
		JsonResponse(w, map[string]string{"message": "Coming soon..."})
	})

	// Protected endpoint (requires valid ClientId & Secret)
	http.HandleFunc(endpointData, AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		// Debugging
		// filePath := "data.json"
		// file, err := os.Open(filePath)
		// if err != nil {
		// 	log.Fatalf("Error opening file: %v", err)
		// }
		// defer file.Close()
		// log.Println("File opened successfully!")

		data, err := LoadJSONFromFile("data.json")
		if err != nil {
			// log.Println("data.json should be here")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		JsonResponse(w, data)
	}))

	http.HandleFunc(endpointDataHeartbeat, AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		result, err := dbconnections.FetchHeartbeat()
		if err != nil {
			log.Println("Error fetching heartbeat:", err)
			http.Error(w, `{"error": "Failed to fetch heartbeat"}`, http.StatusInternalServerError)
			return
		}

		// Extract only the "status" field
		status, ok := result["status"].(string)
		// status, ok := result["command"].(string)
		if !ok {
			http.Error(w, "Invalid data format", http.StatusInternalServerError)
			return
		}

		JsonResponse(w, map[string]string{"status": status})
		// JsonResponse(w, map[string]string{"command": status})
		// JsonResponse(w, []map[string]string{{"command": status}})
	}))

	http.HandleFunc(endpointReceive, AuthMiddleware(HandleReceive))

	log.Println("Server running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
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
func ValidateClientAuth(clientId, secret string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var clientAuth dbconnections.ClientAuth
	filter := bson.M{"ClientId": clientId, "Secret": secret, "Active": true}

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
		clientId := r.Header.Get("X-Client-Id")
		// log.Println(clientId)
		clientSecret := r.Header.Get("X-API-Secret")
		// log.Println(clientSecret)
		clientIP := r.RemoteAddr
		requestTime := time.Now().Format(time.RFC3339) // Logs timestamp in ISO format

		// Extract real IP if behind a reverse proxy
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			clientIP = forwarded
		}

		// Log the request details
		log.Printf("[%s] Request from %s | ClientID: %s | Endpoint: %s\n", requestTime, clientIP, clientId, r.URL.Path)

		if clientId == "" || clientSecret == "" {
			http.Redirect(w, r, endpointComingSoon, http.StatusTemporaryRedirect)
			return
		}

		valid, err := ValidateClientAuth(clientId, clientSecret)
		if err != nil || !valid {
			log.Printf("[%s] Unauthorized request from %s | ClientID: %s\n", requestTime, clientIP, clientId)
			http.Redirect(w, r, endpointComingSoon, http.StatusTemporaryRedirect)
			return
		}

		next.ServeHTTP(w, r) // Call the next handler
	}
}
