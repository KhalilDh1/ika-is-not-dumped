package main

import (
	"apartments-clone-server/routes"
	"apartments-clone-server/storage"
	"apartments-clone-server/utils"
	"log"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/jwt"
)

func main() {
	// Load environment variables and initialize storage
	godotenv.Load()
	storage.InitializeDB()
	storage.InitializeStorage(storage.DB)
	storage.InitializeRedis()

	app := iris.Default()
	app.Validator = validator.New()

	// Ensure the Stripe keys are set
	if os.Getenv("STRIPE_SECRET_KEY") == "" {
		log.Fatal("STRIPE_SECRET_KEY environment variable is not set")
	}

	// JWT Verifiers setup
	resetTokenVerifier := jwt.NewVerifier(jwt.HS256, []byte(os.Getenv("EMAIL_TOKEN_SECRET")))
	resetTokenVerifier.WithDefaultBlocklist()
	resetTokenVerifierMiddleware := resetTokenVerifier.Verify(func() interface{} {
		return new(utils.ForgotPasswordToken)
	})

	accessTokenVerifier := jwt.NewVerifier(jwt.HS256, []byte(os.Getenv("ACCESS_TOKEN_SECRET")))
	accessTokenVerifier.WithDefaultBlocklist()
	accessTokenVerifierMiddleware := accessTokenVerifier.Verify(func() interface{} {
		return new(utils.AccessToken)
	})

	refreshTokenVerifier := jwt.NewVerifier(jwt.HS256, []byte(os.Getenv("REFRESH_TOKEN_SECRET")))
	refreshTokenVerifier.WithDefaultBlocklist()
	refreshTokenVerifierMiddleware := refreshTokenVerifier.Verify(func() interface{} {
		return new(jwt.Claims)
	})

	// Custom refresh token extractor
	refreshTokenVerifier.Extractors = append(refreshTokenVerifier.Extractors, func(ctx iris.Context) string {
		var tokenInput utils.RefreshTokenInput
		err := ctx.ReadJSON(&tokenInput)
		if err != nil {
			return ""
		}
		return tokenInput.RefreshToken
	})

	// API Routes
	location := app.Party("/api/location")
	{
		location.Get("/autocomplete", routes.Autocomplete)
		location.Get("/search", routes.Search)
	}

	user := app.Party("/api/user")
	{
		user.Post("/register", routes.Register)
		user.Post("/login", routes.Login)
		user.Post("/facebook", routes.FacebookLoginOrSignUp)
		user.Post("/google", routes.GoogleLoginOrSignUp)
		user.Post("/apple", routes.AppleLoginOrSignUp)
		user.Post("/forgotpassword", routes.ForgotPassword)
		user.Post("/resetpassword", resetTokenVerifierMiddleware, routes.ResetPassword)
		user.Get("/{id}/properties/saved", accessTokenVerifierMiddleware, utils.UserIDMiddleware, routes.GetUserSavedProperties)
		user.Patch("/{id}/properties/saved", accessTokenVerifierMiddleware, utils.UserIDMiddleware, routes.AlterUserSavedProperties)
		user.Patch("/{id}/pushtoken", accessTokenVerifierMiddleware, utils.UserIDMiddleware, routes.AlterPushToken)
		user.Patch("/{id}/settings/notifications", accessTokenVerifierMiddleware, utils.UserIDMiddleware, routes.AllowsNotifications)
		user.Get("/{id}/properties/contacted", accessTokenVerifierMiddleware, utils.UserIDMiddleware, routes.GetUserContactedProperties)
	}

	property := app.Party("/api/property")
	{
		property.Post("/", routes.CreateProperty)
		property.Get("/{id}", routes.GetProperty)
		property.Get("/userid/{id}", accessTokenVerifierMiddleware, utils.UserIDMiddleware, routes.GetPropertiesByUserID)
		property.Delete("/{id}", accessTokenVerifierMiddleware, routes.DeleteProperty)
		property.Patch("/update/{id}", accessTokenVerifierMiddleware, routes.UpdateProperty)
		property.Post("/search", routes.GetPropertiesByBoundingBox)
	}

	apartment := app.Party("/api/apartment")
	{
		apartment.Get("/property/{id}", routes.GetApartmentsByPropertyID)
		apartment.Patch("/property/{id}", accessTokenVerifierMiddleware, routes.UpdateApartments)
	}

	review := app.Party("/api/review")
	{
		review.Post("/property/{id}", accessTokenVerifierMiddleware, routes.CreateReview)
	}

	conversation := app.Party("/api/conversation")
	{
		conversation.Post("/", accessTokenVerifierMiddleware, routes.CreateConversation)
		conversation.Get("/{id}", accessTokenVerifierMiddleware, routes.GetConversationByID)
		conversation.Get("/user/{id}", accessTokenVerifierMiddleware, utils.UserIDMiddleware, routes.GetConversationsByUserID)
	}

	messages := app.Party("/api/messages")
	{
		messages.Post("/", accessTokenVerifierMiddleware, routes.CreateMessage)
	}

	reservation := app.Party("/api/reservation")
	{
		reservation.Post("/", accessTokenVerifierMiddleware, routes.CreateReservation)
		reservation.Put("/:id/status", accessTokenVerifierMiddleware, routes.UpdateReservationStatus)
		reservation.Get("/user", accessTokenVerifierMiddleware, routes.GetUserReservations)
		reservation.Get("/property/:propertyId", accessTokenVerifierMiddleware, routes.GetPropertyReservations)
		reservation.Get("/user/all", accessTokenVerifierMiddleware, routes.GetUserReservations)
		reservation.Post("/:id/action", accessTokenVerifierMiddleware, routes.HandleReservation)
		reservation.Get("/:propertyId", accessTokenVerifierMiddleware, routes.GetPropertyReservations)
		reservation.Post("/payment", accessTokenVerifierMiddleware, routes.HandlePayment) // New payment route
		// Removed the undefined route for payment handling

	}

	app.Post("/api/refresh", refreshTokenVerifierMiddleware, utils.RefreshToken)

	// Start server
	app.Listen(":4000")
}
