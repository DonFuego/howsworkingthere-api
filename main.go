package main

import (
	"github.com/howsworkingthere/hows-working-there-api/handler"
	"github.com/howsworkingthere/hows-working-there-api/middleware"
	"gofr.dev/pkg/gofr"
)

func main() {
	app := gofr.New()

	// Auth0 JWT middleware — validates Bearer token on all requests
	app.UseMiddleware(middleware.Auth0Middleware)

	// Add a new location
	app.POST("/api/v1/locations", handler.CreateLocation)

	// Location search
	app.GET("/api/v1/locations/search", handler.SearchLocations)

	// Full check-in (new or existing location + speed test + noise level + ratings)
	app.POST("/api/v1/check-ins", handler.CreateCheckIn)

	// Check-in at an existing location
	app.POST("/api/v1/locations/{location_id}/check-ins", handler.CreateCheckInAtLocation)

	// User's tested locations (averaged scores)
	app.GET("/api/v1/users/{user_id}/locations", handler.GetUserLocations)

	// All tested locations (averaged scores)
	app.GET("/api/v1/locations", handler.GetAllLocations)

	// Single location comprehensive detail
	app.GET("/api/v1/locations/{location_id}/detail", handler.GetLocationDetail)

	// User search by email
	app.GET("/api/v1/users/search", handler.SearchUserByEmail)

	// Friends management
	app.POST("/api/v1/friends/request", handler.SendFriendRequest)
	app.POST("/api/v1/friends/accept", handler.AcceptFriendRequest)
	app.POST("/api/v1/friends/deny", handler.DenyFriendRequest)
	app.DELETE("/api/v1/friends/{friend_id}", handler.RemoveFriend)
	app.GET("/api/v1/friends", handler.ListFriends)
	app.GET("/api/v1/friends/activity", handler.GetFriendsActivity)

	// Notifications
	app.GET("/api/v1/notifications", handler.ListNotifications)
	app.POST("/api/v1/notifications/{id}/read", handler.MarkNotificationRead)

	// Auth0 post-registration trigger (uses its own JWT auth via Auth0TriggerMiddleware)
	app.POST("/user/register", handler.RegisterUser)

	app.Run()
}
