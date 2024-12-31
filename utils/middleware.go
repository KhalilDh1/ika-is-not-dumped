package utils

import (
	"net/http"
	"os"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/jwt"
)

func UserIDMiddleware(ctx iris.Context) {
	params := ctx.Params()
	id := params.Get("id")

	claims := jwt.Get(ctx).(*AccessToken)

	userID := strconv.FormatUint(uint64(claims.ID), 10)

	if userID != id {
		ctx.StatusCode(iris.StatusForbidden)
		return
	}
	ctx.Next()
}

func AccessTokenVerifierMiddleware(ctx iris.Context) {
	auth := ctx.GetHeader("Authorization")
	if auth == "" {
		ctx.StatusCode(http.StatusUnauthorized)
		ctx.JSON(iris.Map{"error": "No authorization header"})
		return
	}

	token := auth[7:] // Remove "Bearer " prefix

	verifier := jwt.NewVerifier(jwt.HS256, []byte(os.Getenv("ACCESS_TOKEN_SECRET")))
	verifier.WithDefaultBlocklist()

	verifiedToken, err := verifier.VerifyToken([]byte(token))
	if err != nil {
		ctx.StatusCode(http.StatusUnauthorized)
		ctx.JSON(iris.Map{"error": "Invalid token"})
		return
	}

	var customClaims AccessToken
	if err := verifiedToken.Claims(&customClaims); err != nil {
		ctx.StatusCode(http.StatusUnauthorized)
		ctx.JSON(iris.Map{"error": "Invalid token claims"})
		return
	}

	// Set user ID in context
	ctx.Values().Set("userID", customClaims.ID)

	ctx.Next()
}


func GetUserFromToken(ctx iris.Context) *AccessToken {
	token := jwt.Get(ctx)
	if token == nil {
		return nil
	}
	
	claims, ok := token.(*AccessToken)
	if !ok {
		return nil
	}
	
	return claims
}


// TokenPair represents both access and refresh tokens
type TokenPair struct {
	AccessToken  []byte
	RefreshToken []byte
}

func AddCustomValidation(tag string, fn validator.Func) {

    validate := validator.New()

    validate.RegisterValidation(tag, fn)

}