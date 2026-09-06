package constants

// No endpoint creates a profile, so these keys are the whole universe a payload
// can name. The first three are organization ceilings; ProfileMember is the only
// participant role, because the owner of an organization participates on that
// organization's own ceiling.
const (
	ProfileAdmin   = "ADMIN"
	ProfileManager = "MANAGER_PROFILE"
	ProfileLogin   = "LOGIN_PROFILE"

	ProfileMember = "MEMBER_PROFILE"

	// The name every organization's own admin profile is created with. It is not a
	// key: the key is generated as `<organization_id>:ADMIN`, so the row is scoped
	// and two organizations can both have one.
	ProfileOrganizationAdmin = "Admin"
)
