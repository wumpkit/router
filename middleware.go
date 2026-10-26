package router

// Middleware intercepts command execution before reaching the handler.
type Middleware func(next CommandHandler) CommandHandler
