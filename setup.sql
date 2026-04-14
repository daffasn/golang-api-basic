-- ============================================================
-- FILE SQL SETUP DATABASE
-- Jalankan perintah ini di PostgreSQL sebelum menjalankan API
-- ============================================================

-- 1. Buat database baru
CREATE DATABASE go_api_db;

-- 2. Hubungkan ke database
\c go_api_db

-- 3. Tabel users akan dibuat OTOMATIS oleh GORM (AutoMigrate)
--    Tapi jika ingin buat manual, ini strukturnya:

CREATE TABLE IF NOT EXISTS users (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,
    email       VARCHAR(100) UNIQUE NOT NULL,
    password    VARCHAR(255) NOT NULL,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at  TIMESTAMP WITH TIME ZONE  -- Untuk soft delete (data tidak benar-benar dihapus)
);

-- Index untuk mempercepat pencarian berdasarkan email
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- Tampilkan struktur tabel
\d users
