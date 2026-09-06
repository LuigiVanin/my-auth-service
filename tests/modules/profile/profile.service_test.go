package profile_test

import (
	"encoding/json"
	"testing"

	e "auth_service/app/errors"
	dto "auth_service/app/modules/core/profile/models"
	"auth_service/app/modules/core/profile/services"
	entity "auth_service/infra/entities"
	"auth_service/shared/constants"
	"auth_service/shared/permissions"
	mock "auth_service/tests/modules/mock"

	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

const organizationId = "11111111-1111-1111-1111-111111111111"

type ProfileServiceTestSuite struct {
	suite.Suite
	repository *mock.MockProfileRepository
	service    *services.ProfileService
}

func (this *ProfileServiceTestSuite) SetupTest() {
	this.repository = new(mock.MockProfileRepository)
	this.service = services.NewProfileService(this.repository, zap.NewNop())
}

// A manager shaped ceiling: wide enough to author from, narrow enough that
// as::apps::CREATE falls outside it.
func callerHolding(grants ...string) *services.ProfileWriteContext {
	document, _ := json.Marshal(permissions.Document{Grants: grants})

	return &services.ProfileWriteContext{
		Organization: &entity.Organization{
			ID:      organizationId,
			Profile: &entity.Profile{Permissions: document},
		},
		Participant: &entity.Participant{
			ID:        "participant-id",
			ProfileId: "caller-profile-id",
			Profile:   &entity.Profile{Permissions: document},
		},
	}
}

func codeOf(t assert.TestingT, err error) e.AppErrorCode {
	appError, ok := err.(*e.AppError)

	if !assert.True(t, ok, "expected an AppError, got %v", err) {
		return ""
	}

	return appError.Code.First
}

func (this *ProfileServiceTestSuite) TestCreateGeneratesTheScopedKey() {
	caller := callerHolding("as::users::READ")

	this.repository.
		On("Create", testifymock.Anything, testifymock.Anything).
		Return(&entity.Profile{ID: "created"}, nil)

	_, err := this.service.CreateForOrganization(caller, &dto.CreateProfile{
		Name:        "Gestão de Vendas",
		Permissions: dto.ProfilePermissions{Grants: []string{"as::users::READ"}},
	})

	this.NoError(err)

	written := this.repository.Calls[0].Arguments[0].(entity.Profile)

	this.Equal(organizationId+":GESTAO_DE_VENDAS", written.Key)
	this.Equal(organizationId, *written.OrganizationId)
}

// A name that reduces to nothing is a 400, never a key ending in the separator.
func (this *ProfileServiceTestSuite) TestCreateRefusesANameWithNoIdentifierInIt() {
	_, err := this.service.CreateForOrganization(callerHolding("as::users::READ"), &dto.CreateProfile{
		Name:        "!!!",
		Permissions: dto.ProfilePermissions{Grants: []string{"as::users::READ"}},
	})

	this.Equal(e.BadRequestCode.First, codeOf(this.T(), err))
	this.repository.AssertNotCalled(this.T(), "Create", testifymock.Anything, testifymock.Anything)
}

func (this *ProfileServiceTestSuite) TestCreateRefusesWhatExceedsTheCaller() {
	_, err := this.service.CreateForOrganization(callerHolding("as::users::READ"), &dto.CreateProfile{
		Name:        "Wider",
		Permissions: dto.ProfilePermissions{Grants: []string{"as::users::READ", "as::apps::CREATE"}},
	})

	this.Equal(e.PermissionDeniedErrorCode.First, codeOf(this.T(), err))
	this.repository.AssertNotCalled(this.T(), "Create", testifymock.Anything, testifymock.Anything)
}

// The platform administrator writes its ceiling in api and declares no grant at
// all, so a check that compared grant lists would leave it unable to author
// anything. The comparison is against the resolved api.
func (this *ProfileServiceTestSuite) TestCreateAllowsAnApiShapedCallerToAuthorGrants() {
	document := json.RawMessage(`{"api": {"*": {"methods": ["*"]}}}`)

	caller := &services.ProfileWriteContext{
		Organization: &entity.Organization{
			ID:      organizationId,
			Profile: &entity.Profile{Permissions: document},
		},
		Participant: &entity.Participant{
			ProfileId: "caller-profile-id",
			Profile:   &entity.Profile{Permissions: document},
		},
	}

	this.repository.
		On("Create", testifymock.Anything, testifymock.Anything).
		Return(&entity.Profile{ID: "created"}, nil)

	_, err := this.service.CreateForOrganization(caller, &dto.CreateProfile{
		Name:        "Editor",
		Permissions: dto.ProfilePermissions{Grants: []string{"as::apps::CREATE"}},
	})

	this.NoError(err)
}

func (this *ProfileServiceTestSuite) TestUpdateRefusesAGlobalProfile() {
	this.repository.
		On("FindByIdVisibleTo", "global-id", organizationId, testifymock.Anything).
		Return(&entity.Profile{ID: "global-id", Key: constants.ProfileManager}, nil)

	_, err := this.service.UpdateForOrganization("global-id", callerHolding("as::users::READ"), &dto.UpdateProfile{})

	this.Equal(e.PermissionDeniedErrorCode.First, codeOf(this.T(), err))
	this.repository.AssertNotCalled(this.T(), "Update", testifymock.Anything, testifymock.Anything, testifymock.Anything)
}

// Rule 6: the wildcard is what keeps the Admin row tracking the ceiling of its
// organization, so narrowing it would be a one way door.
func (this *ProfileServiceTestSuite) TestUpdateRefusesTheAdminProfileOfTheOrganization() {
	scope := organizationId

	this.repository.
		On("FindByIdVisibleTo", "admin-id", organizationId, testifymock.Anything).
		Return(&entity.Profile{
			ID:             "admin-id",
			Key:            services.ScopedKey(organizationId, constants.ProfileOrganizationAdmin),
			OrganizationId: &scope,
		}, nil)

	_, err := this.service.UpdateForOrganization("admin-id", callerHolding("as::users::READ"), &dto.UpdateProfile{})

	this.Equal(e.PermissionDeniedErrorCode.First, codeOf(this.T(), err))
	this.repository.AssertNotCalled(this.T(), "Update", testifymock.Anything, testifymock.Anything, testifymock.Anything)
}

// Rule 1: capping the document at what the caller holds is not enough on the row
// the caller itself participates with - widening it widens the caller.
func (this *ProfileServiceTestSuite) TestUpdateRefusesTheProfileTheCallerParticipatesWith() {
	scope := organizationId

	this.repository.
		On("FindByIdVisibleTo", "caller-profile-id", organizationId, testifymock.Anything).
		Return(&entity.Profile{
			ID:             "caller-profile-id",
			Key:            organizationId + ":EDITOR",
			OrganizationId: &scope,
		}, nil)

	_, err := this.service.UpdateForOrganization("caller-profile-id", callerHolding("as::users::READ"), &dto.UpdateProfile{})

	this.Equal(e.PermissionDeniedErrorCode.First, codeOf(this.T(), err))
	this.repository.AssertNotCalled(this.T(), "Update", testifymock.Anything, testifymock.Anything, testifymock.Anything)
}

// The endpoint speaks grants only, so a hand written api half survives a PUT. It
// is the granularity no grant expresses, and erasing it would be silent.
func (this *ProfileServiceTestSuite) TestUpdateKeepsTheApiHalfOfTheStoredDocument() {
	scope := organizationId

	stored := &entity.Profile{
		ID:             "editor-id",
		Key:            organizationId + ":EDITOR",
		OrganizationId: &scope,
		Permissions: json.RawMessage(
			`{"api": {"/core/users": {"methods": ["GET"], "query": {"skip": "^[0-9]+$"}}}, "grants": []}`,
		),
	}

	this.repository.On("FindByIdVisibleTo", "editor-id", organizationId, testifymock.Anything).Return(stored, nil)
	this.repository.
		On("Update", testifymock.Anything, testifymock.Anything, testifymock.Anything).
		Return(int64(1), nil)

	_, err := this.service.UpdateForOrganization("editor-id", callerHolding("as::users::READ"), &dto.UpdateProfile{
		Permissions: &dto.ProfilePermissions{Grants: []string{"as::users::READ"}},
	})

	this.NoError(err)

	for _, call := range this.repository.Calls {
		if call.Method != "Update" {
			continue
		}

		written, parseError := permissions.Parse(*call.Arguments[1].(dto.ProfileUpdateDao).Permissions)

		this.NoError(parseError)
		this.Equal([]string{"as::users::READ"}, written.Grants)
		this.Equal(map[string]string{"skip": "^[0-9]+$"}, written.Api["/core/users"].Query)

		return
	}

	this.Fail("the update never reached the repository")
}

// organization_id narrows what is already visible; it is not a lens into another
// organization.
func (this *ProfileServiceTestSuite) TestListingAnotherOrganizationIsAnEmptyPage() {
	page, err := this.service.FindAllVisibleTo(organizationId, &dto.GetProfilesQuery{
		OrganizationId: "22222222-2222-2222-2222-222222222222",
	})

	this.NoError(err)
	this.Empty(page.Data)
	this.repository.AssertNotCalled(this.T(), "FindVisibleTo", testifymock.Anything, testifymock.Anything)
}

func TestProfileServiceSuite(t *testing.T) {
	suite.Run(t, new(ProfileServiceTestSuite))
}
