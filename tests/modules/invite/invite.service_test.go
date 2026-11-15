package invite_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	e "auth_service/app/errors"
	dto "auth_service/app/modules/core/invite/models"
	"auth_service/app/modules/core/invite/services"
	otpdto "auth_service/app/modules/core/otp/models"
	os "auth_service/app/modules/core/otp/services"
	udto "auth_service/app/modules/core/user/models"
	"auth_service/infra/config"
	entity "auth_service/infra/entities"
	"auth_service/shared/constants"
	"auth_service/shared/email"
	"auth_service/shared/permissions"
	repo "auth_service/shared/repository"
	"auth_service/shared/storage"
	mock "auth_service/tests/modules/mock"

	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

const (
	organizationId = "11111111-1111-1111-1111-111111111111"
	poolId         = "pool-id"
	appId          = "app-id"
	otpId          = "22222222-2222-2222-2222-222222222222"
	secret         = "link-secret"
	invitedEmail   = "guest@example.com"
	acceptUrl      = "https://app.test/invite?token={token}&organization={organization_id}"
)

var anything = testifymock.Anything

func document(grants ...string) json.RawMessage {
	encoded, _ := json.Marshal(permissions.Document{Grants: grants})

	return encoded
}

func codeOf(t assert.TestingT, err error) e.AppErrorCode {
	appError, ok := err.(*e.AppError)

	if !assert.True(t, ok, "expected an AppError, got %v", err) {
		return ""
	}

	return appError.Code.First
}

type InviteServiceTestSuite struct {
	suite.Suite

	invites      *mock.MockInviteRepository
	participants *mock.MockParticipantRepository
	users        *mock.MockUserRepository
	participant  *mock.MockParticipantService
	profiles     *mock.MockProfileService
	userService  *mock.MockUserService
	otps         *mock.MockOtpService
	txManager    *mock.MockTransactionManager
	mailbox      *mock.FakeEmailAdapter
	cfg          *config.Config

	tx     *repo.Tx
	option []repo.Option

	service *services.InviteService

	app          *entity.App
	organization *entity.Organization
	caller       *entity.Participant
	inviter      *entity.User
}

func (this *InviteServiceTestSuite) SetupTest() {
	this.invites = new(mock.MockInviteRepository)
	this.participants = new(mock.MockParticipantRepository)
	this.users = new(mock.MockUserRepository)
	this.participant = new(mock.MockParticipantService)
	this.profiles = new(mock.MockProfileService)
	this.userService = new(mock.MockUserService)
	this.otps = new(mock.MockOtpService)
	this.txManager = new(mock.MockTransactionManager)
	this.mailbox = &mock.FakeEmailAdapter{}
	this.cfg = &config.Config{
		App:    config.AppConfig{Env: "development"},
		Invite: config.InviteConfig{AcceptUrl: acceptUrl},
	}

	this.tx = &repo.Tx{}
	this.option = []repo.Option{{Tx: this.tx}}

	this.service = services.NewInviteService(
		this.invites,
		this.participants,
		this.users,
		this.participant,
		this.profiles,
		this.userService,
		this.otps,
		this.txManager,
		email.NewEmailManager(this.mailbox),
		this.cfg,
		zap.NewNop(),
		storage.NewUrlResolver("https://cdn.test"),
	)

	this.app = &entity.App{ID: appId, UsersPool: entity.UsersPool{ID: poolId}}

	// A ceiling wider than what the caller holds inside it, so "within the
	// organization" and "within the caller" are different answers.
	this.organization = &entity.Organization{
		ID:          organizationId,
		Name:        "Acme",
		UsersPoolId: poolId,
		Profile:     &entity.Profile{Permissions: document("as::users::READ", "as::apps::CREATE")},
	}

	this.caller = &entity.Participant{
		ID:      "caller-participation",
		UserId:  7,
		Profile: &entity.Profile{Permissions: document("as::users::READ")},
	}

	this.inviter = &entity.User{ID: 7, Name: "Ada <admin>"}
}

func (this *InviteServiceTestSuite) narrowProfile() *entity.Profile {
	profile := &entity.Profile{ID: "narrow-profile", Key: "READER", Permissions: document("as::users::READ")}
	this.profiles.On("FindByIdVisibleTo", profile.ID, organizationId).Return(profile, nil)

	return profile
}

