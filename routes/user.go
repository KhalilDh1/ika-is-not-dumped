package routes

import (
	"apartments-clone-server/models"
	"apartments-clone-server/storage"
	"apartments-clone-server/utils"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/MicahParks/keyfunc"
	"github.com/golang-jwt/jwt/v4"
	"github.com/kataras/iris/v12"
	jsonWT "github.com/kataras/iris/v12/middleware/jwt"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/exp/slices"
)

func Register(ctx iris.Context) {
	var userInput RegisterUserInput
	err := ctx.ReadJSON(&userInput)
	if err != nil {
		utils.HandleValidationErrors(err, ctx)
		return
	}

	var newUser models.User
	userExists, userExistsErr := getAndHandleUserExists(&newUser, userInput.Email)
	if userExistsErr != nil {
		utils.CreateInternalServerError(ctx)
		return
	}

	if userExists == true {
		utils.CreateEmailAlreadyRegistered(ctx)
		return
	}

	hashedPassword, hashErr := hashAndSaltPassword(userInput.Password)
	if hashErr != nil {
		utils.CreateInternalServerError(ctx)
		return
	}

	newUser = models.User{
		FirstName:      userInput.FirstName,
		LastName:       userInput.LastName,
		Email:          strings.ToLower(userInput.Email),
		Password:       hashedPassword,
		SocialLogin:    false,
		MembershipTier: models.FreeTier,
		Avatar:         "https://static.vecteezy.com/system/resources/previews/013/485/975/original/letter-k-comic-style-typeface-with-transparent-background-file-png.png", // Set a default avatar or handle it accordingly
	}

	storage.DB.Create(&newUser)

	returnUser(newUser, ctx)
}

func Login(ctx iris.Context) {
	var userInput LoginUserInput
	err := ctx.ReadJSON(&userInput)
	if err != nil {
		utils.HandleValidationErrors(err, ctx)
		return
	}

	var existingUser models.User
	errorMsg := "Invalid email or password."
	userExists, userExistsErr := getAndHandleUserExists(&existingUser, userInput.Email)
	if userExistsErr != nil {
		utils.CreateInternalServerError(ctx)
		return
	}

	if userExists == false {
		utils.CreateError(iris.StatusUnauthorized, "Credentials Error", errorMsg, ctx)
		return
	}

	// Questionable as to whether you should let userInput know they logged in with Oauth
	// typically the fewer things said the better
	// If you don't want this, simply comment it out and the app will still work
	if existingUser.SocialLogin == true {
		utils.CreateError(iris.StatusUnauthorized, "Credentials Error", "Social Login Account", ctx)
		return
	}

	passwordErr := bcrypt.CompareHashAndPassword([]byte(existingUser.Password), []byte(userInput.Password))
	if passwordErr != nil {
		utils.CreateError(iris.StatusUnauthorized, "Credentials Error", errorMsg, ctx)
		return
	}

	returnUser(existingUser, ctx)
}

func FacebookLoginOrSignUp(ctx iris.Context) {
	var userInput FacebookOrGoogleUserInput
	err := ctx.ReadJSON(&userInput)
	if err != nil {
		utils.HandleValidationErrors(err, ctx)
		return
	}

	endpoint := "https://graph.facebook.com/me?fields=id,name,email&access_token=" + userInput.AccessToken
	client := &http.Client{}
	req, _ := http.NewRequest("GET", endpoint, nil)
	res, facebookErr := client.Do(req)
	if facebookErr != nil {
		utils.CreateInternalServerError(ctx)
		return
	}

	defer res.Body.Close()
	body, bodyErr := ioutil.ReadAll(res.Body)
	if bodyErr != nil {
		log.Panic(bodyErr)
		utils.CreateInternalServerError(ctx)
		return
	}

	var facebookBody FacebookUserRes
	json.Unmarshal(body, &facebookBody)

	if facebookBody.Email != "" {
		var user models.User
		userExists, userExistsErr := getAndHandleUserExists(&user, facebookBody.Email)

		if userExistsErr != nil {
			utils.CreateInternalServerError(ctx)
			return
		}

		if userExists == false {
			nameArr := strings.SplitN(facebookBody.Name, " ", 2)
			user = models.User{FirstName: nameArr[0], LastName: nameArr[1], Email: facebookBody.Email, SocialLogin: true, SocialProvider: "Facebook"}
			storage.DB.Create(&user)

			returnUser(user, ctx)
			return
		}

		if user.SocialLogin == true && user.SocialProvider == "Facebook" {
			returnUser(user, ctx)
			return
		}

		utils.CreateEmailAlreadyRegistered(ctx)
		return
	}
}

