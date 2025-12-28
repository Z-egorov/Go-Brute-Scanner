package wordlists

// Common contains internal wordlist
type Common struct{}

// New creates new list
func New() *Common {
	return &Common{}
}

// GetAll returns all base words
func (c *Common) GetAll() []string {
	return []string{
		// API endpoints
		"api", "api/v1", "api/v2", "api/v3",
		"rest", "rest/api", "graphql", "gql",
		"v1", "v2", "v3",

		// Users
		"users", "user", "profiles", "profile",
		"accounts", "account",

		// Auth
		"auth", "login", "register", "signup", "logout",
		"signin", "signout", "oauth", "oauth2",
		"token", "refresh", "session",

		// Admin
		"admin", "administrator", "dashboard", "cp",
		"admin/login", "admin/dashboard", "admin/users",
		"admin/settings", "wp-admin", "wp-login.php",

		// Products
		"products", "product", "items", "item",
		"goods", "catalog", "categories", "category",

		// Orders
		"orders", "order", "cart", "checkout",
		"basket", "purchase", "transactions",

		// Health & Status
		"health", "status", "ping", "metrics",
		"info", "version", "about",

		// Config
		"config", "settings", "configuration",
		"options", "preferences",

		// Search
		"search", "find", "query", "filter",
		"lookup", "discover",

		// Files
		"upload", "download", "file", "files",
		"media", "images", "image", "static",
		"assets", "resources",

		// WebSocket
		"ws", "websocket", "socket.io", "wss",

		// Webhooks
		"webhook", "callback", "notify", "notification",
		"hook", "event",

		// Misc
		"help", "support", "contact", "faq",
		"docs", "documentation", "guide",
	}
}

// GetAPI returns API-specific words
func (c *Common) GetAPI() []string {
	return []string{
		"api", "api/v1", "api/v2", "api/v3",
		"rest", "rest/api", "graphql", "gql",
		"v1", "v2", "v3", "endpoints",
	}
}

// GetAdmin returns admin's base paths
func (c *Common) GetAdmin() []string {
	return []string{
		"admin", "administrator", "dashboard", "cp",
		"admin/login", "admin/dashboard", "admin/users",
		"admin/settings", "wp-admin", "wp-login.php",
		"backend", "backoffice", "management",
		"control", "panel", "console",
	}
}

// GetAuth returns auth's base paths
func (c *Common) GetAuth() []string {
	return []string{
		"auth", "login", "register", "signup", "logout",
		"signin", "signout", "oauth", "oauth2",
		"token", "refresh", "session", "authorize",
		"authenticate", "password", "reset",
	}
}
