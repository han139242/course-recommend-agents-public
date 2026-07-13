package dto

// RegisterRequest defines the public student self-registration payload.
// Public registration never accepts privileged roles.
type RegisterRequest struct {
	Name     string `json:"name" binding:"omitempty,max=64"`
	Email    string `json:"email" binding:"required,email,max=128"`
	Password string `json:"password" binding:"required,min=6,max=72"`
	Role     string `json:"role" binding:"omitempty,oneof=student"`
}