func (this *InviteServiceTestSuite) create(payload *dto.CreateInvite) (*dto.InviteResponse, error) {
	return this.service.Create(organizationId, this.organization, this.caller, this.inviter, this.app, payload)
}

func payloadFor(profileId string) *dto.CreateInvite {
	return &dto.CreateInvite{Email: "  Guest@Example.com ", ProfileId: profileId}
}

// Everything a successful creation touches, past the refusals.
func (this *InviteServiceTestSuite) arrangeIssue(ttl time.Duration) {
	this.userService.On("FindUserInPool", invitedEmail, poolId).Return(nil, nil)
	this.invites.On("FindPendingFor", organizationId, invitedEmail, anything).Return([]entity.Invite{}, nil)

	expires := time.Now().Add(ttl)

	this.otps.
		On("Issue", constants.ActionInvite, this.app, invitedEmail, ttl).
		Return(&os.IssuedOtp{Otp: &entity.Otp{ID: otpId, ExpiresAt: expires}, Secret: secret}, nil)

	this.invites.
		On("Create", anything, anything).
		Return(func(invite entity.Invite, _ ...repo.Option) *entity.Invite {
			invite.ID = "invite-id"
			return &invite
		}, nil)
}

func (this *InviteServiceTestSuite) TestCreateRefusesAnotherOrganization() {
	_, err := this.service.Create("another-organization", this.organization, this.caller, this.inviter, this.app, payloadFor("narrow-profile"))

	this.Equal(e.PermissionDeniedErrorCode.First, codeOf(this.T(), err))
	this.otps.AssertNotCalled(this.T(), "Issue", anything, anything, anything, anything)
}

func (this *InviteServiceTestSuite) TestCreateAnswers501WithoutAnAcceptPage() {
	this.cfg.Invite.AcceptUrl = ""

	_, err := this.create(payloadFor("narrow-profile"))

	this.Equal(e.NotImplementedErrorCode.First, codeOf(this.T(), err))
}

func (this *InviteServiceTestSuite) TestCreateRefusesAnAppOfAnotherPool() {
	this.app.UsersPool.ID = "another-pool"

	_, err := this.create(payloadFor("narrow-profile"))

	this.Equal(e.PermissionDeniedErrorCode.First, codeOf(this.T(), err))
}

func (this *InviteServiceTestSuite) TestCreateRefusesAProfileTheOrganizationCannotSee() {
	this.profiles.On("FindByIdVisibleTo", "foreign-profile", organizationId).Return(nil, nil)

	_, err := this.create(payloadFor("foreign-profile"))

	this.Equal(e.BadRequestCode.First, codeOf(this.T(), err))
}

// RC-1. Within the organization but beyond the caller: two narrow participants
// would otherwise invite each other's accounts in as administrators.
func (this *InviteServiceTestSuite) TestCreateRefusesAProfileWiderThanTheCaller() {
	wide := &entity.Profile{ID: "wide-profile", Key: "WIDE", Permissions: document("as::users::READ", "as::apps::CREATE")}
	this.profiles.On("FindByIdVisibleTo", wide.ID, organizationId).Return(wide, nil)

	_, err := this.create(payloadFor(wide.ID))

	this.Equal(e.PermissionDeniedErrorCode.First, codeOf(this.T(), err))
	this.otps.AssertNotCalled(this.T(), "Issue", anything, anything, anything, anything)
}

// RC-5.
func (this *InviteServiceTestSuite) TestCreateRefusesSomeoneWhoAlreadyParticipates() {
	this.narrowProfile()
	member := &entity.User{ID: 9, Email: invitedEmail}

	this.userService.On("FindUserInPool", invitedEmail, poolId).Return(member, nil)
	this.participant.On("FindForUserInOrganization", organizationId, member.ID).Return(&entity.Participant{ID: "p"}, nil)

	_, err := this.create(payloadFor("narrow-profile"))

	this.Equal(e.ConflictErrorCode.First, codeOf(this.T(), err))
}

