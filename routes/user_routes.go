package routes

import (
	"dinsos_kuburaya/controllers"
	"dinsos_kuburaya/middleware"

	"github.com/gin-gonic/gin"
)

func UserRoutes(router *gin.RouterGroup) {
	users := router.Group("/users")

	users.POST("/push-token", middleware.AuthMiddleware(), controllers.StorePushToken)

	// deactivated for security reasons, only superadmin can create users
	users.POST("/superadmin", controllers.CreateSuperAdmin)

	users.POST("/admin", middleware.AuthMiddleware(), middleware.RoleMiddleware("superadmin"), controllers.CreateAdmin)

	users.POST("/staff", middleware.AuthMiddleware(), middleware.RoleMiddleware("superadmin"), controllers.CreateStaff)

	usersAuth := users.Group("")
	usersAuth.Use(middleware.AuthMiddleware())
	{
		usersAuth.GET("/me", controllers.GetMe)

		usersAuth.GET("", middleware.RoleMiddleware("admin", "superadmin"), controllers.GetUsers)

		usersAuth.GET("/:id", middleware.RoleMiddleware("admin", "superadmin"), controllers.GetUserByID)

		usersAuth.PUT("/:id", middleware.UserSelfOrSuperAdmin(), controllers.UpdateUser)

		usersAuth.PUT("/:id/reset-password", middleware.RoleMiddleware("superadmin"), controllers.ResetPassword)

		usersAuth.DELETE("/:id", middleware.RoleMiddleware("superadmin"), controllers.DeleteUser)

		usersAuth.GET("/for-filter", middleware.RoleMiddleware("admin", "superadmin"), controllers.GetUsersForFilter)
	}
}
