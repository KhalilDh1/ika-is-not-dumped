// package models

// import (
// 	"time"

// 	"gorm.io/gorm"
// )

// type ReservationStatus string
// type PaymentStatus string

// const (
//     Pending   ReservationStatus = "pending"
//     Accepted  ReservationStatus = "accepted"
//     Rejected  ReservationStatus = "rejected"
//     Cancelled ReservationStatus = "cancelled"

//     PaymentPending   PaymentStatus = "pending"
//     PaymentCompleted PaymentStatus = "completed"
//     PaymentFailed    PaymentStatus = "failed"
// )

// type Reservation struct {
//     gorm.Model
//     PropertyID      uint              `json:"propertyId"`
//     Property        Property          `json:"property"`
//     UserID          uint              `json:"userId"`
//     User            User              `json:"user"`
//     OwnerID         uint              `json:"ownerId"`
//     StartDate       time.Time         `json:"startDate"`
//     EndDate         time.Time         `json:"endDate"`
//     GuestCount      int               `json:"guestCount"`
//     TotalPrice      float64           `json:"totalPrice"`
//     Status          ReservationStatus `json:"status"`
//     PaymentStatus   PaymentStatus     `json:"paymentStatus"`
//     SpecialRequests string            `json:"specialRequests"`
// }

// models/reservation.go

package models

import (
	"time"

	"gorm.io/gorm"
)

type ReservationStatus string
type PaymentStatus string

const (
	Pending   ReservationStatus = "pending"
	Accepted  ReservationStatus = "accepted"
	Rejected  ReservationStatus = "rejected"
	Cancelled ReservationStatus = "cancelled"

	PaymentPending   PaymentStatus = "pending"
	PaymentSucceeded PaymentStatus = "succeeded"
	PaymentFailed    PaymentStatus = "failed"
)

type Reservation struct {
	gorm.Model
	PropertyID      uint              `json:"propertyId"`
	Property        Property          `json:"property"`
	UserID          uint              `json:"userId"`
	User            User              `json:"user"`
	OwnerID         uint              `json:"ownerId"`
	StartDate       time.Time         `json:"startDate"`
	EndDate         time.Time         `json:"endDate"`
	GuestCount      int               `json:"guestCount"`
	TotalPrice      float64           `json:"totalPrice"`
	Status          ReservationStatus `json:"status"`
	PaymentStatus   PaymentStatus     `json:"paymentStatus"`
	PaymentIntentID string            `json:"paymentIntentId"` // Add this field
	SpecialRequests string            `json:"specialRequests"`
}
