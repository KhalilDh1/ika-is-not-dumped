// routes/reservation.go

package routes

import (
	"apartments-clone-server/models"
	"apartments-clone-server/storage"
	"apartments-clone-server/utils"

	// "fmt"
	"log"
	"os"
	"time"

	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/paymentintent"

	// "github.com/go-playground/validator/v10"
	"github.com/kataras/iris/v12"
)

// CreateReservation handles the creation of a new reservation
// func CreateReservation(ctx iris.Context) {
// 	// Get authenticated user from context
// 	stripe.Key = os.Getenv("STRIPE_SECRET_KEY") // Ensure this is set

// 	claims := utils.GetUserFromToken(ctx)
// 	if claims == nil {
// 		utils.CreateError(iris.StatusUnauthorized, "Authentication Error", "Invalid or expired token", ctx)
// 		return
// 	}

// 	var input CreateReservationInput
// 	if err := ctx.ReadJSON(&input); err != nil {
// 		utils.HandleValidationErrors(err, ctx)
// 		return
// 	}

// 	// Verify property exists and is available
// 	var property models.Property
// 	if err := storage.DB.First(&property, input.PropertyID).Error; err != nil {
// 		utils.CreateError(iris.StatusNotFound, "Not Found", "Property not found", ctx)
// 		return
// 	}

// 	// Verify user is not the property owner
// 	if property.UserID == claims.ID {
// 		utils.CreateError(iris.StatusBadRequest, "Invalid Request", "Cannot make reservation for your own property", ctx)
// 		return
// 	}

// 	// Check for date conflicts
// 	var conflictingReservations int64
// 	storage.DB.Model(&models.Reservation{}).
// 		Where("property_id = ? AND status != ? AND ((start_date BETWEEN ? AND ?) OR (end_date BETWEEN ? AND ?))",
// 			input.PropertyID,
// 			models.Cancelled,
// 			input.StartDate,
// 			input.EndDate,
// 			input.StartDate,
// 			input.EndDate).
// 		Count(&conflictingReservations)

// 	if conflictingReservations > 0 {
// 		utils.CreateError(iris.StatusConflict, "Date Conflict", "Selected dates are not available", ctx)
// 		return
// 	}

// 	// Create reservation
// 	reservation := models.Reservation{
// 		PropertyID:      input.PropertyID,
// 		UserID:          claims.ID,
// 		OwnerID:         property.UserID,
// 		StartDate:       input.StartDate,
// 		EndDate:         input.EndDate,
// 		GuestCount:      input.GuestCount,
// 		TotalPrice:      input.TotalPrice,
// 		Status:          models.Pending,
// 		PaymentStatus:   models.PaymentPending,
// 		SpecialRequests: input.SpecialRequests,
// 	}

// 	if err := storage.DB.Create(&reservation).Error; err != nil {
// 		utils.CreateError(iris.StatusInternalServerError, "Database Error", "Failed to create reservation", ctx)
// 		return
// 	}

// 	params := &stripe.PaymentIntentParams{
// 		Amount:   stripe.Int64(int64(reservation.TotalPrice * 100)), // Amount in cents
// 		Currency: stripe.String(string(stripe.CurrencyUSD)),         // Change to your currency
// 	}
// 	pi, err := paymentintent.New(params)
// 	if err != nil {
// 		utils.CreateError(iris.StatusInternalServerError, "Payment Error", "Failed to create payment intent", ctx)
// 		return
// 	}

// 	// Return the client secret to the frontend
// 	ctx.JSON(iris.Map{
// 		"clientSecret": pi.ClientSecret, // Ensure this is included in the response
// 		"reservation":  reservation,     // Include reservation details if needed
// 	})

// 	// Load associations for response
// 	storage.DB.Preload("Property").Preload("User").First(&reservation, reservation.ID)
// 	ctx.JSON(reservation)
// }

// UpdateReservationStatus handles status updates by property owners
func UpdateReservationStatus(ctx iris.Context) {
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

	var input UpdateReservationStatusInput
	if err := ctx.ReadJSON(&input); err != nil {
		utils.HandleValidationErrors(err, ctx)
		return
	}

	var reservation models.Reservation
	if err := storage.DB.Preload("Property").First(&reservation, reservationID).Error; err != nil {
		utils.CreateError(iris.StatusNotFound, "Not Found", "Reservation not found", ctx)
		return
	}

	// Verify user is the property owner
	if reservation.Property.UserID != claims.ID {
		utils.CreateError(iris.StatusForbidden, "Forbidden", "Not authorized to update this reservation", ctx)
		return
	}

	// Update status
	reservation.Status = input.Status
	if err := storage.DB.Save(&reservation).Error; err != nil {
		utils.CreateError(iris.StatusInternalServerError, "Database Error", "Failed to update reservation", ctx)
		return
	}

	ctx.JSON(reservation)
}

// func HandlePayment(ctx iris.Context) {
// 	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

// 	var input struct {
// 		ClientSecret string `json:"clientSecret" validate:"required"`
// 	}

// 	if err := ctx.ReadJSON(&input); err != nil {
// 		utils.HandleValidationErrors(err, ctx)
// 		return
// 	}

// 	// Confirm the payment using the client secret
// 	paymentIntent, err := paymentintent.Get(input.ClientSecret, nil)
// 	if err != nil {
// 		utils.CreateError(iris.StatusInternalServerError, "Payment Error", "Failed to confirm payment", ctx)
// 		return
// 	}

// 	// Check the payment status
// 	if paymentIntent.Status != stripe.PaymentIntentStatusSucceeded {
// 		utils.CreateError(iris.StatusBadRequest, "Payment Error", "Payment not successful", ctx)
// 		return
// 	}