func (this *InviteServiceTestSuite) TestCreateIssuesAnOtpAndEmailsTheLink() {
	this.narrowProfile()
	this.arrangeIssue(services.DefaultInviteTtl)

	invite, err := this.create(payloadFor("narrow-profile"))

	this.NoError(err)
	this.Equal(dto.InviteStatusPending, invite.Status)
	this.Equal(invitedEmail, invite.Email)
	this.Equal(otpId, invite.OtpId)

	this.invites.AssertCalled(this.T(), "Create", testifymock.MatchedBy(func(created entity.Invite) bool {
		return created.Email == invitedEmail &&
			created.OtpId == otpId &&
			created.ProfileId == "narrow-profile" &&
			created.InvitedByUserId == this.inviter.ID &&
			!created.ExpiresAt.IsZero()
	}), anything)

	if this.Len(this.mailbox.Sent, 1) {
		sent := this.mailbox.Sent[0]

		this.Equal([]string{invitedEmail}, sent.To)
		this.Contains(sent.Body, "token="+otpId+"."+secret)
		this.Contains(sent.Body, "organization="+organizationId)
		this.Contains(sent.Body, "Ada &lt;admin&gt;")
	}
}

// D-4: every pending invite for the address is revoked, and its OTP with it.
func (this *InviteServiceTestSuite) TestCreateRevokesThePendingInvitesItReplaces() {
	this.narrowProfile()
	this.arrangeIssue(services.DefaultInviteTtl)

	this.invites.ExpectedCalls = nil
	this.arrangeIssueWithPending([]entity.Invite{{ID: "old-1", OtpId: "old-otp-1"}, {ID: "old-2", OtpId: "old-otp-2"}})

	_, err := this.create(payloadFor("narrow-profile"))

	this.NoError(err)
	this.invites.AssertCalled(this.T(), "Revoke", "old-1", anything, anything)
	this.invites.AssertCalled(this.T(), "Revoke", "old-2", anything, anything)
	this.otps.AssertCalled(this.T(), "Invalidate", "old-otp-1")
	this.otps.AssertCalled(this.T(), "Invalidate", "old-otp-2")
}

func (this *InviteServiceTestSuite) arrangeIssueWithPending(pending []entity.Invite) {
	this.invites.On("FindPendingFor", organizationId, invitedEmail, anything).Return(pending, nil)
	this.invites.On("Revoke", anything, anything, anything).Return(int64(1), nil)
	this.otps.On("Invalidate", anything).Return()
	this.invites.
		On("Create", anything, anything).
		Return(func(invite entity.Invite, _ ...repo.Option) *entity.Invite { return &invite }, nil)
}

// D-7: the payload, then the organization, then the service, then the default.
func (this *InviteServiceTestSuite) TestCreateResolvesTheValidityInOrder() {
	day := int64(24 * 60 * 60)
	hours := int64(2 * 60 * 60)

	cases := map[string]struct {
		payload, organization *int64
		service               int64
		want                  time.Duration
	}{
		"payload wins":           {payload: &hours, organization: &day, service: 3 * day, want: 2 * time.Hour},
		"organization default":   {organization: &day, service: 3 * day, want: 24 * time.Hour},
		"service configuration":  {service: 3 * day, want: 72 * time.Hour},
		"default of the service": {want: services.DefaultInviteTtl},
	}

	for name, scenario := range cases {
		this.Run(name, func() {
			this.SetupTest()
			this.narrowProfile()
			this.organization.InviteTtlSeconds = scenario.organization
			this.cfg.Invite.TtlSeconds = scenario.service
			this.arrangeIssue(scenario.want)

			payload := payloadFor("narrow-profile")
			payload.TtlSeconds = scenario.payload

			_, err := this.create(payload)

			this.NoError(err)
			this.otps.AssertCalled(this.T(), "Issue", constants.ActionInvite, this.app, invitedEmail, scenario.want)
		})
	}
}

func (this *InviteServiceTestSuite) TestCreateAnswers500WhenTheEmailFails() {
	this.narrowProfile()
	this.arrangeIssue(services.DefaultInviteTtl)
	this.mailbox.Err = errors.New("resend is down")

	_, err := this.create(payloadFor("narrow-profile"))

	this.Equal(e.InternalServerErrorCode.First, codeOf(this.T(), err))
}

func (this *InviteServiceTestSuite) pendingInvite() *entity.Invite {
	return &entity.Invite{
		ID:             "invite-id",
		OrganizationId: organizationId,
		ProfileId:      "narrow-profile",
		Email:          invitedEmail,
		OtpId:          otpId,
		ExpiresAt:      time.Now().Add(time.Hour),
		Organization:   this.organization,
		Profile:        &entity.Profile{ID: "narrow-profile", Name: "Reader", Key: "READER"},
		InvitedBy:      this.inviter,
		Otp:            &entity.Otp{ID: otpId, AppId: appId},
	}
}

