// Package apidocs holds the pieces of the OpenAPI document that every controller
// shares: the tag names, the headers each guard requires and the RFC-7807 error
// payloads the error handler returns.
//
// A route is documented starting from one of the constructors below, picked by
// the guard chain protecting it, so the credentials a caller has to send are
// described the same way on every endpoint:
//
//	PublicRoute       - no guard
//	AppRoute          - AppGuard
//	AuthRoute         - AppGuard + AuthGuard
//	PermissionedRoute - AppGuard + AuthGuard + PermissionsGuard
//
// The endpoint specific parts (query, body and success response) are chained by
// the controller on top of the returned route.
package docs

import (
	e "auth_service/app/errors"

	"github.com/LuigiVanin/openapi-builder/openapi"
	"github.com/gofiber/fiber/v3"
)

const (
	TagAuth          = "Auth"
	TagOtp           = "OTP"
	TagApps          = "Apps"
	TagUsers         = "Users"
	TagUserPools     = "User Pools"
	TagOrganizations = "Organizations"
	TagHealth        = "Health"
)

func PublicRoute(
	builder *openapi.Builder,
	method string,
	path string,
	options ...openapi.Options,
) *openapi.RouteBuilder {
	return builder.
		Route(method, path, options...).
		AddResponse(fiber.StatusInternalServerError, e.ProblemDetail{}, openapi.Options{
			Description: "Unexpected error",
		})
}

func AppRoute(
	builder *openapi.Builder,
	method string,
	path string,
	options ...openapi.Options,
) *openapi.RouteBuilder {
	return PublicRoute(builder, method, path, options...).
		AddHeaderParam("X-Public-Key", openapi.String, openapi.Options{
			Required:    true,
			Description: "Ciphered app id - the `as_` prefixed public key of the application",
		}).
		AddHeaderParam("X-Pool-Key", openapi.String, openapi.Options{
			Required:    true,
			Description: "Ciphered users pool id - has to be the pool the application belongs to",
		}).
		AddHeaderParam("X-Secret-Key", openapi.String, openapi.Options{
			Description: "Application secret key - only required when the application is private",
		}).
		AddResponse(fiber.StatusUnauthorized, e.ProblemDetail{}, openapi.Options{
			Description: "Missing, malformed or mismatching application credentials",
		}).
		AddResponse(fiber.StatusNotFound, e.ProblemDetail{}, openapi.Options{
			Description: "No application matches the given X-Public-Key",
		})
}

func AuthRoute(
	builder *openapi.Builder,
	method string,
	path string,
	options ...openapi.Options,
) *openapi.RouteBuilder {
	return AppRoute(builder, method, path, options...).
		AddHeaderParam("Authorization", openapi.String, openapi.Options{
			Required:    true,
			Description: "Access token returned by /auth/login, /auth/register or /auth/refresh",
		})
}

// AppGuard + AuthGuard + OrganizationGuard + PermissionsGuard.
func PermissionedRoute(
	builder *openapi.Builder,
	method string,
	path string,
	options ...openapi.Options,
) *openapi.RouteBuilder {
	return AuthRoute(builder, method, path, options...).
		AddResponse(fiber.StatusForbidden, e.ProblemDetail{}, openapi.Options{
			Description: "`PERMISSION_DENIED` when the permissions of the user in its current organization do not grant access to this route, `NOT_A_PARTICIPANT` when the user does not participate in the organization it is scoped to",
		})
}

func Validated(route *openapi.RouteBuilder) *openapi.RouteBuilder {
	return route.
		AddResponse(fiber.StatusBadRequest, e.ProblemDetail{}, openapi.Options{
			Description: "Request body could not be parsed",
		}).
		AddResponse(fiber.StatusUnprocessableEntity, e.ProblemDetail{}, openapi.Options{
			Description: "Request body failed validation",
		})
}
