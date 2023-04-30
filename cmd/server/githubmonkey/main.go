package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gofiber/fiber/v2"

	github2 "github.com/ft-t/streamdeck/pkg/github"
)

func main() {
	app := fiber.New()

	listenAddr := ":8080"
	if val, ok := os.LookupEnv("FUNCTIONS_CUSTOMHANDLER_PORT"); ok {
		listenAddr = ":" + val
	}

	githubSvc := github2.NewGithub(os.Getenv("GITHUB_API_TOKEN"))

	app.Get("/api/github/pr/status", func(c *fiber.Ctx) error {
		prUrl := c.Query("url")

		status, err := githubSvc.GetPullStatus(c.Context(), prUrl)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return c.SendString(err.Error())
		}

		return c.JSON(status)
	})

	log.Printf("About to listen on %s. Go to http://0.0.0.0%s/", listenAddr, listenAddr)
	app.Listen(listenAddr)
}
