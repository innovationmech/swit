-- Create auth_service_db database
CREATE DATABASE IF NOT EXISTS auth_service_db DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- Use auth_service_db database
USE auth_service_db;

-- Create or modify tokens table
CREATE TABLE IF NOT EXISTS tokens (
    id CHAR(36) NOT NULL PRIMARY KEY,        -- UUID string, primary key
    user_id CHAR(36) NOT NULL,               -- Associated user ID
    access_token TEXT NOT NULL,              -- Access token string
    refresh_token TEXT NOT NULL,             -- Refresh token string
    access_expires_at TIMESTAMP NOT NULL,    -- Access token expiration time
    refresh_expires_at TIMESTAMP NOT NULL,   -- Refresh token expiration time
    is_valid BOOLEAN NOT NULL DEFAULT TRUE,  -- Whether the token is valid
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,  -- Record creation time
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,  -- Record update time
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES user_service_db.users(id)  -- Foreign key constraint
);