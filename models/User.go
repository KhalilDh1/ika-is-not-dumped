package models

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type MembershipTier string

const (
	FreeTier    MembershipTier = "Free"
	PremiumTier MembershipTier = "Premium"
	ProTier     MembershipTier = "Pro"
)

type User struct {
	gorm.Model
	FirstName           string         `json:"firstName"`
	LastName            string         `json:"lastName"`
	Email               string         `json:"email"`
	Password            string         `json:"password"`
	SocialLogin         bool           `json:"socialLogin"`
	SocialProvider      string         `json:"socialProvider"`
	Properties          []Property     `json:"properties"`
	SavedProperties     datatypes.JSON `json:"savedProperties"`
	PushTokens          datatypes.JSON `json:"pushTokens"`
	AllowsNotifications *bool          `json:"allowsNotifications"`
	MembershipTier      MembershipTier `json:"membershipTier" gorm:"type:membership_tier;default:'Free'"`
	Avatar              string         `json:"avatar"` // Ensure this field is included
}