func GoogleLoginOrSignUp(ctx iris.Context) {
	var userInput FacebookOrGoogleUserInput
	err := ctx.ReadJSON(&userInput)
	if err != nil {
		utils.HandleValidationErrors(err, ctx)
		return
	}

	endpoint := "https://www.googleapis.com/userinfo/v2/me"

	client := &http.Client{}
	req, _ := http.NewRequest("GET", endpoint, nil)
	header := "Bearer " + userInput.AccessToken
	req.Header.Set("Authorization", header)
	res, googleErr := client.Do(req)
	if googleErr != nil {
		utils.CreateInternalServerError(ctx)
		return
	}

	defer res.Body.Close()
	body, bodyErr := ioutil.ReadAll(res.Body)
	if bodyErr != nil {
		log.Panic(bodyErr)
		utils.CreateInternalServerError(ctx)
		return
	}

	var googleBody GoogleUserRes
	json.Unmarshal(body, &googleBody)

	if googleBody.Email != "" {
		var user models.User
		userExists, userExistsErr := getAndHandleUserExists(&user, googleBody.Email)

		if userExistsErr != nil {
			utils.CreateInternalServerError(ctx)
			return
		}

		if userExists == false {
			user = models.User{FirstName: googleBody.GivenName, LastName: googleBody.FamilyName, Email: googleBody.Email, SocialLogin: true, SocialProvider: "Google"}
			storage.DB.Create(&user)

			returnUser(user, ctx)
			return
		}

		if user.SocialLogin == true && user.SocialProvider == "Google" {
			returnUser(user, ctx)
			return
		}

		utils.CreateEmailAlreadyRegistered(ctx)
		return

	}
}

func AppleLoginOrSignUp(ctx iris.Context) {
	var userInput AppleUserInput
	err := ctx.ReadJSON(&userInput)
	if err != nil {
		utils.HandleValidationErrors(err, ctx)
		return
	}

	res, httpErr := http.Get("https://appleid.apple.com/auth/keys")
	if httpErr != nil {
		utils.CreateInternalServerError(ctx)
		return
	}

	defer res.Body.Close()

	body, bodyErr := ioutil.ReadAll(res.Body)
	if bodyErr != nil {
		utils.CreateInternalServerError(ctx)
		return
	}

	jwks, jwksErr := keyfunc.NewJSON(body)
	//The JWKS.Keyfunc method will automatically select the key with the matching kid (if present) and return its public key as the correct Go type to its caller.
	token, tokenErr := jwt.Parse(userInput.IdentityToken, jwks.Keyfunc)

	if jwksErr != nil || tokenErr != nil {
		utils.CreateInternalServerError(ctx)
		return
	}

	if !token.Valid {
		utils.CreateError(iris.StatusUnauthorized, "Unauthorized", "Invalid user token.", ctx)
		return
	}

	email := fmt.Sprint(token.Claims.(jwt.MapClaims)["email"])
	if email != "" {
		var user models.User
		userExists, userExistsErr := getAndHandleUserExists(&user, email)

		if userExistsErr != nil {
			utils.CreateInternalServerError(ctx)
			return
		}

		if userExists == false {
			user = models.User{FirstName: "", LastName: "", Email: email, SocialLogin: true, SocialProvider: "Apple"}
			storage.DB.Create(&user)

			returnUser(user, ctx)
			return
		}

		if user.SocialLogin == true && user.SocialProvider == "Apple" {
			returnUser(user, ctx)
			return
		}

		utils.CreateEmailAlreadyRegistered(ctx)
		return
	}
}