func (this *InviteServiceTestSuite) inviteIs(invite *entity.Invite) {
	this.invites.On("FindOne", entity.Invite{OtpId: otpId}, anything).Return(invite, nil)
}

func (this *InviteServiceTestSuite) secretIs(err error) {
	this.otps.
		On("ValidateConsumable", otpdto.PayloadOtpData{Id: otpId, Code: secret}, appId, constants.ActionInvite).
		Return(&entity.Otp{ID: otpId}, err)
}

func (this *InviteServiceTestSuite) TestResolveAnswers404ForAMalformedToken() {
	for _, token := range []string{"", "no-separator", "not-a-uuid.secret", otpId + "."} {
		_, err := this.service.Resolve(token, this.app)

		this.Equal(e.NotFoundErrorCode.First, codeOf(this.T(), err), token)
	}

	this.invites.AssertNotCalled(this.T(), "FindOne", anything, anything)
}

func (this *InviteServiceTestSuite) TestResolveAnswers404ForAnUnknownToken() {
	this.inviteIs(nil)

	_, err := this.service.Resolve(otpId+"."+secret, this.app)

	this.Equal(e.NotFoundErrorCode.First, codeOf(this.T(), err))
}

// RC-3: for another application the invite does not exist.
func (this *InviteServiceTestSuite) TestResolveAnswers404ToAnotherApp() {
	invite := this.pendingInvite()
	invite.Otp.AppId = "another-app"
	this.inviteIs(invite)

	_, err := this.service.Resolve(otpId+"."+secret, this.app)

	this.Equal(e.NotFoundErrorCode.First, codeOf(this.T(), err))
	this.otps.AssertNotCalled(this.T(), "ValidateConsumable", anything, anything, anything)
}

func (this *InviteServiceTestSuite) TestResolveTellsWhyAnInviteIsNoLongerPending() {
	now := time.Now()

	for name, change := range map[string]func(*entity.Invite){
		"accepted": func(invite *entity.Invite) { invite.AcceptedAt = &now },
		"revoked":  func(invite *entity.Invite) { invite.RevokedAt = &now },
		"expired":  func(invite *entity.Invite) { invite.ExpiresAt = now.Add(-time.Minute) },
	} {
		this.Run(name, func() {
			this.SetupTest()
			invite := this.pendingInvite()
			change(invite)
			this.inviteIs(invite)

			_, err := this.service.Resolve(otpId+"."+secret, this.app)

			this.Equal(e.BadRequestCode.First, codeOf(this.T(), err))
			this.Contains(err.(*e.AppError).Detail, name)
		})
	}
}

// RC-9: a wrong secret reads exactly like an invite that does not exist.
func (this *InviteServiceTestSuite) TestResolveAnswers404ForAWrongSecret() {
	this.inviteIs(this.pendingInvite())
	this.secretIs(e.ThrowBadRequest("Invalid otp code"))

	_, err := this.service.Resolve(otpId+"."+secret, this.app)

	this.Equal(e.NotFoundErrorCode.First, codeOf(this.T(), err))
}

func (this *InviteServiceTestSuite) TestResolveKeepsAServerErrorOfTheOtp() {
	this.inviteIs(this.pendingInvite())
	this.secretIs(e.ThrowInternalServerError("Failed to compare otp code"))

	_, err := this.service.Resolve(otpId+"."+secret, this.app)

	this.Equal(e.InternalServerErrorCode.First, codeOf(this.T(), err))
}

func (this *InviteServiceTestSuite) TestDescribeTellsWhetherTheAddressHasAnAccount() {
	this.inviteIs(this.pendingInvite())
	this.secretIs(nil)
	this.userService.On("FindUserInPool", invitedEmail, poolId).Return(&entity.User{ID: 9}, nil)

	description, err := this.service.Describe(otpId+"."+secret, this.app)

	this.NoError(err)
	this.True(description.AccountExists)
	this.Equal("Acme", description.Organization.Name)
	this.Equal("READER", description.Profile.Key)
	this.Equal(this.inviter.Name, description.InvitedBy.Name)
}

