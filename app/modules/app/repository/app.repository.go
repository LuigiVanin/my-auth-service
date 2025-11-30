package repository

import (
	entity "auth_service/infra/entities"
	"encoding/json"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type AppRepository struct {
	client *sqlx.DB
}

var _ IAppRepository = &AppRepository{}

func NewAppRepository(client *sqlx.DB) IAppRepository {
	return &AppRepository{
		client: client,
	}
}

func (r *AppRepository) FindAppbyIdWithPool(id string) (*entity.AppWithPool, error) {
	query := `
	SELECT 
		-- users_pool columns
		up.name AS users_pool_name,
		up.id AS users_pool_id,
		up.created_at AS users_pool_created_at,
		up.updated_at AS users_pool_updated_at,

		-- apps columns
		a.id AS apps_id,
		a.users_pool_id AS apps_user_pools_id,
		a.name AS apps_name,
		a.public_key AS apps_public_key,
		a.secret_key AS apps_secret_key,
		a.private AS apps_private,
		a.verified_email_date AS apps_verified_email_date,
		a.login_types AS apps_login_types,
		a.token_type AS apps_token_type,
		a.token_expiration_time AS apps_token_expiration_time,
		a.metadata AS apps_metadata,
		a.created_at AS apps_created_at,
		a.updated_at AS apps_updated_at
		
	FROM users_pool AS up
		JOIN apps AS a
		ON a.users_pool_id = up.id
	WHERE a.id = $1
	`

	// Helper struct to scan the flat result
	var dest struct {
		PoolName      string    `db:"users_pool_name"`
		PoolID        string    `db:"users_pool_id"`
		PoolCreatedAt time.Time `db:"users_pool_created_at"`
		PoolUpdatedAt time.Time `db:"users_pool_updated_at"`

		AppID            string          `db:"apps_id"`
		AppUserPoolID    string          `db:"apps_user_pools_id"`
		AppName          string          `db:"apps_name"`
		AppPublicKey     string          `db:"apps_public_key"`
		AppSecretKey     string          `db:"apps_secret_key"`
		AppPrivate       bool            `db:"apps_private"`
		AppVerifiedEmail *time.Time      `db:"apps_verified_email_date"`
		AppLoginTypes    pq.StringArray  `db:"apps_login_types"`
		AppTokenType     string          `db:"apps_token_type"`
		AppTokenExpr     int64           `db:"apps_token_expiration_time"`
		AppMetadata      json.RawMessage `db:"apps_metadata"`
		AppCreatedAt     time.Time       `db:"apps_created_at"`
		AppUpdatedAt     time.Time       `db:"apps_updated_at"`
	}

	err := r.client.Get(&dest, query, id)
	if err != nil {
		return nil, err
	}

	// Map to domain entity
	appWithPool := &entity.AppWithPool{
		Pool: entity.UserPool{
			ID:        dest.PoolID,
			Name:      dest.PoolName,
			CreatedAt: dest.PoolCreatedAt,
			UpdatedAt: dest.PoolUpdatedAt,
		},

		App: entity.App{
			ID:                  dest.AppID,
			UsersPoolId:         dest.AppUserPoolID,
			Name:                dest.AppName,
			PublicKey:           dest.AppPublicKey,
			SecretKey:           dest.AppSecretKey,
			Private:             dest.AppPrivate,
			LoginTypes:          []string(dest.AppLoginTypes),
			TokenType:           dest.AppTokenType,
			TokenExpirationTime: dest.AppTokenExpr,
			Metadata:            dest.AppMetadata,
			CreatedAt:           dest.AppCreatedAt,
			UpdatedAt:           dest.AppUpdatedAt,
		},
	}
	return appWithPool, nil
}
