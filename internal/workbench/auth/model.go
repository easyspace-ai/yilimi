package auth

// User is the public identity returned to clients and forwarded to downstream proxies.
type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

// guestUser is used when no valid Bearer token is present.
var guestUser = User{
	ID:    "00000000-0000-0000-0000-000000000001",
	Email: "guest@local",
	Name:  "Guest",
	Role:  "guest",
}
