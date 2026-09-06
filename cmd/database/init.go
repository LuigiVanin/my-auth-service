package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	cipher "auth_service/app/modules/utils/cipher"
	hash "auth_service/app/modules/utils/hash/services"
	"auth_service/infra/config"
	entity "auth_service/infra/entities"
	"auth_service/shared/constants"
	"auth_service/shared/permissions"
	"auth_service/shared/utils"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

const (
	mainPoolName   = "main_app_pool"
	mainAppName    = "main_app"
	adminUserName  = "Admin User"
	adminUserEmail = "admin@example.com"
	adminOrgName   = "admin_organization"

	credentialsFile = "credentials.txt"
)

// A grant is the authoring format: `as::{feature}::{subfeature?}::{ACTION}`,
// translated into route patterns by shared/permissions. Writing api by hand is
// the administration escape hatch, and it is what ADMIN uses on purpose - see
// below. Every grant here is checked against the catalog by validateSeededGrants.
var (
	// The "*" of api matches any registered route, catalogued or not, and the guard
	// reads that key first. The grant rides along only so the admin reports a grants
	// list instead of an empty one - it does narrow IsSubsetOf, which prefers an exact
	// key over "*". See docs/steering/modules/profiles.md.
	adminPermissions = json.RawMessage(`{"api": {"*": {"methods": ["*"]}}, "grants": ["as::*::*"]}`)

	// Carries the whole auth and otp set because it is the ceiling of
	// LOGIN_PROFILE, and IsSubsetOf refuses a child that names a route the parent
	// does not.
	managerPermissions = json.RawMessage(`{"grants": [
		"as::apps::CREATE", "as::apps::READ", "as::apps::UPDATE",
		"as::users::READ", "as::users::me::READ",
		"as::users_pool::CREATE", "as::users_pool::READ",
		"as::organizations::CREATE", "as::organizations::READ",
		"as::organizations::switch::UPDATE", "as::organizations::participants::READ",
		"as::organizations::participants::UPDATE",
		"as::profiles::CREATE", "as::profiles::READ", "as::profiles::UPDATE",
		"as::grants::READ",
		"as::login::CREATE", "as::register::CREATE", "as::authorize::CREATE",
		"as::refresh::CREATE", "as::forgot_password::UPDATE", "as::otp::CREATE"
	]}`)

	// NOTE: nothing enforces /auth yet, so the login and register grants are
	// expressive rather than effective. They are here because the profile a pool
	// defaults to is where "may this pool sign users up" belongs.
	loginPermissions = json.RawMessage(`{"grants": [
		"as::login::CREATE", "as::register::CREATE",
		"as::organizations::READ", "as::organizations::switch::UPDATE"
	]}`)

	// Nothing assigns it yet; seeded for the invite flow.
	memberPermissions = json.RawMessage(`{"grants": [
		"as::organizations::READ", "as::organizations::participants::READ"
	]}`)

	emptyJson = json.RawMessage(`{}`)
)

// initSeed holds the values produced by the seed that are needed once the
// transaction is committed (credentials output).
type initSeed struct {
	usersPoolId    string
	poolPublicKey  string
	appId          string
	appPublicKey   string
	appSecretKey   string
	organizationId string
	password       string
}

// Fails the seed loudly when a literal above names a grant the catalog does not
// know. Nothing else checks these strings: a key renamed or removed in
// shared/permissions turns them into rows that grant nothing, silently.
func validateSeededGrants() error {
	documents := []json.RawMessage{
		adminPermissions,
		managerPermissions,
		loginPermissions,
		memberPermissions,
	}

	for _, document := range documents {
		parsed, err := permissions.Parse(document)

		if err != nil {
			return err
		}

		if err := permissions.ValidateGrants(parsed.Grants); err != nil {
			return err
		}
	}

	return nil
}

