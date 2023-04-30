package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gofiber/fiber/v2"

	github2 "github.com/ft-t/streamdeck/pkg/github"
)

func main() {
	app := fiber.New()

	githubSvc := github2.NewGithub(os.Getenv("GITHUB_API_TOKEN"))

	app.Get("/github/pr/status", func(c *fiber.Ctx) error {
		prUrl := c.Query("url")

		status, err := githubSvc.GetPullStatus(c.Context(), prUrl)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return c.SendString(err.Error())
		}

		fmt.Println(status)
		return c.SendString("Hello, World 👋!")
	})

	app.Listen(":3000")
}
