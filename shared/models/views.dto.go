package models

import (
	entity "auth_service/infra/entities"
	"auth_service/shared/storage"
)

// The views are how an entity that owns an avatar leaves the service: the
// entity carries only the key, and avatar_url is derived here. Every relation
// that can hold one of them is shadowed by its view too, or the nested copy
// would leave without avatar_url. See steering/models-layer.md.

type UserView struct {
	entity.User
	AvatarUrl           *string           `json:"avatar_url"`
	CurrentOrganization *OrganizationView `json:"current_organization,omitempty"`
}

type OrganizationView struct {
	entity.Organization
	AvatarUrl    *string           `json:"avatar_url"`
	OwnerUser    *UserView         `json:"owner_user,omitempty"`
	Participants []ParticipantView `json:"participants,omitempty"`
}

type AppView struct {
	entity.App
	AvatarUrl    *string           `json:"avatar_url"`
	UsersPool    UsersPoolView     `json:"users_pool,omitempty"`
	OwnerUser    *UserView         `json:"owner_user,omitempty"`
	ParentApp    *AppView          `json:"parent_app,omitempty"`
	Organization *OrganizationView `json:"organization,omitempty"`
}

type ParticipantView struct {
	entity.Participant
	User *UserView `json:"user,omitempty"`
}

type UsersPoolView struct {
	entity.UsersPool
	OwnerUser    *UserView         `json:"owner_user,omitempty"`
	Organization *OrganizationView `json:"organization,omitempty"`
}

func NewUserView(user *entity.User, urls storage.IUrlResolver) *UserView {
	if user == nil {
		return nil
	}

	return &UserView{
		User:                *user,
		AvatarUrl:           urls.PublicUrl(user.AvatarKey),
		CurrentOrganization: NewOrganizationView(user.CurrentOrganization, urls),
	}
}

func NewUserViews(users []entity.User, urls storage.IUrlResolver) []UserView {
	views := make([]UserView, 0, len(users))

	for index := range users {
		views = append(views, *NewUserView(&users[index], urls))
	}

	return views
}

func NewOrganizationView(organization *entity.Organization, urls storage.IUrlResolver) *OrganizationView {
	if organization == nil {
		return nil
	}

	var participants []ParticipantView

	if organization.Participants != nil {
		participants = NewParticipantViews(organization.Participants, urls)
	}

	return &OrganizationView{
		Organization: *organization,
		AvatarUrl:    urls.PublicUrl(organization.AvatarKey),
		OwnerUser:    NewUserView(organization.OwnerUser, urls),
		Participants: participants,
	}
}

func NewOrganizationViews(organizations []entity.Organization, urls storage.IUrlResolver) []OrganizationView {
	views := make([]OrganizationView, 0, len(organizations))

	for index := range organizations {
		views = append(views, *NewOrganizationView(&organizations[index], urls))
	}

	return views
}

func NewAppView(app *entity.App, urls storage.IUrlResolver) *AppView {
	if app == nil {
		return nil
	}

	return &AppView{
		App:          *app,
		AvatarUrl:    urls.PublicUrl(app.AvatarKey),
		UsersPool:    *NewUsersPoolView(&app.UsersPool, urls),
		OwnerUser:    NewUserView(app.OwnerUser, urls),
		ParentApp:    NewAppView(app.ParentApp, urls),
		Organization: NewOrganizationView(app.Organization, urls),
	}
}

func NewAppViews(apps []entity.App, urls storage.IUrlResolver) []AppView {
	views := make([]AppView, 0, len(apps))

	for index := range apps {
		views = append(views, *NewAppView(&apps[index], urls))
	}

	return views
}

func NewParticipantView(participant *entity.Participant, urls storage.IUrlResolver) *ParticipantView {
	if participant == nil {
		return nil
	}

	return &ParticipantView{
		Participant: *participant,
		User:        NewUserView(participant.User, urls),
	}
}

func NewParticipantViews(participants []entity.Participant, urls storage.IUrlResolver) []ParticipantView {
	views := make([]ParticipantView, 0, len(participants))

	for index := range participants {
		views = append(views, *NewParticipantView(&participants[index], urls))
	}

	return views
}

func NewUsersPoolView(pool *entity.UsersPool, urls storage.IUrlResolver) *UsersPoolView {
	if pool == nil {
		return nil
	}

	return &UsersPoolView{
		UsersPool:    *pool,
		OwnerUser:    NewUserView(pool.OwnerUser, urls),
		Organization: NewOrganizationView(pool.Organization, urls),
	}
}

func NewUsersPoolViews(pools []entity.UsersPool, urls storage.IUrlResolver) []UsersPoolView {
	views := make([]UsersPoolView, 0, len(pools))

	for index := range pools {
		views = append(views, *NewUsersPoolView(&pools[index], urls))
	}

	return views
}