func run(db *gorm.DB, cfg *config.Config) error {
	cipherService := cipher.NewCipherService(cfg)
	hashService := hash.NewHashService()

	if err := validateSeededGrants(); err != nil {
		return fmt.Errorf("seeded profiles carry an unknown grant: %w", err)
	}

	fmt.Println()
	utils.PrintHeader("Initializing Database Seed")

	seed := initSeed{}

	err := db.Transaction(func(tx *gorm.DB) error {
		// The order is dictated by the NOT NULL columns and has no slack in it:
		// profiles before the pool, the organization before the user, and the user
		// before the owner of the organization can be stamped.

		// 1. Create Profiles
		adminProfile, err := upsertProfile(tx, entity.Profile{
			Key:         constants.ProfileAdmin,
			Name:        "Admin Profile",
			Permissions: adminPermissions,
			Metadata:    emptyJson,
		})
		if err != nil {
			return fmt.Errorf("failed to insert or retrieve admin profile: %w", err)
		}
		utils.PrintSuccess("Admin Profile created/retrieved")

		managerProfile, err := upsertProfile(tx, entity.Profile{
			Key:         constants.ProfileManager,
			Name:        "Manager Profile",
			Permissions: managerPermissions,
			Metadata:    emptyJson,
		})
		if err != nil {
			return fmt.Errorf("failed to insert or retrieve manager profile: %w", err)
		}
		utils.PrintSuccess("Manager Profile created/retrieved")

		if _, err := upsertProfile(tx, entity.Profile{
			Key:         constants.ProfileLogin,
			Name:        "Login Profile",
			Permissions: loginPermissions,
			Metadata:    emptyJson,
		}); err != nil {
			return fmt.Errorf("failed to insert or retrieve login profile: %w", err)
		}
		utils.PrintSuccess("Login Profile created/retrieved")

		if _, err := upsertProfile(tx, entity.Profile{
			Key:         constants.ProfileMember,
			Name:        "Member Participant Profile",
			Permissions: memberPermissions,
			Metadata:    emptyJson,
		}); err != nil {
			return fmt.Errorf("failed to insert or retrieve member profile: %w", err)
		}
		utils.PrintSuccess("Member Participant Profile created/retrieved")

		// 2. Create Users Pool
		// The id is generated upfront so its ciphered version (the public key)
		// can be persisted on the same insert, mirroring what the user pool
		// repository does at runtime.
		usersPoolId := uuid.New().String()

		poolPublicKey, err := cipherService.EncryptUuidIntoToken(usersPoolId)
		if err != nil {
			return fmt.Errorf("failed to encrypt users pool id: %w", err)
		}

		usersPool := entity.UsersPool{
			ID:        usersPoolId,
			Name:      mainPoolName,
			PublicKey: poolPublicKey,

			// Every other pool is created through the API and starts on LOGIN_PROFILE.
			DefaultProfileId: managerProfile.ID,
		}

		if err := tx.Create(&usersPool).Error; err != nil {
			return fmt.Errorf("failed to insert users pool: %w", err)
		}
		utils.PrintSuccess("Users Pool created")

		// 3. Create App
		appId := uuid.New().String()

		appPublicKey, err := cipherService.EncryptUuidIntoToken(appId)
		if err != nil {
			return fmt.Errorf("failed to encrypt app id: %w", err)
		}

		app := entity.App{
			ID:                  appId,
			UsersPoolId:         usersPool.ID,
			Name:                mainAppName,
			Role:                "ADMIN",
			PublicKey:           appPublicKey,
			SecretKey:           uuid.New().String(),
			LoginTypes:          pq.StringArray{"WITH_PASSWORD"},
			TokenType:           "JWT",
			TokenExpirationTime: 3600,
			Metadata:            emptyJson,

			Enabled2FA: false,
		}

		if err := tx.Create(&app).Error; err != nil {
			return fmt.Errorf("failed to insert app: %w", err)
		}
		utils.PrintSuccess("App created")

		// 4. Create the organization of the platform admin
		//
		// NOTE: it lives inside the very pool it goes on to own, and this is the ONLY
		// place where that is allowed - everywhere else the owning organization sits
		// in the parent pool, which is what keeps a user from seeing the pool it
		// belongs to. See docs/specs/2026-08-23-organizations.md.
		adminOrganization := entity.Organization{
			UsersPoolId: usersPool.ID,
			ProfileId:   adminProfile.ID,
			Name:        adminOrgName,
			Description: "Organization of the platform administrator",
			Metadata:    emptyJson,
		}

		if err := tx.Create(&adminOrganization).Error; err != nil {
			return fmt.Errorf("failed to insert admin organization: %w", err)
		}
		utils.PrintSuccess("Admin Organization created")

		// 5. Create Admin User
		// password := uuid.New().String()
		password := "Senha123!"

		adminHashedPassword, err := hashService.HashText(password, uuid.New().String())
		if err != nil {
			return fmt.Errorf("failed to hash admin password: %w", err)
		}

		adminUser := entity.User{
			Name:                  adminUserName,
			Email:                 adminUserEmail,
			UsersPoolId:           usersPool.ID,
			CurrentOrganizationId: adminOrganization.ID,
			PasswordHash:          adminHashedPassword,
			Metadata:              emptyJson,
		}

		if err := tx.Create(&adminUser).Error; err != nil {
			return fmt.Errorf("failed to insert admin user: %w", err)
		}
		utils.PrintSuccess("Admin User created")

		// 6. Close the cycle: the organization now has an owner
		if err := tx.Model(&entity.Organization{}).
			Where("id = ?", adminOrganization.ID).
			Update("owner_user_id", adminUser.ID).Error; err != nil {
			return fmt.Errorf("failed to set the admin organization owner: %w", err)
		}
		utils.PrintSuccess("Admin Organization owner set")

		// 7. The owner participates on the ceiling of its own organization.
		adminParticipant := entity.Participant{
			OrganizationId: adminOrganization.ID,
			UserId:         adminUser.ID,
			ProfileId:      adminProfile.ID,
			Metadata:       emptyJson,
		}

		if err := tx.Create(&adminParticipant).Error; err != nil {
			return fmt.Errorf("failed to insert admin participant: %w", err)
		}
		utils.PrintSuccess("Admin Participant created")

		// 8. The column every listing filters by.
		if err := tx.Model(&entity.UsersPool{}).
			Where("id = ?", usersPool.ID).
			Update("organization_id", adminOrganization.ID).Error; err != nil {
			return fmt.Errorf("failed to set the users pool organization: %w", err)
		}

		if err := tx.Model(&entity.App{}).
			Where("id = ?", app.ID).
			Update("organization_id", adminOrganization.ID).Error; err != nil {
			return fmt.Errorf("failed to set the app organization: %w", err)
		}
		utils.PrintSuccess("Users Pool and App bound to the Admin Organization")

		// 9. ADMIN is born global because its organization does not exist yet when
		// the profiles are written, and is scoped here. The other three stay global:
		// main_app_pool defaults to MANAGER_PROFILE, so no existing path starts
		// depending on a profile it cannot see.
		if err := tx.Model(&entity.Profile{}).
			Where("id = ?", adminProfile.ID).
			Update("organization_id", adminOrganization.ID).Error; err != nil {
			return fmt.Errorf("failed to scope the admin profile to the admin organization: %w", err)
		}
		utils.PrintSuccess("Admin Profile scoped to the Admin Organization")

		seed = initSeed{
			usersPoolId:    usersPool.ID,
			poolPublicKey:  poolPublicKey,
			appId:          app.ID,
			appPublicKey:   appPublicKey,
			appSecretKey:   app.SecretKey,
			organizationId: adminOrganization.ID,
			password:       password,
		}

		return nil
	})

	if err != nil {
		return err
	}

	writeCredentialsFile(seed)
	printCredentials(seed)

	return nil
}

