-- Seed test data for development
-- This script creates test user pools and apps with known credentials

-- ============================================
-- FIRST USER POOL AND APP
-- ============================================

-- Insert first test user pool
INSERT INTO users_pool (id, name)
VALUES ('00000000-0000-0000-0000-000000000001', 'Development Pool')
ON CONFLICT (id) DO NOTHING;

-- Insert first test app
INSERT INTO apps (
  users_pool_id,
  name,
  public_key,
  secret_key,
  login_types,
  token_type,
  token_expiration_time
)
VALUES (
  '00000000-0000-0000-0000-000000000001',
  'Main App',
  'app-public-key',
  'app-secret-key',
  ARRAY['WITH_LOGIN', 'WITH_OTP']::AUTH_METHOD[],
  'JWT'::TOKEN_TYPE,
  6000
)
ON CONFLICT DO NOTHING
RETURNING id, name, public_key, secret_key;

-- ============================================
-- SECOND USER POOL AND APP
-- ============================================

-- Insert second test user pool
INSERT INTO users_pool (id, name)
VALUES ('00000000-0000-0000-0000-000000000002', 'Production Pool')
ON CONFLICT (id) DO NOTHING;

-- Insert second test app
INSERT INTO apps (
  users_pool_id,
  name,
  public_key,
  secret_key,
  login_types,
  token_type,
  token_expiration_time
)
VALUES (
  '00000000-0000-0000-0000-000000000002',
  'Production App',
  'prod-app-public-key',
  'prod-app-secret-key',
  ARRAY['WITH_LOGIN']::AUTH_METHOD[],
  'FAST_JWT'::TOKEN_TYPE,
  7200
)
ON CONFLICT DO NOTHING
RETURNING id, name, public_key, secret_key;

-- ============================================
-- DISPLAY CREATED DATA
-- ============================================

-- Display all created pools and apps for easy reference
SELECT 
  up.id as pool_id,
  up.name as pool_name,
  a.id as app_id,
  a.name as app_name,
  a.public_key as "X-App-Key",
  a.secret_key,
  a.login_types,
  a.token_type,
  a.token_expiration_time
FROM users_pool up
LEFT JOIN apps a ON a.users_pool_id = up.id
WHERE up.id IN ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000002')
ORDER BY up.id, a.id;
