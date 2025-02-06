package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"github.com/gorilla/mux"
	"google.golang.org/api/option"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Models
type User struct {
	gorm.Model
	DeviceToken string `json:"device_token"`
	Platform    string `json:"platform"` // ios or android
	Email       string `json:"email" gorm:"unique"`
}

type Notification struct {
	gorm.Model
	Title  string    `json:"title"`
	Body   string    `json:"body"`
	UserID uint      `json:"user_id"`
	Status string    `json:"status"` // sent, failed, pending
	SendAt time.Time `json:"send_at"`
}

// Global variables
var db *gorm.DB
var firebaseApp *firebase.App

// Database initialization
func initDB() {
	var err error
	dsn := "postgresql://neondb_owner:npg_n9kxwzWRBSY6@ep-patient-lake-a4ejqvcg-pooler.us-east-1.aws.neon.tech/neondb?sslmode=require"

	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to Neon database:", err)
	}

	log.Println("Successfully connected to Neon database")

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Failed to get database instance:", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Auto migrate the schemas
	err = db.AutoMigrate(&User{}, &Notification{})
	if err != nil {
		log.Fatal("Failed to migrate database schemas:", err)
	}
}

// Firebase initialization
func initFirebase() {
	opt := option.WithCredentialsFile("/home/suarim/superleap/stuff/ollamachatbot/basic_firebase/first-846ad-firebase-adminsdk-fbsvc-1daf0dbce3.json") // Make sure this path is correct
	config := &firebase.Config{
		ProjectID: "first-846ad",
	}
	var err error
	firebaseApp, err = firebase.NewApp(context.Background(), config, opt)
	if err != nil {
		log.Fatalf("Failed to initialize Firebase app: %v", err)
	}
	log.Println("Firebase initialized successfully")
}

// Handlers
func createNotification(w http.ResponseWriter, r *http.Request) {
	var notification Notification
	if err := json.NewDecoder(r.Body).Decode(&notification); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	notification.Status = "pending"
	if notification.SendAt.IsZero() {
		notification.SendAt = time.Now()
	}

	if result := db.Create(&notification); result.Error != nil {
		http.Error(w, result.Error.Error(), http.StatusInternalServerError)
		return
	}

	go sendNotification(notification)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(notification)
}

func getNotifications(w http.ResponseWriter, r *http.Request) {
	var notifications []Notification
	if result := db.Find(&notifications); result.Error != nil {
		http.Error(w, result.Error.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(notifications)
}

func registerUser(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if result := db.Create(&user); result.Error != nil {
		http.Error(w, result.Error.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

// FCM related functions
func handleFCMError(err error) string {
	switch {
	case messaging.IsRegistrationTokenNotRegistered(err):
		return "token_expired"
	case messaging.IsInvalidArgument(err):
		return "invalid_token"
	case messaging.IsMessageRateExceeded(err):
		return "rate_limited"
	case messaging.IsServerUnavailable(err):
		return "server_unavailable"
	default:
		return "failed"
	}
}

func sendNotification(notification Notification) {
	var user User
	if result := db.First(&user, notification.UserID); result.Error != nil {
		log.Printf("Failed to find user: %v", result.Error)
		updateNotificationStatus(notification.ID, "failed")
		return
	}

	ctx := context.Background()
	client, err := firebaseApp.Messaging(ctx)
	if err != nil {
		log.Printf("Firebase messaging client error: %v", err)
		updateNotificationStatus(notification.ID, "failed")
		return
	}

	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title: notification.Title,
			Body:  notification.Body,
		},
		Token: user.DeviceToken,
		Data: map[string]string{
			"notificationId": fmt.Sprintf("%d", notification.ID),
		},
	}

	response, err := client.Send(ctx, message)
	if err != nil {
		log.Printf("Detailed FCM error: %+v", err)
		status := handleFCMError(err)
		updateNotificationStatus(notification.ID, status)
		return
	}

	log.Printf("Successfully sent message: %v", response)
	updateNotificationStatus(notification.ID, "sent")
}

func sendBatchNotifications(notifications []Notification) {
	ctx := context.Background()
	client, err := firebaseApp.Messaging(ctx)
	if err != nil {
		log.Printf("Failed to get Messaging client: %v", err)
		return
	}

	messages := make([]*messaging.Message, len(notifications))

	for i, notification := range notifications {
		var user User
		if result := db.First(&user, notification.UserID); result.Error != nil {
			continue
		}

		messages[i] = &messaging.Message{
			Notification: &messaging.Notification{
				Title: notification.Title,
				Body:  notification.Body,
			},
			Token: user.DeviceToken,
		}
	}

	batch, err := client.SendAll(ctx, messages)
	if err != nil {
		log.Printf("Failed to send batch: %v", err)
		return
	}

	for i, response := range batch.Responses {
		if response.Error != nil {
			updateNotificationStatus(notifications[i].ID, "failed")
		} else {
			updateNotificationStatus(notifications[i].ID, "sent")
		}
	}
}

func updateNotificationStatus(id uint, status string) {
	db.Model(&Notification{}).Where("id = ?", id).Update("status", status)
}

// Middleware
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.RequestURI)
		next.ServeHTTP(w, r)
	})
}

func main() {
	initDB()
	initFirebase()

	router := mux.NewRouter()

	// Apply middleware
	router.Use(loggingMiddleware)

	// Routes
	router.HandleFunc("/notifications", createNotification).Methods("POST")
	router.HandleFunc("/notifications", getNotifications).Methods("GET")
	router.HandleFunc("/users", registerUser).Methods("POST")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