// upsertProfile mirrors `ON CONFLICT (key) DO UPDATE SET key=EXCLUDED.key RETURNING id`:
// the existing profile is returned untouched when the key is already taken.
func upsertProfile(tx *gorm.DB, profile entity.Profile) (*entity.Profile, error) {
	var existing entity.Profile

	err := tx.Where("key = ?", profile.Key).First(&existing).Error

	if err == nil {
		return &existing, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if err := tx.Create(&profile).Error; err != nil {
		return nil, err
	}

	return &profile, nil
}

func writeCredentialsFile(seed initSeed) {
	newCredentials := fmt.Sprintf(`
Initialization Complete & Credentials
=====================================

Admin Credentials
-----------------
Username            : %s
Email               : %s
Password            : %s

System IDs (Ciphered)
---------------------
Users Pool ID       : %s
Encrypted Pool ID   : %s

App ID              : %s
Encrypted App ID    : %s
App Secret Key      : %s

Organization ID     : %s

Use the Encrypted IDs in your request headers (X-Pool-Key, X-Public-Key)
`, adminUserName, adminUserEmail, seed.password, seed.usersPoolId, seed.poolPublicKey, seed.appId, seed.appPublicKey, seed.appSecretKey, seed.organizationId)

	// Get System Info
	hostname, _ := os.Hostname()
	addrs, _ := net.InterfaceAddrs()

	var ip string
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip = ipnet.IP.String()
				break
			}
		}
	}

	timestamp := time.Now().Format(time.RFC1123)

	separator := fmt.Sprintf("\n\n------------------------------------------------------------\n"+
		"Generated on: %s\n"+
		"Device: %s\n"+
		"IP: %s\n"+
		"------------------------------------------------------------\n\n\n", timestamp, hostname, ip)

	// Check if file exists and read content
	var finalContent []byte

	existingContent, err := os.ReadFile(credentialsFile)
	if err == nil && len(existingContent) > 0 {
		// File exists, prepend new content with separator
		finalContent = append([]byte(newCredentials), []byte(separator)...)
		finalContent = append(finalContent, existingContent...)
	} else {
		// New file
		finalContent = []byte(newCredentials)
	}

	if err := os.WriteFile(credentialsFile, finalContent, 0644); err != nil {
		utils.PrintError(fmt.Sprintf("Failed to write %s", credentialsFile), err)
		return
	}

	utils.PrintSuccess(fmt.Sprintf("Credentials saved to %s", credentialsFile))
}

func printCredentials(seed initSeed) {
	fmt.Println()
	utils.PrintHeader("Initialization Complete & Credentials")

	utils.PrintSection("Admin Credentials")
	utils.PrintKeyValue("Username", adminUserName)
	utils.PrintKeyValue("Email", adminUserEmail)
	utils.PrintKeyValue("Password", seed.password)

	utils.PrintSection("System IDs (Ciphered)")
	utils.PrintKeyValue("Users Pool ID", seed.usersPoolId)
	utils.PrintKeyValue("Encrypted Pool ID", seed.poolPublicKey)
	fmt.Println()
	utils.PrintKeyValue("App ID", seed.appId)
	utils.PrintKeyValue("Encrypted App ID", seed.appPublicKey)
	utils.PrintKeyValue("App Secret Key", seed.appSecretKey)

	fmt.Println()
	utils.PrintSection("Current Organization")
	utils.PrintKeyValue("Organization ID", seed.organizationId)

	fmt.Println()
	utils.PrintHint("Use the Encrypted IDs in your request headers (X-Pool-Key, X-Public-Key)")
	fmt.Println()
}