func ForgotPassword(ctx iris.Context) {
	var emailInput EmailRegisteredInput
	err := ctx.ReadJSON(&emailInput)
	if err != nil {
		utils.HandleValidationErrors(err, ctx)
		return
	}

	var user models.User
	userExists, userExistsErr := getAndHandleUserExists(&user, emailInput.Email)

	if userExistsErr != nil {
		utils.CreateInternalServerError(ctx)
		return
	}

	if !userExists {
		utils.CreateError(iris.StatusUnauthorized, "Credentials Error", "Invalid email.", ctx)
		return
	}

	if userExists {
		if user.SocialLogin {
			utils.CreateError(iris.StatusUnauthorized, "Credentials Error", "Social Login Account", ctx)
			return
		}

		link := "exp://http://192.168.100.3:19000/--/resetpassword/"
		token, tokenErr := utils.CreateForgotPasswordToken(user.ID, user.Email)

		if tokenErr != nil {
			utils.CreateInternalServerError(ctx)
			return
		}

		link += token
		subject := "Forgot Your Password?"

		html := `
		<p>It looks like you forgot your password. 
		If you did, please click the link below to reset it. 
		If you did not, disregard this email. Please update your password
		within 10 minutes, otherwise you will have to repeat this
		process. <a href=` + link + `>Click to Reset Password</a>
		</p><br />`

		emailSent, emailSentErr := utils.SendMail(user.Email, subject, html)
		if emailSentErr != nil {
			utils.CreateInternalServerError(ctx)
			return
		}

		if emailSent {
			ctx.JSON(iris.Map{
				"emailSent": true,
			})
			return
		}

		ctx.JSON(iris.Map{"emailSent": false})
	}
}

func ResetPassword(ctx iris.Context) {
	var password ResetPasswordInput
	err := ctx.ReadJSON(&password)
	if err != nil {
		utils.HandleValidationErrors(err, ctx)
		return
	}

	hashedPassword, hashErr := hashAndSaltPassword(password.Password)
	if hashErr != nil {
		utils.CreateInternalServerError(ctx)
		return
	}

	claims := jsonWT.Get(ctx).(*utils.ForgotPasswordToken)

	var user models.User
	storage.DB.Model(&user).Where("id = ?", claims.ID).Update("password", hashedPassword)

	ctx.JSON(iris.Map{
		"passwordReset": true,
	})
}

func GetUserSavedProperties(ctx iris.Context) {
	params := ctx.Params()
	id := params.Get("id")

	user := getUserByID(id, ctx)
	if user == nil {
		return
	}

	var properties []models.Property
	var savedProperties []uint
	unmarshalErr := json.Unmarshal(user.SavedProperties, &savedProperties)
	if unmarshalErr != nil {
		utils.CreateInternalServerError(ctx)
		return
	}

	propertiesExist := storage.DB.Where("id IN ?", savedProperties).Find(&properties)

	if propertiesExist.Error != nil {
		utils.CreateInternalServerError(ctx)
		return
	}

	ctx.JSON(properties)
}

func AlterUserSavedProperties(ctx iris.Context) {
	params := ctx.Params()
	id := params.Get("id")

	user := getUserByID(id, ctx)
	if user == nil {
		return
	}

	var req AlterSavedPropertiesInput
	err := ctx.ReadJSON(&req)
	if err != nil {
		utils.HandleValidationErrors(err, ctx)
		return
	}

	propertyID := strconv.FormatUint(uint64(req.PropertyID), 10)

	validPropertyID := GetPropertyAndAssociationsByPropertyID(propertyID, ctx)

	if validPropertyID == nil {
		return
	}

	var savedProperties []uint
	var unMarshalledProperties []uint

	if user.SavedProperties != nil {
		unmarshalErr := json.Unmarshal(user.SavedProperties, &unMarshalledProperties)

		if unmarshalErr != nil {
			utils.CreateInternalServerError(ctx)
			return
		}
	}

	if req.Op == "add" {
		if !slices.Contains(unMarshalledProperties, req.PropertyID) {
			savedProperties = append(unMarshalledProperties, req.PropertyID)
		} else {
			savedProperties = unMarshalledProperties
		}
	} else if req.Op == "remove" && len(unMarshalledProperties) > 0 {
		for _, propertyID := range unMarshalledProperties {
			if req.PropertyID != propertyID {
				savedProperties = append(savedProperties, propertyID)
			}
		}
	}

	marshalledProperties, marshalErr := json.Marshal(savedProperties)

	if marshalErr != nil {
		utils.CreateInternalServerError(ctx)
		return
	}

	user.SavedProperties = marshalledProperties

	rowsUpdated := storage.DB.Model(&user).Updates(user)

	if rowsUpdated.Error != nil {
		utils.CreateInternalServerError(ctx)
		return
	}

	ctx.StatusCode(iris.StatusNoContent)
}