// 	// Payment was successful, you can proceed with your logic here
// 	ctx.JSON(iris.Map{"message": "Payment successful"})
// }

// routes/reservation.go

func CreateReservation(ctx iris.Context) {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	claims := utils.GetUserFromToken(ctx)
	if claims == nil {
		utils.CreateError(iris.StatusUnauthorized, "Authentication Error", "Invalid or expired token", ctx)
		return
	}

	var input CreateReservationInput
	if err := ctx.ReadJSON(&input); err != nil {
		utils.HandleValidationErrors(err, ctx)
		return
	}

	// Verify property exists and is available
	var property models.Property
	if err := storage.DB.First(&property, input.PropertyID).Error; err != nil {
		utils.CreateError(iris.StatusNotFound, "Not Found", "Property not found", ctx)
		return
	}

	// Verify user is not the property owner
	if property.UserID == claims.ID {
		utils.CreateError(iris.StatusBadRequest, "Invalid Request", "Cannot make reservation for your own property", ctx)
		return
	}

	// Check for date conflicts
	var conflictingReservations int64
	storage.DB.Model(&models.Reservation{}).
		Where("property_id = ? AND status != ? AND ((start_date BETWEEN ? AND ?) OR (end_date BETWEEN ? AND ?))",
			input.PropertyID,
			models.Cancelled,
			input.StartDate,
			input.EndDate,
			input.StartDate,
			input.EndDate).
		Count(&conflictingReservations)

	if conflictingReservations > 0 {
		utils.CreateError(iris.StatusConflict, "Date Conflict", "Selected dates are not available", ctx)
		return
	}

	// Create Stripe PaymentIntent
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(int64(input.TotalPrice * 100)), // Convert to cents
		Currency: stripe.String(string(stripe.CurrencyUSD)),
		PaymentMethodTypes: []*string{
			stripe.String("card"),
		},
		// Metadata: map[string]string{
		// 	"propertyId": fmt.Sprintf("%d", input.PropertyID),
		// 	"userId":     fmt.Sprintf("%d", claims.ID),
		// 	"startDate":  input.StartDate.Format(time.RFC3339),
		// 	"endDate":    input.EndDate.Format(time.RFC3339),
		// },
	}

	pi, err := paymentintent.New(params)

	if err != nil {
		utils.CreateError(iris.StatusInternalServerError, "Payment Error", "Failed to create payment intent", ctx)
		return
	}

	// Create reservation with payment intent ID
	reservation := models.Reservation{
		PropertyID:      input.PropertyID,
		UserID:          claims.ID,
		OwnerID:         property.UserID,
		StartDate:       input.StartDate,
		EndDate:         input.EndDate,
		GuestCount:      input.GuestCount,
		TotalPrice:      input.TotalPrice,
		Status:          models.Pending,
		PaymentStatus:   models.PaymentPending,
		PaymentIntentID: pi.ID,
		SpecialRequests: input.SpecialRequests,
	}

	if err := storage.DB.Create(&reservation).Error; err != nil {
		// Attempt to cancel payment intent if reservation creation fails
		_, cancelErr := paymentintent.Cancel(pi.ID, nil)
		if cancelErr != nil {
			log.Printf("Failed to cancel payment intent: %v", cancelErr)
		}
		utils.CreateError(iris.StatusInternalServerError, "Database Error", "Failed to create reservation", ctx)
		return
	}

	// Load associations for response
	storage.DB.Preload("Property").Preload("User").First(&reservation, reservation.ID)

	// Return both the client secret and reservation details
	ctx.JSON(iris.Map{
		"clientSecret": pi.ClientSecret,
		"reservation":  reservation,
	})
}

func HandlePayment(ctx iris.Context) {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	var input struct {
		PaymentIntentID string `json:"paymentIntentId" validate:"required"`
	}

	if err := ctx.ReadJSON(&input); err != nil {
		utils.HandleValidationErrors(err, ctx)
		return
	}

	// Retrieve the payment intent
	pi, err := paymentintent.Get(input.PaymentIntentID, nil)
	if err != nil {
		utils.CreateError(iris.StatusInternalServerError, "Payment Error", "Failed to retrieve payment intent", ctx)
		return
	}

	// Update reservation payment status based on payment intent status
	var reservation models.Reservation
	if err := storage.DB.Where("payment_intent_id = ?", pi.ID).First(&reservation).Error; err != nil {
		utils.CreateError(iris.StatusNotFound, "Not Found", "Reservation not found", ctx)
		return
	}

	switch pi.Status {
	case stripe.PaymentIntentStatusSucceeded:
		reservation.PaymentStatus = models.PaymentSucceeded
	case stripe.PaymentIntentStatusCanceled, stripe.PaymentIntentStatusRequiresPaymentMethod:
		reservation.PaymentStatus = models.PaymentFailed
	default:
		reservation.PaymentStatus = models.PaymentPending
	}

	if err := storage.DB.Save(&reservation).Error; err != nil {
		utils.CreateError(iris.StatusInternalServerError, "Database Error", "Failed to update reservation", ctx)
		return
	}

	ctx.JSON(iris.Map{
		"status":      "success",
		"reservation": reservation,
	})
}

// Input structs
type CreateReservationInput struct {
	PropertyID      uint      `json:"propertyId" validate:"required"`
	StartDate       time.Time `json:"startDate" validate:"required"`
	EndDate         time.Time `json:"endDate" validate:"required,gtfield=StartDate"`
	GuestCount      int       `json:"guestCount" validate:"required,min=1"`
	TotalPrice      float64   `json:"totalPrice" validate:"required,min=0"`
	SpecialRequests string    `json:"specialRequests"`
}

type UpdateReservationStatusInput struct {
	Status models.ReservationStatus `json:"status" validate:"required,oneof=accepted rejected cancelled"`
}