func (this *InviteServiceTestSuite) TestAcceptAnswers404WhenTheAddressHasNoAccount() {
	this.inviteIs(this.pendingInvite())
	this.secretIs(nil)
	this.userService.On("FindUserInPool", invitedEmail, poolId).Return(nil, nil)

	_, err := this.service.Accept(otpId+"."+secret, this.app)

	this.Equal(e.NotFoundErrorCode.First, codeOf(this.T(), err))
	this.txManager.AssertNotCalled(this.T(), "Tx")
}

// The unit of work of the accept, past Resolve. Returns the channel the detached
// OTP invalidation reports on: Detach swallows a panic, so a channel is the
// only reliable assertion.
func (this *InviteServiceTestSuite) arrangeConsume(user *entity.User, existing *entity.Participant) chan string {
	this.profiles.On("FindByIdVisibleTo", "narrow-profile", organizationId).
		Return(&entity.Profile{ID: "narrow-profile"}, nil)

	this.txManager.On("Tx").Return(this.tx, nil)
	this.participants.On("FindByOrganizationAndUser", organizationId, user.ID, this.option).Return(existing, nil)
	this.participants.On("Create", anything, this.option).Return(&entity.Participant{ID: "new-participation"}, nil)
	this.invites.On("Accept", "invite-id", anything, this.option).Return(int64(1), nil)
	this.users.On("Update", anything, anything, this.option).Return(int64(1), nil)
	this.participants.On("FindOne", anything, anything).Return(&entity.Participant{ID: "read-back", User: user}, nil)

	invalidated := make(chan string, 1)
	this.otps.On("Invalidate", anything).Run(func(args testifymock.Arguments) {
		invalidated <- args.String(0)
	}).Return()

	return invalidated
}

func (this *InviteServiceTestSuite) TestAcceptCreatesTheParticipationAndConsumesTheInvite() {
	user := &entity.User{ID: 9, Email: invitedEmail, UsersPoolId: poolId}

	this.inviteIs(this.pendingInvite())
	this.secretIs(nil)
	this.userService.On("FindUserInPool", invitedEmail, poolId).Return(user, nil)
	invalidated := this.arrangeConsume(user, nil)

	accepted, err := this.service.Accept(otpId+"."+secret, this.app)

	this.NoError(err)
	this.Equal(organizationId, accepted.Organization.ID)

	this.participants.AssertCalled(this.T(), "Create", entity.Participant{
		OrganizationId: organizationId,
		UserId:         user.ID,
		ProfileId:      "narrow-profile",
	}, this.option)
	this.invites.AssertCalled(this.T(), "Accept", "invite-id", anything, this.option)

	// RC-8: consuming the link proves the address.
	verified := true
	this.users.AssertCalled(this.T(), "Update", entity.User{ID: user.ID}, udto.UserUpdateDao{VerifyEmail: &verified}, this.option)

	select {
	case id := <-invalidated:
		this.Equal(otpId, id)
	case <-time.After(2 * time.Second):
		this.T().Fatal("the otp of the invite was never invalidated")
	}
}

// RC-6: the existing participation keeps its profile, and the invite is consumed.
func (this *InviteServiceTestSuite) TestAcceptKeepsAnExistingParticipation() {
	user := &entity.User{ID: 9, Email: invitedEmail, UsersPoolId: poolId, VerifyEmail: true}

	this.inviteIs(this.pendingInvite())
	this.secretIs(nil)
	this.userService.On("FindUserInPool", invitedEmail, poolId).Return(user, nil)
	this.arrangeConsume(user, &entity.Participant{ID: "existing", ProfileId: "admin-profile"})

	_, err := this.service.Accept(otpId+"."+secret, this.app)

	this.NoError(err)
	this.participants.AssertNotCalled(this.T(), "Create", anything, anything)
	this.users.AssertNotCalled(this.T(), "Update", anything, anything, anything)
	this.invites.AssertCalled(this.T(), "Accept", "invite-id", anything, this.option)
}

// RC-13: two accepts at once end with one participation - the loser's
// transaction is rolled back.
func (this *InviteServiceTestSuite) TestAcceptRefusesTheLoserOfARace() {
	user := &entity.User{ID: 9, Email: invitedEmail, UsersPoolId: poolId}

	this.inviteIs(this.pendingInvite())
	this.secretIs(nil)
	this.userService.On("FindUserInPool", invitedEmail, poolId).Return(user, nil)
	this.arrangeConsume(user, nil)
	this.invites.ExpectedCalls = nil
	this.inviteIs(this.pendingInvite())
	this.invites.On("Accept", "invite-id", anything, this.option).Return(int64(0), nil)

	_, err := this.service.Accept(otpId+"."+secret, this.app)

	this.Equal(e.BadRequestCode.First, codeOf(this.T(), err))
	this.otps.AssertNotCalled(this.T(), "Invalidate", anything)
}

