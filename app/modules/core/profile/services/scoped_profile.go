package services

import (
	"encoding/json"

	entity "auth_service/infra/entities"
	"auth_service/shared/constants"
	"auth_service/shared/utils"
)

// The uuid prefix is what lets `EDITOR` exist in two organizations without the caller
// tripping over a key conflict that is not one.
const scopeSeparator = ":"

// The document of every organization's Admin profile, and the reason it is a wildcard
// rather than a copy of the ceiling: Resolve(ceiling, wildcard) collapses back onto
// the ceiling, whatever the ceiling becomes later. A copy would be a snapshot, and the
// owner would stay pinned to the ceiling of the day the organization was born.
var organizationAdminPermissions = json.RawMessage(`{"grants": ["as::*::*"]}`)

// ScopedKey is the only place a scoped key is built, so the two writers - the create
// endpoint and the birth of an organization - cannot disagree on what
// `<organization_id>:ADMIN` looks like.
//
// Returns "" when the name reduces to nothing, which is a 400 on the endpoint and
// impossible on the Admin row, whose name is a constant.
func ScopedKey(organizationId string, name string) string {
	identifier := utils.MacroCase(name)

	if identifier == "" {
		return ""
	}

	return organizationId + scopeSeparator + identifier
}

// OrganizationAdminProfile is the row every organization is born with, written inside
// the same transaction that creates the organization so a rollback undoes both. The
// owner participates on it instead of on the ceiling itself, which would otherwise be
// a row shared with every other organization born in the same pool.
//
// It is refused by ProfileService.writable, so the wildcard is what it stays.
func OrganizationAdminProfile(organizationId string) entity.Profile {
	return entity.Profile{
		Name:           constants.ProfileOrganizationAdmin,
		Key:            ScopedKey(organizationId, constants.ProfileOrganizationAdmin),
		OrganizationId: &organizationId,
		Permissions:    organizationAdminPermissions,
		Metadata:       json.RawMessage(`{}`),
	}
}
