package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/theairlock/airlock/apps/api/internal/middleware"
)

func GetMe(c *fiber.Ctx) error {
	user, ok := middleware.GetUser(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	return c.JSON(fiber.Map{"user": userToMap(user)})
}

func GetPublic(c *fiber.Ctx) error {
	user, ok := middleware.GetUser(c)
	res := fiber.Map{
		"message":       "public endpoint",
		"authenticated": ok,
	}
	if ok {
		res["user"] = userToMap(user)
	}
	return c.JSON(res)
}

func userToMap(u *middleware.User) fiber.Map {
	m := fiber.Map{
		"id":       u.ID,
		"provider": u.Provider,
	}
	if u.Contact != nil {
		m["contact"] = fiber.Map{
			"email": u.Contact.Email,
			"tel":   u.Contact.Tel,
		}
	}
	return m
}
