package organization_test

import (
	"encoding/json"
	"testing"

	e "auth_service/app/errors"
	"auth_service/app/modules/core/organization/services"
	pdto "auth_service/app/modules/core/participant/models"
	entity "auth_service/infra/entities"
	"auth_service/shared/permissions"
	mock "auth_service/tests/modules/mock"

	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

const (
	organizationId = "11111111-1111-1111-1111-111111111111"
	ownerUserId    = uint(1)
)

type UpdateParticipantTestSuite struct {
	suite.Suite
	participants *mock.MockParticipantRepository
	profiles     *mock.MockProfileService
	service      *services.OrganizationService

	organization *entity.Organization
	caller       *entity.Participant
}

func document(grants ...string) json.RawMessage {
	encoded, _ := json.Marshal(permissions.Document{Grants: grants})

	return encoded
}

func (this *UpdateParticipantTestSuite) SetupTest() {
	this.participants = new(mock.MockParticipantRepository)
	this.profiles = new(mock.MockProfileService)

	this.service = services.NewOrganizationService(
		new(mock.MockOrganizationRepository),
		this.participants,
		new(mock.MockParticipantService),
		new(mock.MockProfileRepository),
		this.profiles,
		new(mock.MockUserPoolService),
		new(mock.MockUserService),
		nil,
		zap.NewNop(),
	)

	owner := ownerUserId

	// A ceiling wider than what the caller holds inside it: the difference between
	// the two is what rule 5 is about.
	this.organization = &entity.Organization{
		ID:          organizationId,
		OwnerUserId: &owner,
		Profile:     &entity.Profile{Permissions: document("as::users::READ", "as::apps::CREATE")},
	}

	this.caller = &entity.Participant{
		ID:        "caller-participation",
		UserId:    7,
		ProfileId: "narrow-profile",
		Profile:   &entity.Profile{Permissions: document("as::users::READ")},
	}
}

func (this *UpdateParticipantTestSuite) update(participantId string, profileId string) error {
	_, err := this.service.UpdateParticipant(
		organizationId,
		participantId,
		this.organization,
		this.caller,
		&pdto.UpdateParticipant{ProfileId: profileId},
	)

	return err
}

func codeOf(t assert.TestingT, err error) e.AppErrorCode {
	appError, ok := err.(*e.AppError)

	if !assert.True(t, ok, "expected an AppError, got %v", err) {
		return ""
	}

	return appError.Code.First
}

func (this *UpdateParticipantTestSuite) targetIs(participant *entity.Participant) {
	this.participants.
		On("FindOne", testifymock.Anything, testifymock.Anything).
		Return(participant, nil)
}

func (this *UpdateParticipantTestSuite) profileIs(profile *entity.Profile) {
	this.profiles.On("FindByIdVisibleTo", profile.ID, organizationId).Return(profile, nil)
}

// Rule 2. Without it, the grant to hand out a profile becomes the grant to take
// one.
func (this *UpdateParticipantTestSuite) TestRefusesTheCallersOwnParticipation() {
	this.targetIs(this.caller)

	err := this.update(this.caller.ID, "wide-profile")

	this.Equal(e.PermissionDeniedErrorCode.First, codeOf(this.T(), err))
	this.participants.AssertNotCalled(this.T(), "Update", testifymock.Anything, testifymock.Anything, testifymock.Anything)
}

// Rule 3, and the guarantee riding on it: an organization always keeps one
// administrator nobody can demote, so nobody can lock everyone out of it.
func (this *UpdateParticipantTestSuite) TestRefusesTheParticipationOfTheOwner() {
	this.targetIs(&entity.Participant{ID: "owner-participation", UserId: ownerUserId})

	err := this.update("owner-participation", "narrow-profile")

	this.Equal(e.PermissionDeniedErrorCode.First, codeOf(this.T(), err))
	this.participants.AssertNotCalled(this.T(), "Update", testifymock.Anything, testifymock.Anything, testifymock.Anything)
}

// Rule 5. Two narrow participants with the grant would otherwise promote each
// other and end up holding everything the organization holds.
func (this *UpdateParticipantTestSuite) TestRefusesAProfileWiderThanTheCallerHolds() {
	this.targetIs(&entity.Participant{ID: "other-participation", UserId: 9})
	this.profileIs(&entity.Profile{
		ID:          "wide-profile",
		Key:         organizationId + ":WIDE",
		Permissions: document("as::users::READ", "as::apps::CREATE"),
	})

	err := this.update("other-participation", "wide-profile")

	this.Equal(e.PermissionDeniedErrorCode.First, codeOf(this.T(), err))
	this.participants.AssertNotCalled(this.T(), "Update", testifymock.Anything, testifymock.Anything, testifymock.Anything)
}

// The comparison is against what the caller holds, not against the ceiling of the
// organization: the wide profile above fits the ceiling and is still refused.
func (this *UpdateParticipantTestSuite) TestAcceptsAProfileInsideWhatTheCallerHolds() {
	this.targetIs(&entity.Participant{ID: "other-participation", UserId: 9})
	this.profileIs(&entity.Profile{
		ID:          "fitting-profile",
		Key:         organizationId + ":READER",
		Permissions: document("as::users::READ"),
	})
	this.participants.
		On("Update", testifymock.Anything, testifymock.Anything, testifymock.Anything).
		Return(int64(1), nil)

	this.NoError(this.update("other-participation", "fitting-profile"))
}

// A profile scoped to another organization is invisible here, and invisible is
// indistinguishable from missing.
func (this *UpdateParticipantTestSuite) TestRefusesAProfileTheOrganizationCannotSee() {
	this.targetIs(&entity.Participant{ID: "other-participation", UserId: 9})
	this.profiles.On("FindByIdVisibleTo", "foreign-profile", organizationId).Return(nil, nil)

	err := this.update("other-participation", "foreign-profile")

	this.Equal(e.BadRequestCode.First, codeOf(this.T(), err))
}

// The PermissionsGuard answered the request against the organization the caller is
// currently in, so a write reaching another one would be authorized with powers
// held somewhere else.
func (this *UpdateParticipantTestSuite) TestRefusesAnOrganizationOtherThanTheCurrentOne() {
	_, err := this.service.UpdateParticipant(
		"22222222-2222-2222-2222-222222222222",
		"other-participation",
		this.organization,
		this.caller,
		&pdto.UpdateParticipant{ProfileId: "narrow-profile"},
	)

	this.Equal(e.PermissionDeniedErrorCode.First, codeOf(this.T(), err))
	this.participants.AssertNotCalled(this.T(), "FindOne", testifymock.Anything, testifymock.Anything)
}

func TestUpdateParticipantSuite(t *testing.T) {
	suite.Run(t, new(UpdateParticipantTestSuite))
}
