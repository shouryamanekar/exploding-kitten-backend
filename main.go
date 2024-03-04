package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"log"
    "os"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"golang.org/x/net/context"
	"golang.org/x/crypto/bcrypt"
	"github.com/joho/godotenv"
)

var ctx = context.Background()
var client *redis.Client

// User struct to represent a user
type User struct {
	Username string `json:"username"`
	Points   int    `json:"points"`
	Password string `json:"password"`
}

type LeaderboardEntry struct {
    Username string `json:"username"`
    Points   int    `json:"points"`
    Rank     int    `json:"rank"`
}


func main() {
	// Load environment variables from .env file
    if err := godotenv.Load(); err != nil {
        log.Fatal("Error loading .env file")
    }

	// Redis connection
	client = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"),
    Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	router := mux.NewRouter()

	// Increment user points endpoint
	router.HandleFunc("/api/user/incrementPoints", func(w http.ResponseWriter, r *http.Request) {
    username := r.Header.Get("Username")

    user, err := retrieveUser(username)
    if err != nil {
        http.Error(w, "Error retrieving user", http.StatusInternalServerError)
        return
    }

    if user == nil {
        http.Error(w, fmt.Sprintf("User not found for username: %s", username), http.StatusNotFound)
        return
    }

    user.Points++
    if err := saveUser(*user); err != nil {
        http.Error(w, "Error saving user points", http.StatusInternalServerError)
        return
    }

    fmt.Fprintf(w, "User points incremented for %s. New points: %d", username, user.Points)
	})


	// Get user points endpoint
	router.HandleFunc("/api/user/points", func(w http.ResponseWriter, r *http.Request) {
    
		// Retrieve the user points based on the username
    	username := r.Header.Get("Username")
    	user, err := retrieveUser(username)
    	if err != nil {
        	http.Error(w, "Error retrieving user points", http.StatusInternalServerError)
        	return
    	}

    	if user == nil {
			fmt.Printf("User not found for username: %s\n", username)
        	http.Error(w, "User not found", http.StatusNotFound)
        	return
    	}
    	// Respond with user points
    	response := map[string]int{"points": user.Points}
    	w.Header().Set("Content-Type", "application/json")
    	json.NewEncoder(w).Encode(response)
	})

	// Save data to Redis
	router.HandleFunc("/save", func(w http.ResponseWriter, r *http.Request) {
		var data map[string]string
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		

		key := data["key"]
		value := data["value"]

		if err := saveData(key, value); err != nil {
			http.Error(w, "Error saving data to Redis", http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Data saved to Redis: Key=%s, Value=%s", key, value)
	})

	// User authentication endpoint
	router.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		var user User
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Check if the user already exists in Redis
		existingUser, err := retrieveUser(user.Username)
		if err != nil {
			http.Error(w, "Error checking user existence", http.StatusInternalServerError)
			return
		}

		if existingUser == nil {
			// User doesn't exist, create a new user
			fmt.Println("Creating a new user:", user.Username)
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
			if err != nil {
				http.Error(w, "Error creating hashed password", http.StatusInternalServerError)
				return
			}

			// Save the new user
			user.Password = string(hashedPassword)
			user.Points = 0
			if err := saveUser(user); err != nil {
				http.Error(w, "Error creating user", http.StatusInternalServerError)
				return
			}
		} 
		
			// User exists, compare the provided password with the stored hashed password
			if err := bcrypt.CompareHashAndPassword([]byte(existingUser.Password), []byte(user.Password)); err != nil {
				fmt.Printf("Invalid credentials for user %s\n", user.Username)
				http.Error(w, "Invalid credentials", http.StatusUnauthorized)
				return
			}
		
	
		// Respond with user data
		json.NewEncoder(w).Encode(user)
	})

	// Leaderboard endpoint
router.HandleFunc("/api/leaderboard", func(w http.ResponseWriter, r *http.Request) {
	// Get the username from the request context
	username := r.Header.Get("Username")
	user, err := retrieveUser(username)
	if err != nil {
	   http.Error(w, "Error retrieving User Rank", http.StatusInternalServerError)
	   return
	}

	// Fetch the top 10 leaderboard entries
	leaderboard, err := getLeaderboard(user.Username, 10) // Pass 'username' instead of 'user'
	if err != nil {
	   http.Error(w, "Error fetching leaderboard", http.StatusInternalServerError)
	   return
	}

	// Convert the leaderboard to JSON
	leaderboardJSON, err := json.Marshal(leaderboard)
	if err != nil {
	   http.Error(w, "Error converting leaderboard to JSON", http.StatusInternalServerError)
	   return
	}

	// Respond with the JSON string
	w.Header().Set("Content-Type", "application/json")
	w.Write(leaderboardJSON)
 })
 



	// CORS configuration
	allowedOrigins := strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",")
	c := cors.New(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
		Debug:          true,
	})

	handler := c.Handler(router)

	// Handle preflight requests explicitly
