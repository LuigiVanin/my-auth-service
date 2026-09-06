# Auth Service

A robust y authentication and authorization service built with Go, designed to handle user management, multi-tenancy, and fine-grained access control.

## Features

- **User Authentication** - Secure login with email/password and OTP verification
- **JWT Token Management** - Access and refresh token handling with configurable expiration
- **Multi-tenancy** - User pools and application-based isolation
- **Role-Based Access Control (RBAC)** - Fine-grained permissions and profile-based authorization
- **Session Management** - Secure session tracking and management
- **Email Verification** - OTP-based email verification using Resend
- **Application Management** - Create and manage multiple applications with different configurations
- **Hot Reloading** - Development mode with Air for rapid iteration
- **Structured Logging** - Production-ready logging with Zap
- **Dependency Injection** - Clean architecture using Uber FX

## Tech Stack

- **Language**: Go 1.25.3
- **Web Framework**: [Fiber v3](https://github.com/gofiber/fiber)
- **Database**: PostgreSQL with [GORM](https://gorm.io/)
- **Dependency Injection**: [Uber FX](https://github.com/uber-go/fx)
- **Logging**: [Zap](https://github.com/uber-go/zap)
- **JWT**: [golang-jwt/jwt](https://github.com/golang-jwt/jwt)
- **Email**: [Resend API](https://resend.com/)
- **Hot Reload**: [Air](https://github.com/cosmtrek/air)

## Getting Started

### Prerequisites

- Go 1.25.3 or higher
- PostgreSQL 12 or higher
- [Air](https://github.com/cosmtrek/air) (for development with hot reload)

### Installation

1. Clone the repository:
```bash
git clone https://github.com/LuigiVanin/my-auth-service.git
cd auth-service
```

2. Install dependencies:
```bash
go mod download
```

3. Set up environment variables:

Create a `.env.development` file or `.env.development.yaml` file based on `.env.example` or `.env.example.yaml`:

**Option 1: .env file**
```env
SERVER_PORT=3000
APP_NAME=auth-service
APP_ENCRYPTION_KEY=your-encryption-key
APP_ENV=development

DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your-password
DB_NAME=auth_db
DB_SSLMODE=disable

RESEND_API_KEY=your-resend-api-key
```

**Option 2: YAML configuration**
```yaml
app:
  name: auth-service
  encryption_key: your-encryption-key
  env: development

server:
  port: "3000"

database:
  host: localhost
  port: "5432"
  user: postgres
  password: your-password
  name: auth_db
  sslmode: disable

email:
  resend_api_key: your-resend-api-key
```

4. Initialize the database:
```bash
make migrate up
make seed init
```

This command will:
- Create an ADMIN application
- Create an ADMIN user
- Create the default profiles (ADMIN, MANAGER, CONSUMER)

### Running the Application

**Development mode (with hot reload):**
```bash
make dev
```

**Production mode:**
```bash
make run
```

**Build executable:**
```bash
make build
```

The compiled binary will be available at `./build/auth_service`.

## API Endpoints

### Authentication

- `POST /auth/register` - Register a new user
- `POST /auth/login` - Login with credentials
- `POST /auth/authorize` - Authorize and get access token
- `POST /auth/refresh` - Refresh access token

- `PUT /auth/forgot_passwords` - Update password

### OTP Management

- `POST /otp/generate` - Generate OTP for email verification
- `PUT /otp/validate` - Validate OTP code

### Core Resources

- `GET /core/users?app_id=:uuid` - List the users of the pool an application belongs to, within the current organization (requires permissions)
- `GET /core/users?pool_id=:uuid` - List the users of a pool of the current organization (requires permissions)
- `GET /core/users/me` - Get the authenticated user - shortcut for `/core/users/:id` on its own id
- `GET /core/users/:id` - Get user details (requires permissions)
- `GET /core/user/:user_id/apps` - Get user's applications (requires permissions) [TODO]

- `POST /core/apps` - Create a new application
- `GET /core/apps` - List the applications of the current organization (requires permissions)
- `GET /core/apps?pool_id=:uuid` - Narrow that listing to one pool of the current organization (requires permissions)
- `GET /core/apps/:uuid` - Get application details, without `secret_key` (requires permissions)
- `PUT /core/apps/:uuid` - Update application (requires permissions) [TODO]

- `POST /core/users_pool`
- `GET /core/users_pool` - List the users pools of the current organization (requires permissions)
- `GET /core/users_pool/:uuid` - Get users pool details (requires permissions)
- `PUT /core/users_pool/:id` [TODO]

## Database Management

### Migrations

The project uses GORM's AutoMigrate feature. To run migrations:

```bash
make migrate up
```

To rollback migrations:

```bash
make migrate down
```

### Database Seeds

To set up the database with an initial admin application and user:

```bash
make seed init
```

To wipe every seeded table so `init` can run from scratch again (the schema is kept,
so there is no need to re-run the migrations):

```bash
make seed reset
```

Both accept an optional environment: `make seed init prod`.

## Security Features

### Guards (Middleware)

- **AppGuard** - Validates application credentials (public/secret keys)
- **AuthGuard** - Validates JWT authentication tokens
- **OtpGuard** - Validates OTP verification status
- **PermissionsGuard** - Validates role-based permissions

### Password Security

- Passwords are hashed using argon
- Support for password complexity validation
- Secure password reset flow with OTP

### Token Security

- JWT-based authentication
- Separate access and refresh tokens
- Configurable token expiration times
- Token encryption support

## Configuration

The service supports three configuration methods (in order of precedence):

1. **YAML Configuration** - `.env.{environment}.yaml`
2. **Environment File** - `.env.{environment}`
3. **System Environment Variables**

Environment can be specified when running the application:

```bash
go run ./cmd/main.go production
# or
make dev production
```

## Development Tools

### Hot Reloading

Air is configured for hot reloading during development:

```bash
make dev
```

Configuration is available in [.air.toml](.air.toml).

### Git Hooks

Husky is configured for commit message validation and pre-commit checks.

### Cipher Helper

Generate encrypted values for configuration:

```bash
make cipher <value>
```

## Testing

Run tests:

```bash
go test ./...
```

Run specific module tests:

```bash
go test ./tests/modules/login/...
```

## Architecture Patterns

### Dependency Injection

The application uses Uber FX for dependency injection, providing clean separation of concerns and testability.

### Module Pattern

Each feature is organized as a self-contained module with:
- **Controller** - HTTP handlers
- **Service** - Business logic
- **Repository** - Data access layer
- **Module** - FX module definition

### Clean Architecture

- **Entities** - Core business entities (infra/entities)
- **Use Cases** - Application business rules (modules/services)
- **Interface Adapters** - Controllers and repositories

## Multi-tenancy

The service implements multi-tenancy through:

- **User Pools** - Isolated user groups
- **Applications** - Different apps can have separate user pools
- **Parent-Child Apps** - Hierarchical application relationships

## License

This project is licensed under the MIT License - see the [license.md](license.md) file for details.

## Support

For issues and questions, please open an issue at:
https://github.com/LuigiVanin/my-auth-service/issues
