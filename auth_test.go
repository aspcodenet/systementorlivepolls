package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aspcodenet/systementorlivepolls/pages" // Assuming pages.Userkey is defined here
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie" // Using cookie store for simplicity in tests
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert" // For convenient assertions
)

// Mock the pages.Userkey if it's not exported or for isolated testing
// For this example, we assume pages.Userkey is exported as a const string.
// If it's not, you might need to mock or define it here for testing purposes.
// const Userkey = "user" // Example if pages.Userkey wasn't available

func TestWebPageAuthRequired_NoUser(t *testing.T) {
	// 1. Setup Gin in test mode
	gin.SetMode(gin.TestMode)
	router := gin.New() // Use gin.New() to avoid default logger/recovery for cleaner tests

	// 2. Setup session middleware (required by WebPageAuthRequired)
	store := cookie.NewStore([]byte("test_secret")) // Use a test secret
	router.Use(sessions.Sessions("mysession", store))

	// 3. Apply the middleware to a test route
	router.GET("/admin/test", WebPageAuthRequired, func(c *gin.Context) {
		c.String(http.StatusOK, "Authenticated Content")
	})

	// 4. Create a request to simulate a client accessing the protected route
	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/admin/test", nil)
	// Set the Host header explicitly for redirect_uri calculation
	req.URL.Host = "localhost:8080"
	req.Host = "localhost:8080"

	// 5. Create a ResponseRecorder to capture the HTTP response
	w := httptest.NewRecorder()

	// 6. Perform the request
	router.ServeHTTP(w, req)

	// 7. Assertions
	// Check if the status code is 302 (Found - Redirect)
	assert.Equal(t, http.StatusFound, w.Code, "Expected HTTP 302 redirect for unauthenticated user")

	expectedRedirectPath := "/loginv1"
	//Assert w header starts wiith expectedRedirectPath
	assert.True(t, strings.HasPrefix(w.Header().Get("Location"), expectedRedirectPath), "Expected redirect to login page with correct redirect_uri")
	//	assert.Equal(t, expectedRedirectPath, w.Header().Get("Location"), "Expected redirect to login page with correct redirect_uri")

	// Ensure no content was written by the next handler
	//	assert.Empty(t, w.Body.String(), "Expected empty response body on redirect")
}

func TestWebPageAuthRequired_UserAuthenticated(t *testing.T) {
	// 1. Setup Gin in test mode
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 2. Setup session middleware
	store := cookie.NewStore([]byte("test_secret"))
	router.Use(sessions.Sessions("mysession", store)) // Apply session middleware globally

	// 3. Apply the middleware and a dummy handler that should be reached
	//    Add an intermediate handler to set the session *after* sessions.Sessions runs
	router.GET("/admin/test", func(c *gin.Context) {
		// This handler runs after sessions.Sessions has initialized the session in the context.
		// We can now safely get the session and set the user.
		session := sessions.Default(c)
		session.Set(pages.Userkey, "test_user_id") // Set a dummy user ID
		session.Save()                             // IMPORTANT: Save the session changes for subsequent middleware/handlers
		c.Next()                                   // Continue to the next handler in the chain (WebPageAuthRequired)
	}, WebPageAuthRequired, func(c *gin.Context) {
		// This handler should be reached if authentication passes
		c.String(http.StatusOK, "Authenticated Content")
	})

	// 4. Create a request
	req := httptest.NewRequest(http.MethodGet, "/admin/test", nil)
	w := httptest.NewRecorder()

	// 5. Perform the request (this will run through the entire middleware chain)
	router.ServeHTTP(w, req)

	// 6. Assertions
	// Check if the status code is 200 (OK)
	assert.Equal(t, http.StatusOK, w.Code, "Expected HTTP 200 OK for authenticated user")

	// Check if the body contains the content from the next handler
	assert.Equal(t, "Authenticated Content", w.Body.String(), "Expected content from authenticated handler")

	// Ensure no redirect occurred
	assert.Empty(t, w.Header().Get("Location"), "Expected no redirect for authenticated user")
}
