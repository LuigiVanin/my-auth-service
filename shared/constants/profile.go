package constants

// No endpoint creates a profile, so these keys are the whole universe a payload
// can name. All three are organization ceilings. A participant holds either the
// organization's own Admin profile or one the organization created for itself -
// there is no seeded participant role.
const (
	ProfileAdmin   = "ADMIN"
	ProfileManager = "MANAGER_PROFILE"
	ProfileLogin   = "LOGIN_PROFILE"

	// The name every organization's own admin profile is created with. It is not a
	// key: the key is generated as `<organization_id>:ADMIN`, so the row is scoped
	// and two organizations can both have one.
	ProfileOrganizationAdmin = "Admin"
)