func GetUserContactedProperties(ctx iris.Context) {
	params := ctx.Params()
	id := params.Get("id")

	var conversations []models.Conversation
	conversationsExist := storage.DB.Where("tenant_id = ?", id).Find(&conversations)
	if conversationsExist.Error != nil {
		utils.CreateInternalServerError(ctx)
		return
	}

	if conversationsExist.RowsAffected == 0 {
		utils.CreateNotFound(ctx)
		return
	}

	var properties []models.Property
	var propertyIDs []uint
	for _, conversation := range conversations {
		propertyIDs = append(propertyIDs, conversation.PropertyID)
	}

	propertiesExist := storage.DB.Where("id IN ?", propertyIDs).Find(&properties)

	if propertiesExist.Error != nil {
		utils.CreateInternalServerError(ctx)
		return
	}

	ctx.JSON(properties)
}

func AlterPushToken(ctx iris.Context) {
	params := ctx.Params()
	id := params.Get("id")

	user := getUserByID(id, ctx)
	if user == nil {
		return
	}

	var req AlterPushTokenInput
	err := ctx.ReadJSON(&req)
	if err != nil {
		utils.HandleValidationErrors(err, ctx)
		return
	}

	var unMarshalledTokens []string
	var pushTokens []string

	if user.PushTokens != nil {
		unmarshalErr := json.Unmarshal(user.PushTokens, &unMarshalledTokens)

		if unmarshalErr != nil {
			utils.CreateInternalServerError(ctx)
			return
		}
	}

	if req.Op == "add" {
		if !slices.Contains(unMarshalledTokens, req.Token) {
			pushTokens = append(unMarshalledTokens, req.Token)
		} else {
			pushTokens = unMarshalledTokens
		}
	} else if req.Op == "remove" && len(unMarshalledTokens) > 0 {
		for _, token := range unMarshalledTokens {
			if req.Token != token {
				pushTokens = append(pushTokens, token)
			}
		}
	}

	marshalledTokens, marshalErr := json.Marshal(pushTokens)

	if marshalErr != nil {
		utils.CreateInternalServerError(ctx)
		return
	}

	user.PushTokens = marshalledTokens

	rowsUpdated := storage.DB.Model(&user).Updates(user)

	if rowsUpdated.Error != nil {
		utils.CreateInternalServerError(ctx)
		return
	}

	ctx.StatusCode(iris.StatusNoContent)
}

func AllowsNotifications(ctx iris.Context) {
	params := ctx.Params()
	id := params.Get("id")

	user := getUserByID(id, ctx)
	if user == nil {
		return
	}

	var req AllowsNotificationsInput
	err := ctx.ReadJSON(&req)
	if err != nil {
		utils.HandleValidationErrors(err, ctx)
		return
	}

	user.AllowsNotifications = req.AllowsNotifications

	rowsUpdated := storage.DB.Model(&user).Updates(user)

	if rowsUpdated.Error != nil {
		utils.CreateInternalServerError(ctx)
		return
	}

	ctx.StatusCode(iris.StatusNoContent)
}


func GetUserOwnedProperties(ctx iris.Context) {
	userID := ctx.Values().Get("userID").(uint)

	var properties []models.Property
	if err := storage.DB.Where("user_id = ?", userID).Find(&properties).Error; err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		ctx.JSON(iris.Map{"error": "Failed to retrieve properties"})
		return
	}

	ctx.JSON(properties)
}

func GetPropertyReservations(ctx iris.Context) {
	propertyID := ctx.Params().GetUintDefault("propertyId", 0)
	if propertyID == 0 {
		ctx.StatusCode(http.StatusBadRequest)
		ctx.JSON(iris.Map{"error": "Invalid property ID"})
		return
	}

	var reservations []models.Reservation
	if err := storage.DB.Where("property_id = ?", propertyID).Preload("User").Find(&reservations).Error; err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		ctx.JSON(iris.Map{"error": "Failed to retrieve reservations for this property"})
		return
	}

	ctx.JSON(reservations)
}

