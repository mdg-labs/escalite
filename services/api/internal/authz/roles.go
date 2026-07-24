package authz

const (
	RoleAdmin  = "admin"
	RoleMember = "member"
)

// IsAdmin reports whether the user has org-wide admin privileges.
func IsAdmin(role string) bool {
	return role == RoleAdmin
}
