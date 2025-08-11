package model

import (
	"alc/repository"
	"github.com/google/uuid"
)

// AuthenticatedUser represents the user data stored in the context.
type AuthenticatedUser struct {
	ID    uuid.UUID
	Name  string
	Email string
	Role  repository.UserRole
}
