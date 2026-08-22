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

	credentialsFile = "credentials.txt"
)

var (
	adminPermissions = json.RawMessage(`{ "api": { "*": { "methods": ["*"] } } }`)

	managerPermissions = json.RawMessage(`{"api": {"/core/apps": {"methods": ["POST", "GET"]}, "/core/apps/:id": {"methods": ["GET", "PUT", "DELETE"]}, "/core/users_pool": {"methods": ["GET", "POST", "PUT"]}, "/core/apps/:id/users": {"query": {"name": "^.+$", "skip": "^[0-9]+$", "email": "^.+$", "limit": "^[0-9]+$"}, "methods": ["GET", "POST"]}}}`)

	emptyJson = json.RawMessage(`{}`)
)

// initSeed holds the values produced by the seed that are needed once the
// transaction is committed (credentials output).
type initSeed struct {
	usersPoolId   string
	poolPublicKey string
	appId         string
	appPublicKey  string
	appSecretKey  string
	password      string
}

func run(db *gorm.DB, cfg *config.Config) error {
	cipherService := cipher.NewCipherService(cfg)
	hashService := hash.NewHashService()

	fmt.Println()
	utils.PrintHeader("Initializing Database Seed")

	seed := initSeed{}

	err := db.Transaction(func(tx *gorm.DB) error {
		// 1. Create Users Pool
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
		}

		if err := tx.Create(&usersPool).Error; err != nil {
			return fmt.Errorf("failed to insert users pool: %w", err)
		}
		utils.PrintSuccess("Users Pool created")

		// 2. Create App
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

		// 3. Create Profiles
		adminProfile, err := upsertProfile(tx, entity.Profile{
			Key:         "ADMIN",
			Name:        "Admin Profile",
			Permissions: adminPermissions,
			Metadata:    emptyJson,
		})
		if err != nil {
			return fmt.Errorf("failed to insert or retrieve admin profile: %w", err)
		}
		utils.PrintSuccess("Admin Profile created/retrieved")

		managerProfile, err := upsertProfile(tx, entity.Profile{
			Key:             "MANAGER",
			Name:            "Manager Profile",
			ParentProfileId: &adminProfile.ID,
			Permissions:     managerPermissions,
			Metadata:        emptyJson,
		})
		if err != nil {
			return fmt.Errorf("failed to insert or retrieve manager profile: %w", err)
		}
		utils.PrintSuccess("Manager Profile created/retrieved")

		consumerProfile, err := upsertProfile(tx, entity.Profile{
			Key:             "CONSUMER",
			Name:            "Consumer Profile",
			ParentProfileId: &managerProfile.ID,
			Permissions:     emptyJson,
			Metadata:        emptyJson,
		})
		if err != nil {
			return fmt.Errorf("failed to insert or retrieve consumer profile: %w", err)
		}
		utils.PrintSuccess("Consumer Profile created/retrieved")

		// 4. Create App Role Profiles
		// NOTE: Here I will create the relation between the app types and the user roles
		//       Where I will define what kind of users each app will have.
		// Example:
		//  - The Admin app will have admin users and managers users - those users will have the profile associated with the
		//    app via database APP_ROLE -> PROFILES.
		//  - The User app will have consumer users - those users will have the profile associated with the
		//    app via database APP_ROLE -> PROFILES.
		if err := ensureAppRoleProfile(tx, adminProfile.ID, "ADMIN", 999, json.RawMessage(`{ "register": false }`)); err != nil {
			return fmt.Errorf("failed to insert admin app role profile: %w", err)
		}
		utils.PrintSuccess("Admin App Role Profile created")

		if err := ensureAppRoleProfile(tx, managerProfile.ID, "ADMIN", 0, json.RawMessage(`{ "register": true }`)); err != nil {
			return fmt.Errorf("failed to insert manager app role profile: %w", err)
		}
		utils.PrintSuccess("Manager App Role Profile created")

		if err := ensureAppRoleProfile(tx, consumerProfile.ID, "USER", 0, json.RawMessage(`{ "register": true }`)); err != nil {
			return fmt.Errorf("failed to insert consumer app role profile: %w", err)
		}
		utils.PrintSuccess("Consumer App Role Profile created")

		// 5. Create Admin User
		// password := uuid.New().String()
		password := "Senha123!"

		adminHashedPassword, err := hashService.HashText(password, uuid.New().String())
		if err != nil {
			return fmt.Errorf("failed to hash admin password: %w", err)
		}

		adminUser := entity.User{
			Name:         adminUserName,
			Email:        adminUserEmail,
			UsersPoolId:  usersPool.ID,
			ProfileId:    adminProfile.ID,
			PasswordHash: adminHashedPassword,
			Metadata:     emptyJson,
		}

		if err := tx.Create(&adminUser).Error; err != nil {
			return fmt.Errorf("failed to insert admin user: %w", err)
		}
		utils.PrintSuccess("Admin User created")

		seed = initSeed{
			usersPoolId:   usersPool.ID,
			poolPublicKey: poolPublicKey,
			appId:         app.ID,
			appPublicKey:  appPublicKey,
			appSecretKey:  app.SecretKey,
			password:      password,
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

// ensureAppRoleProfile creates the profile/role relation only when it does not
// exist yet. The insert goes through a map because GORM drops zero valued struct
// fields that declare a `default` tag - `priority = 0` would silently become 999.
func ensureAppRoleProfile(tx *gorm.DB, profileId string, role string, priority int, permission json.RawMessage) error {
	var count int64

	err := tx.Model(&entity.AppRoleProfile{}).
		Where("profile_id = ? AND role = ?", profileId, role).
		Count(&count).Error

	if err != nil {
		return err
	}

	if count > 0 {
		return nil
	}

	return tx.Model(&entity.AppRoleProfile{}).Create(map[string]any{
		"profile_id": profileId,
		"role":       role,
		"priority":   priority,
		"permission": string(permission),
		"relation":   string(emptyJson),
		"metadata":   string(emptyJson),
	}).Error
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

Use the Encrypted IDs in your request headers (X-Pool-Key, X-Public-Key)
`, adminUserName, adminUserEmail, seed.password, seed.usersPoolId, seed.poolPublicKey, seed.appId, seed.appPublicKey, seed.appSecretKey)

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
	utils.PrintHint("Use the Encrypted IDs in your request headers (X-Pool-Key, X-Public-Key)")
	fmt.Println()
}
