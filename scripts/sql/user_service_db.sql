-- Create user_service_db database
CREATE DATABASE IF NOT EXISTS user_service_db DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- Use user_service_db database
USE user_service_db;

-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id CHAR(36) NOT NULL PRIMARY KEY,          -- UUID string, primary key
    username VARCHAR(50) NOT NULL UNIQUE,      -- Username, unique constraint
    email VARCHAR(100) NOT NULL UNIQUE,        -- Email, unique constraint
    password_hash VARCHAR(255) NOT NULL,       -- Encrypted password hash
    role VARCHAR(20) NOT NULL DEFAULT 'user',  -- User role, default is 'user'
    is_active BOOLEAN NOT NULL DEFAULT TRUE,   -- Whether the user is active
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,  -- Record creation time
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP  -- Record update time
);