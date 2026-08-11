package contextkeys

type contextKey string

const (
	RequestIDKey contextKey = "request_id"
	RoleKey      contextKey = "role"
	UserIDKey    contextKey = "user_id"
)
