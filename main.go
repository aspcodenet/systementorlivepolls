package main

import (
	"fmt"
	"log"
	"os"

	"github.com/aspcodenet/systementorlivepolls/data"
	"github.com/aspcodenet/systementorlivepolls/pages"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	cors "github.com/rs/cors/wrapper/gin"

	//	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

var secret = "qweew3eeeqw"

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, assuming environment variables are set.")
	}

	gin.SetMode(gin.ReleaseMode)
	if os.Getenv("GIN_MODE") == "debug" {
		gin.SetMode(gin.DebugMode)
	}

	data.InitDb(&data.DbConfig{
		Username: os.Getenv("ADMIN_DATABASE_USER"),
		Password: os.Getenv("ADMIN_DATABASE_PASS"),
		Database: os.Getenv("ADMIN_DATABASE_DATABASE"),
		Server:   os.Getenv("ADMIN_DATABASE_SERVER")})

	pages.Init(os.Getenv("ADMIN_SSO_CLIENTID"), os.Getenv("ADMIN_SSO_CLIENTSECRET"))

	r := gin.Default()

	if os.Getenv("ADMIN_REDIS_SERVER") != "" {
		store, _eee := redis.NewStore(10, "tcp", os.Getenv("ADMIN_REDIS_SERVER"), "", "", []byte(secret))
		if _eee != nil {
			fmt.Println(_eee.Error())
		}
		r.Use(sessions.Sessions("mysessionRDS", store))
	} else {
		log.Println("ADMIN_REDIS_SERVER environment variable is not set. - using memory session store")
		store := cookie.NewStore([]byte(secret))
		r.Use(sessions.Sessions("mysession", store))
	}

	corsConfig := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Origin", "Content-Type"},
		AllowCredentials: true,
	})
	r.Use(corsConfig)

	r.Static("/assets/img", "assets/img")
	r.Static("/assets/js", "assets/js")
	r.Static("/assets/css", "assets/css")

	r.LoadHTMLGlob("templates/**")

	r.GET("/", pages.Start)
	r.GET("/loginv1", pages.GithubLoginHandler)
	r.GET("/login/oauth2/code/github", pages.GithubCallbackHandler)
	r.GET("/logout", pages.Logout)
	r.GET("/admin/polls", WebPageAuthRequired, pages.AdminPolls)
	r.GET("/admin/polls/new", WebPageAuthRequired, pages.AdminPollsNew)
	r.POST("/admin/polls/save", WebPageAuthRequired, pages.AdminPollsSavePOST)

	r.GET("/admin/polls/edit/:pollID", WebPageAuthRequired, pages.AdminPollsEdit)
	r.GET("/admin/polls/controlpanel/:pollID", WebPageAuthRequired, pages.AdminPollsControlPanel)

	r.GET("/ws/:pollID", handleWebSocket)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port for json-server
	}

	log.Fatal(r.Run(":" + port))
}