func (this *InviteServiceTestSuite) TestAcceptForRefusesAProfileTheOrganizationNoLongerSees() {
	user := &entity.User{ID: 9, UsersPoolId: poolId}
	this.profiles.On("FindByIdVisibleTo", "narrow-profile", organizationId).Return(nil, nil)

	_, err := this.service.AcceptFor(this.pendingInvite(), user)

	this.Equal(e.BadRequestCode.First, codeOf(this.T(), err))
	this.txManager.AssertNotCalled(this.T(), "Tx")
}

func (this *InviteServiceTestSuite) TestListRefusesAnUnknownStatus() {
	_, err := this.service.List(organizationId, this.organization, &dto.GetInvitesQuery{Status: "lost"})

	this.Equal(e.BadRequestCode.First, codeOf(this.T(), err))
}

func (this *InviteServiceTestSuite) TestListReportsTheStatusOfEachInvite() {
	now := time.Now()
	invites := []entity.Invite{
		{ID: "pending", ExpiresAt: now.Add(time.Hour)},
		{ID: "accepted", ExpiresAt: now.Add(time.Hour), AcceptedAt: &now},
		{ID: "revoked", ExpiresAt: now.Add(time.Hour), RevokedAt: &now},
		{ID: "expired", ExpiresAt: now.Add(-time.Hour)},
	}

	this.invites.On("FindForOrganization", organizationId, dto.InviteStatus(""), anything).Return(invites, nil)
	this.invites.On("FindForOrganizationCount", organizationId, dto.InviteStatus(""), anything).Return(int64(4), nil)

	page, err := this.service.List(organizationId, this.organization, &dto.GetInvitesQuery{})

	this.NoError(err)
	this.Equal(int64(4), page.Total)

	for _, invite := range page.Data {
		this.Equal(dto.InviteStatus(invite.ID), invite.Status)
	}
}

func (this *InviteServiceTestSuite) TestRevokeAnswers404ForAnInviteOfAnotherOrganization() {
	inviteId := "33333333-3333-3333-3333-333333333333"
	this.invites.On("FindOne", entity.Invite{ID: inviteId, OrganizationId: organizationId}, anything).Return(nil, nil)

	_, err := this.service.Revoke(organizationId, inviteId, this.organization)

	this.Equal(e.NotFoundErrorCode.First, codeOf(this.T(), err))
}

func (this *InviteServiceTestSuite) TestRevokeRefusesAnInviteThatIsNoLongerPending() {
	inviteId := "33333333-3333-3333-3333-333333333333"
	now := time.Now()
	this.invites.On("FindOne", anything, anything).Return(&entity.Invite{ID: inviteId, AcceptedAt: &now}, nil)
	this.invites.On("Revoke", inviteId, anything, anything).Return(int64(0), nil)

	_, err := this.service.Revoke(organizationId, inviteId, this.organization)

	this.Equal(e.ConflictErrorCode.First, codeOf(this.T(), err))
	this.True(strings.Contains(err.(*e.AppError).Detail, "accepted"))
	this.otps.AssertNotCalled(this.T(), "Invalidate", anything)
}

func (this *InviteServiceTestSuite) TestRevokeKillsTheLinkAtOnce() {
	inviteId := "33333333-3333-3333-3333-333333333333"
	this.invites.On("FindOne", anything, anything).Return(&entity.Invite{ID: inviteId, OtpId: otpId, ExpiresAt: time.Now().Add(time.Hour)}, nil)
	this.invites.On("Revoke", inviteId, anything, anything).Return(int64(1), nil)
	this.otps.On("Invalidate", otpId).Return()

	invite, err := this.service.Revoke(organizationId, inviteId, this.organization)

	this.NoError(err)
	this.Equal(dto.InviteStatusRevoked, invite.Status)
	this.otps.AssertCalled(this.T(), "Invalidate", otpId)
}

func TestInviteServiceSuite(t *testing.T) {
	suite.Run(t, new(InviteServiceTestSuite))
}