router.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Access-Control-Allow-Origin", "*") // Update with your allowed origins
    w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
    w.Header().Set("Access-Control-Allow-Headers", "*")

    // Respond to preflight request
    w.WriteHeader(http.StatusOK)
})

	// Handle preflight requests explicitly
	router.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	fmt.Println("Server is running on :8080")
	http.ListenAndServe(":"+os.Getenv("PORT"), handler)

}

// Save data to Redis
func saveData(key, value string) error {
	return client.Set(ctx, key, value, 0).Err()
}

// Retrieve data from Redis
func retrieveData(key string) (string, error) {
	val, err := client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil // Key does not exist
	} else if err != nil {
		return "", err
	}
	return val, nil
}

// Save user data to Redis
func saveUser(user User) error {
	userData, err := json.Marshal(user)
	if err != nil {
		return err
	}

	return client.Set(ctx, user.Username, userData, 0).Err()
}

// Retrieve user data from Redis
func retrieveUser(username string) (*User, error) {
	val, err := client.Get(ctx, username).Result()
	if err == redis.Nil {
		return nil, nil // User doesn't exist
	} else if err != nil {
		return nil, err
	}

	var user User
	if err := json.Unmarshal([]byte(val), &user); err != nil {
		return nil, err
	}

	return &user, nil
}

var mu sync.Mutex

// Retrieve all users from Redis
func retrieveAllUsers() ([]User, error) {
	mu.Lock()
	defer mu.Unlock()

	var users []User

	keys, err := client.Keys(ctx, "*").Result()
	if err != nil {
    fmt.Printf("Error retrieving keys: %v\n", err)
    return nil, err
	}

	for _, key := range keys {
    val, err := client.Get(ctx, key).Result()
    if err != nil {
        fmt.Printf("Error retrieving value for key %s: %v\n", key, err)
        return nil, err
    }

    var user User
    if err := json.Unmarshal([]byte(val), &user); err != nil {
        fmt.Printf("Error unmarshalling value for key %s: %v\n", key, err)
        continue  
    }

    users = append(users, user)
	}


	return users, nil
}

// Function to get the leaderboard
func getLeaderboard(username string, topN int) ([]LeaderboardEntry, error) {

	// Retrieve all users and their points from Redis
	users, err := retrieveAllUsers()
	if err != nil {
		return nil, err
	}

	// Sort users based on points in descending order
	sort.SliceStable(users, func(i, j int) bool {
		return users[i].Points > users[j].Points
	})

	var leaderboard []LeaderboardEntry

	// Iterate through the sorted users to create the leaderboard
	for rank, user := range users[:min(topN, len(users))] {
		entry := LeaderboardEntry{
			Username: user.Username,
			Points:   user.Points,
			Rank:     rank + 1,
		}

		leaderboard = append(leaderboard, entry)
	}

	// Fetch the user's own rank, username, and points
	for _, entry := range leaderboard {
		if entry.Username == username {
			return leaderboard, nil
		}
	}

	// If the user is not in the leaderboard, append their entry
	user, err := retrieveUser(username)
	if err != nil {
		return nil, err
	}
	if user != nil {
		// Find the user's rank in the sorted list
		userRank := -1
		for i, u := range users {
			if u.Username == user.Username {
				userRank = i + 1
				break
			}
		}
	
		if userRank != -1 {
			userEntry := LeaderboardEntry{
				Username: user.Username,
				Points:   user.Points,
				Rank:     userRank,
			}
	
			leaderboard = append(leaderboard, userEntry)
		} else {
			fmt.Printf("User not found in the sorted list of users.\n")
		}
	}

	return leaderboard, nil
}


func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}