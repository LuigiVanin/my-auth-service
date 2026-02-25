package main

import (
	"fmt"
	"net"
	"os"
	"time"

	cipher "auth_service/app/modules/utils/cipher"
	hash "auth_service/app/modules/utils/hash/services"
	"auth_service/common/constants"
	"auth_service/infra/config"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	cfg := config.CreateEnvConfig()
	url := cfg.FormatDatabaseUrl()

	db, err := sqlx.Connect("postgres", url)
	if err != nil {
		printError("failed to connect to database", err)
		return
	}
	defer db.Close()

	// Start Transaction
	tx, err := db.Beginx()
	if err != nil {
		printError("failed to begin transaction", err)
		return
	}
	// Defer rollback in case of panic or error (if not committed)
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p) // re-panic after rollback
		} else if err != nil {
			_ = tx.Rollback()
		}
	}()

	cipherService := cipher.NewCipherService(cfg)

	fmt.Println()
	printHeader("Initializing Database Seed")

	// 1. Create Users Pool
	var usersPoolId string
	err = tx.QueryRow(`
		INSERT INTO users_pool (name) VALUES ('main_app_pool') RETURNING id
	`).Scan(&usersPoolId)
	if err != nil {
		printError("failed to insert users pool", err)
		return
	}
	printSuccess("Users Pool created")

	// 2. Create App
	var appId string
	err = tx.QueryRow(`
		INSERT INTO apps (
			users_pool_id, name, role, public_key, secret_key, 
			login_types, token_type, token_expiration_time
		) VALUES (
			$1, 'main_app', 'ADMIN', uuid_generate_v4()::text, uuid_generate_v4()::text,
			ARRAY['WITH_PASSWORD']::AUTH_METHOD[], 'JWT', 3600
		) RETURNING id
	`, usersPoolId).Scan(&appId)
	if err != nil {
		printError("failed to insert app", err)
		return
	}
	printSuccess("App created")

	// 3. Create Profiles
	var adminProfileId string
	var managerProfileId string
	var consumerProfileId string

	// Admin Profile
	err = tx.QueryRow(`
		INSERT INTO profiles (key, name, parent_profile_id, permissions)
		VALUES ('ADMIN', 'Admin Profile', NULL, '{ "api": { "*": { "methods": ["*"] } } }')
		ON CONFLICT (key) DO UPDATE SET key=EXCLUDED.key
		RETURNING id
	`).Scan(&adminProfileId)
	if err != nil {
		printError("failed to insert or retrieve admin profile", err)
		return
	}
	printSuccess("Admin Profile created/retrieved")

	managerPermissions := `{ "api": { "/core/apps": { "methods": ["POST", "GET"] }, "/core/apps/:id": { "methods": ["GET", "PUT", "DELETE"] }, "/core/apps/:id/users": { "methods": ["GET", "POST"], "query": { "skip": "^[0-9]+$", "limit": "^[0-9]+$", "name": "^.+$", "email": "^.+$" } } } }`

	// Manager Profile
	err = tx.QueryRow(`
		INSERT INTO profiles (key, name, parent_profile_id, permissions)
		VALUES ('MANAGER', 'Manager Profile', $1, $2)
		ON CONFLICT (key) DO UPDATE SET key=EXCLUDED.key
		RETURNING id
	`, adminProfileId, managerPermissions).Scan(&managerProfileId)
	if err != nil {
		printError("failed to insert or retrieve manager profile", err)
		return
	}
	printSuccess("Manager Profile created/retrieved")

	// Consumer Profile
	err = tx.QueryRow(`
		INSERT INTO profiles (key, name, parent_profile_id)
		VALUES ('CONSUMER', 'Consumer Profile', $1)
		ON CONFLICT (key) DO UPDATE SET key=EXCLUDED.key
		RETURNING id
	`, managerProfileId).Scan(&consumerProfileId)
	if err != nil {
		printError("failed to insert or retrieve consumer profile", err)
		return
	}
	printSuccess("Consumer Profile created/retrieved")

	// 4. Create App Role Profiles
	// NOTE: Here I will create the relation between the app types and the user roles
	//       Where I will define what kind of users each app will have.
	// Example:
	//  - The Admin app will have admin users and managers users - those users will have the profile associated with the
	//    app via database APP_ROLE -> PROFILES.
	//  - The User app will have consumer users - those users will have the profile associated with the
	//    app via database APP_ROLE -> PROFILES.
	_, err = tx.Exec(`
		INSERT INTO app_role_profiles (profile_id, role, priority, permission, relation, metadata)
		SELECT $1, 'ADMIN', 999, '{ "register": false }', '{}', '{}'
		WHERE NOT EXISTS (SELECT 1 FROM app_role_profiles WHERE profile_id = $1 AND role = 'ADMIN')
	`, adminProfileId)

	if err != nil {
		printError("failed to insert admin app role profile", err)
		return
	}
	printSuccess("Admin App Role Profile created")

	_, err = tx.Exec(`
		INSERT INTO app_role_profiles (profile_id, role, priority, permission, relation, metadata)
		SELECT $1, 'ADMIN', 0, '{ "register": true }', '{}', '{}'
		WHERE NOT EXISTS (SELECT 1 FROM app_role_profiles WHERE profile_id = $1 AND role = 'ADMIN')
	`, managerProfileId)
	if err != nil {
		printError("failed to insert manager app role profile", err)
		return
	}
	printSuccess("Manager App Role Profile created")

	_, err = tx.Exec(`
		INSERT INTO app_role_profiles (profile_id, role, priority, permission, relation, metadata)
		SELECT $1, 'USER', 0, '{ "register": true }', '{}', '{}'
		WHERE NOT EXISTS (SELECT 1 FROM app_role_profiles WHERE profile_id = $1 AND role = 'USER')
	`, consumerProfileId)
	if err != nil {
		printError("failed to insert consumer app role profile", err)
		return
	}
	printSuccess("Consumer App Role Profile created")

	// 5. Create Admin User

	hashService := hash.NewHashService()

	adminPassword, err := hashService.HashText(uuid.New().String(), uuid.New().String())
	var adminUserId int // ID is SERIAL
	var adminUserUuid string
	err = tx.QueryRow(`
		INSERT INTO users (name, email, users_pool_id, password_hash, profile_id)
		VALUES ('Admin User', 'admin@example.com', $1, $2, $3)
		RETURNING id, uuid
	`, usersPoolId, adminPassword, adminProfileId).Scan(&adminUserId, &adminUserUuid)
	if err != nil {
		printError("failed to insert admin user", err)
		return
	}
	printSuccess("Admin User created")

	// Commit Transaction
	err = tx.Commit()
	if err != nil {
		printError("failed to commit transaction", err)
		return
	}

	// Encrypt IDs
	encryptedPoolId, err := cipherService.EncryptUuidIntoToken(usersPoolId)
	if err != nil {
		printError("failed to encrypt users pool id", err)
		return
	}
	encryptedAppId, err := cipherService.EncryptUuidIntoToken(appId)
	if err != nil {
		printError("failed to encrypt app id", err)
		return
	}

	// Prepare Output Content
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

	newCredentials := fmt.Sprintf(`
Initialization Complete & Credentials
=====================================

Admin Credentials
-----------------
Username            : Admin User
Email               : admin@example.com
Password            : %s

System IDs (Ciphered)
---------------------
Users Pool ID       : %s
Encrypted Pool ID   : %s

App ID              : %s
Encrypted App ID    : %s

Use the Encrypted IDs in your request headers (X-Pool-Key, X-Public-Key)
`, adminPassword, usersPoolId, encryptedPoolId, appId, encryptedAppId)

	// Check if file exists and read content
	var finalContent []byte

	existingContent, err := os.ReadFile("credentials.txt")
	if err == nil && len(existingContent) > 0 {
		// File exists, prepend new content with separator
		finalContent = append([]byte(newCredentials), []byte(separator)...)
		finalContent = append(finalContent, existingContent...)
	} else {
		// New file
		finalContent = []byte(newCredentials)
	}

	// Write to credentials.txt
	err = os.WriteFile("credentials.txt", finalContent, 0644)
	if err != nil {
		fmt.Printf("%s✘ Failed to write credentials.txt: %v%s\n", constants.ColorRed, err, constants.ColorReset)
	} else {
		fmt.Printf("%s✔ Credentials saved to credentials.txt%s\n", constants.ColorGreen, constants.ColorReset)
	}

	// Print Credentials to Terminal
	fmt.Println()
	printHeader("Initialization Complete & Credentials")

	printSection("Admin Credentials")
	printKeyValue("Username", "Admin User")
	printKeyValue("Email", "admin@example.com")
	printKeyValue("Password", adminPassword)

	printSection("System IDs (Ciphered)")
	printKeyValue("Users Pool ID", usersPoolId)
	printKeyValue("Encrypted Pool ID", encryptedPoolId)
	fmt.Println()
	printKeyValue("App ID", appId)
	printKeyValue("Encrypted App ID", encryptedAppId)

	fmt.Println()
	fmt.Printf("%s%sUse the Encrypted IDs in your request headers (X-Pool-Key, X-Public-Key)%s\n", constants.ColorGray, constants.Bold, constants.ColorReset)
	fmt.Println()
}

func printHeader(text string) {
	fmt.Printf("%s%s=== %s ===%s\n", constants.ColorCyan, constants.Bold, text, constants.ColorReset)
}

func printSection(text string) {
	fmt.Printf("\n%s%s%s%s\n", constants.ColorYellow, constants.Bold, text, constants.ColorReset)
}

func printSuccess(text string) {
	fmt.Printf("%s✔ %s%s\n", constants.ColorGreen, text, constants.ColorReset)
}

func printError(msg string, err error) {
	fmt.Printf("%s✘ %s: %v%s\n", constants.ColorRed, msg, err, constants.ColorReset)
	// Don't panic here to allow deferred rollback to handle it gracefully if needed,
	// or panic if that is the desired behavior to stop immediately.
	// The previous code panic'd, so we'll keep it or let main return.
	// However, since we are inside main and want to rollback, returning is safer
	// BUT we need to make sure the deferred function sees the error or we just panic
	// and let the deferred recover handle it.
	panic(err)
}

func printKeyValue(key, value string) {
	fmt.Printf("%s%-20s:%s %s\n", constants.ColorBlue, key, constants.ColorReset, value)
}