func getAndHandleUserExists(user *models.User, email string) (exists bool, err error) {
	userExistsQuery := storage.DB.Where("email = ?", strings.ToLower(email)).Limit(1).Find(&user)

	if userExistsQuery.Error != nil {
		return false, userExistsQuery.Error
	}

	userExists := userExistsQuery.RowsAffected > 0

	if userExists == true {
		return true, nil
	}

	return false, nil
}

func hashAndSaltPassword(password string) (hashedPassword string, err error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func getUserByID(id string, ctx iris.Context) *models.User {
	var user models.User
	userExists := storage.DB.Where("id = ?", id).Find(&user)

	if userExists.Error != nil {
		utils.CreateInternalServerError(ctx)
		return nil
	}

	if userExists.RowsAffected == 0 {
		utils.CreateError(iris.StatusNotFound, "Not Found", "User not found", ctx)
		return nil
	}

	return &user
}

func returnUser(user models.User, ctx iris.Context) {
	tokenPair, tokenErr := utils.CreateTokenPair(user.ID)
	if tokenErr != nil {
		utils.CreateInternalServerError(ctx)
		return
	}

	ctx.JSON(iris.Map{
		"ID":                  user.ID,
		"firstName":           user.FirstName,
		"lastName":            user.LastName,
		"email":               user.Email,
		"savedProperties":     user.SavedProperties,
		"allowsNotifications": user.AllowsNotifications,
		"accessToken":         string(tokenPair.AccessToken),
		"refreshToken":        string(tokenPair.RefreshToken),
		"membershipTier":      user.MembershipTier,
		"avatar":              user.Avatar, // Ensure this is included
	})
}

type RegisterUserInput struct {
	FirstName string `json:"firstName" validate:"required,max=256"`
	LastName  string `json:"lastName" validate:"required,max=256"`
	Email     string `json:"email" validate:"required,max=256,email"`
	Password  string `json:"password" validate:"required,min=8,max=256"`
}

type LoginUserInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type FacebookOrGoogleUserInput struct {
	AccessToken string `json:"accessToken" validate:"required"`
}

type AppleUserInput struct {
	IdentityToken string `json:"identityToken" validate:"required"`
}

type FacebookUserRes struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type GoogleUserRes struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	GivenName  string `json:"given_name"`
	FamilyName string `json:"family_name"`
}

type EmailRegisteredInput struct {
	Email string `json:"email" validate:"required"`
}

type ResetPasswordInput struct {
	Password string `json:"password" validate:"required,min=8,max=256"`
}

type AlterSavedPropertiesInput struct {
	PropertyID uint   `json:"propertyID" validate:"required"`
	Op         string `json:"op" validate:"required"`
}

type AlterPushTokenInput struct {
	Token string `json:"token" validate:"required"`
	Op    string `json:"op" validate:"required"`
}

type AllowsNotificationsInput struct {
	AllowsNotifications *bool `json:"allowsNotifications" validate:"required"`
}


// Add these functions to routes/user.go

func GetUserReservations(ctx iris.Context) {
    claims := utils.GetUserFromToken(ctx)
    if claims == nil {
        utils.CreateError(iris.StatusUnauthorized, "Authentication Error", "Invalid or expired token", ctx)
        return
    }

    var reservations []models.Reservation
    
    // Fetch both reservations made by the user and reservations for their properties
    if err := storage.DB.Where(
        "(user_id = ? OR owner_id = ?)", 
        claims.ID, 
        claims.ID,
    ).Preload("Property").Preload("User").Find(&reservations).Error; err != nil {
        utils.CreateError(iris.StatusInternalServerError, "Database Error", "Failed to fetch reservations", ctx)
        return
    }

    // Separate reservations into guest and host categories
    response := struct {
        AsGuest []models.Reservation `json:"asGuest"`
        AsHost  []models.Reservation `json:"asHost"`
    }{
        AsGuest: make([]models.Reservation, 0),
        AsHost:  make([]models.Reservation, 0),
    }

    for _, reservation := range reservations {
        if reservation.UserID == claims.ID {
            response.AsGuest = append(response.AsGuest, reservation)
        } else {
            response.AsHost = append(response.AsHost, reservation)
        }
    }

    ctx.JSON(response)
}

// func HandleReservation(ctx iris.Context) {
//     claims := utils.GetUserFromToken(ctx)
//     if claims == nil {
//         utils.CreateError(iris.StatusUnauthorized, "Authentication Error", "Invalid or expired token", ctx)
//         return
//     }

