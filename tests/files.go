package tests

import (
	"fmt"

	"github.com/MelloB1989/karma/files"
	"github.com/gofiber/fiber/v2"
)

func TestKarmaFiles() {
	kf := files.NewKarmaFile("test", "")
	app := fiber.New()
	app.Post("/", func(c *fiber.Ctx) error {
		file, err := c.FormFile("document")
		if err != nil {
			return err
		}
		s, err := kf.HandleSingleFileUpload(file)
		return c.Status(200).JSON(fiber.Map{
			"message": fmt.Sprintf("'%s' uploaded!", file.Filename),
			"data":    s,
		})
	})
	app.Post("/multiple", func(c *fiber.Ctx) error {
		// Parse the multipart form:
		form, err := c.MultipartForm()
		if err != nil {
			return err
		}
		// => *multipart.Form

		// Get all files from "documents" key:
		files := form.File["documents"]
		// => []*multipart.FileHeader

		s, err := kf.HandleMultipleFileUpload(files)
		return c.Status(200).JSON(fiber.Map{
			"message": "Files uploaded!",
			"data":    s,
		})
	})
	app.Listen(":3000")

}
