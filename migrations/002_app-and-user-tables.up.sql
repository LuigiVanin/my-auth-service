CREATE TYPE AUTH_METHOD AS ENUM ('WITH_LOGIN', 'WITH_OTP');
CREATE TYPE TOKEN_TYPE AS ENUM ('JWT', 'FAST_JWT', 'SESSION_UUID');

CREATE TABLE apps (
  id UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_pools_id UUID NOT NULL REFERENCES user_pools(id),

  code TEXT NOT NULL,
  name TEXT NOT NULL,
  secret_key TEXT NOT NULL,
  private BOOLEAN NOT NULL DEFAULT FALSE,

  login_types AUTH_METHOD[] NOT NULL,
  token_type TOKEN_TYPE NOT NULL,
  token_expiration_time bigint NOT NULL, -- in seconds and for refresh token only

  metadata JSONB NOT NULL DEFAULT '{}',
  
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

  CONSTRAINT apps_login_types_check CHECK (array_length(login_types, 1) > 0)
);

CREATE TABLE users (
  -- this should behave as a user public id, exposed to the client and used to identify the user
  id SERIAL NOT NULL PRIMARY KEY,
  -- this should behave as a user private id, only exposed internally and used to sign JWT and Sessions
  uuid UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
  user_pools_id UUID NOT NULL REFERENCES user_pools(id),
  
  name TEXT NOT NULL,
  email TEXT NOT NULL,
  phone TEXT,

  password_hash TEXT NOT NULL,

  metadata JSONB NOT NULL DEFAULT '{}',

  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

  CONSTRAINT users_email_user_pools_unique UNIQUE (email, user_pools_id)
);

CREATE INDEX users_email_idx ON users(email);