//     reservationID := ctx.Params().GetUintDefault("id", 0)
//     if reservationID == 0 {
//         utils.CreateError(iris.StatusBadRequest, "Invalid Input", "Invalid reservation ID", ctx)
//         return
//     }

//     var input struct {
//         Action string `json:"action" validate:"required,oneof=accept reject cancel"`
//     }
//     if err := ctx.ReadJSON(&input); err != nil {
//         utils.HandleValidationErrors(err, ctx)
//         return
//     }

//     var reservation models.Reservation
//     if err := storage.DB.Preload("Property").First(&reservation, reservationID).Error; err != nil {
//         utils.CreateError(iris.StatusNotFound, "Not Found", "Reservation not found", ctx)
//         return
//     }

//     // Verify user has permission to modify this reservation
//     if reservation.Property.UserID != claims.ID && reservation.UserID != claims.ID {
//         utils.CreateError(iris.StatusForbidden, "Forbidden", "Not authorized to modify this reservation", ctx)
//         return
//     }

//     // Handle different actions
//     switch input.Action {
//     case "accept":
//         if reservation.Property.UserID != claims.ID {
//             utils.CreateError(iris.StatusForbidden, "Forbidden", "Only property owner can accept reservations", ctx)
//             return
//         }
//         reservation.Status = models.Accepted
//     case "reject":
//         if reservation.Property.UserID != claims.ID {
//             utils.CreateError(iris.StatusForbidden, "Forbidden", "Only property owner can reject reservations", ctx)
//             return
//         }
//         reservation.Status = models.Rejected
//     case "cancel":
//         if reservation.Status == models.Accepted {
//             utils.CreateError(iris.StatusBadRequest, "Invalid Action", "Cannot cancel accepted reservation", ctx)
//             return
//         }
//         reservation.Status = models.Cancelled
//     }

//     if err := storage.DB.Save(&reservation).Error; err != nil {
//         utils.CreateError(iris.StatusInternalServerError, "Database Error", "Failed to update reservation", ctx)
//         return
//     }

//     ctx.JSON(reservation)
// }

// In routes/user.go
// routes/reservation.go

func HandleReservation(ctx iris.Context) {
    claims := utils.GetUserFromToken(ctx)
    if claims == nil {
        utils.CreateError(iris.StatusUnauthorized, "Authentication Error", "Invalid or expired token", ctx)
        return
    }

    reservationID := ctx.Params().GetUintDefault("id", 0)
    if reservationID == 0 {
        utils.CreateError(iris.StatusBadRequest, "Invalid Input", "Invalid reservation ID", ctx)
        return
    }

    var input struct {
        Action string `json:"action" validate:"required,oneof=accept reject cancel"`
    }
    if err := ctx.ReadJSON(&input); err != nil {
        utils.HandleValidationErrors(err, ctx)
        return
    }

    var reservation models.Reservation
    if err := storage.DB.Preload("Property").First(&reservation, reservationID).Error; err != nil {
        utils.CreateError(iris.StatusNotFound, "Not Found", "Reservation not found", ctx)
        return
    }

    // Verify permissions based on action
    switch input.Action {
    case "accept", "reject":
        if reservation.Property.UserID != claims.ID {
            utils.CreateError(iris.StatusForbidden, "Forbidden", "Only property owner can accept/reject reservations", ctx)
            return
        }
    case "cancel":
        if reservation.UserID != claims.ID {
            utils.CreateError(iris.StatusForbidden, "Forbidden", "Only the guest can cancel their reservation", ctx)
            return
        }
        // Don't allow cancellation of already accepted reservations
        if reservation.Status == models.Accepted {
            utils.CreateError(iris.StatusBadRequest, "Invalid Action", "Cannot cancel an accepted reservation", ctx)
            return
        }
    }

    // Update reservation status
    switch input.Action {
    case "accept":
        reservation.Status = models.Accepted
    case "reject":
        reservation.Status = models.Rejected
    case "cancel":
        reservation.Status = models.Cancelled
    }

    if err := storage.DB.Save(&reservation).Error; err != nil {
        utils.CreateError(iris.StatusInternalServerError, "Database Error", "Failed to update reservation", ctx)
        return
    }

    ctx.JSON(reservation)
}