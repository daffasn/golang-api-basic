# 🚀 Panduan Membuat REST API dengan Go + PostgreSQL

## Untuk Pemula — Dijelaskan dari Nol!

---

## 📋 Daftar Isi
1. [Apa itu REST API?](#apa-itu-rest-api)
2. [Tools yang Digunakan](#tools-yang-digunakan)
3. [Persiapan & Instalasi](#persiapan--instalasi)
4. [Struktur Project](#struktur-project)
5. [Cara Menjalankan](#cara-menjalankan)
6. [Testing API](#testing-api--cara-mencoba)
7. [Konsep Penting](#konsep-penting)

---

## Apa itu REST API?

Bayangkan API seperti **pelayan restoran**:
- **Kamu (Client/Frontend)** = tamu yang pesan makanan
- **API** = pelayan yang menerima pesanan
- **Database** = dapur tempat makanan dibuat
- **Response** = makanan yang diantarkan ke mejamu

REST API berkomunikasi menggunakan **HTTP Method**:
| Method | Fungsi         | Contoh                    |
|--------|----------------|---------------------------|
| GET    | Ambil data     | Lihat profil user          |
| POST   | Kirim data baru| Daftar akun, Login         |
| PUT    | Update data    | Edit profil                |
| DELETE | Hapus data     | Hapus akun                 |

---

## Tools yang Digunakan

| Tool | Fungsi | Kenapa Dipilih |
|------|--------|----------------|
| **Go (Golang)** | Bahasa pemrograman | Super cepat, mudah dipelajari |
| **Gin** | Web framework | Ringan & tercepat untuk Go |
| **GORM** | ORM (akses database) | Tidak perlu nulis SQL manual |
| **PostgreSQL** | Database | Andal, gratis, populer |
| **JWT** | Autentikasi token | Standar industri untuk API |
| **bcrypt** | Enkripsi password | Sangat aman untuk hash password |

---

## Persiapan & Instalasi

### 1. Install Go
Download dari https://go.dev/dl/ — pilih versi terbaru

Verifikasi instalasi:
```bash
go version
# Output: go version go1.21.x ...
```

### 2. Install PostgreSQL
Download dari https://www.postgresql.org/download/

### 3. Setup Project

```bash
# Buat folder project
mkdir go-api
cd go-api

# Inisialisasi module Go (seperti package.json di Node.js)
go mod init go-api

# Install semua dependency sekaligus
go get github.com/gin-gonic/gin
go get github.com/golang-jwt/jwt/v5
go get github.com/joho/godotenv
go get golang.org/x/crypto/bcrypt
go get gorm.io/gorm
go get gorm.io/driver/postgres
```

### 4. Setup Database

Buka terminal PostgreSQL:
```bash
psql -U postgres
```

Jalankan perintah:
```sql
CREATE DATABASE go_api_db;
```

### 5. Setup File .env

```bash
# Salin file contoh
cp .env.example .env

# Edit file .env dengan text editor
nano .env  # atau buka manual dengan VSCode
```

Isi file `.env`:
```
DATABASE_URL=host=localhost user=postgres password=passwordmu dbname=go_api_db port=5432 sslmode=disable
JWT_SECRET=rahasia-panjang-acak-ganti-ini
PORT=8080
```

---

## Struktur Project

```
go-api/
├── main.go          ← File utama (semua kode ada di sini)
├── go.mod           ← Daftar dependency (seperti package.json)
├── go.sum           ← Hash dependency (otomatis dibuat)
├── .env             ← Konfigurasi rahasia (JANGAN di-upload ke Git!)
├── .env.example     ← Contoh .env (aman di-upload)
├── setup.sql        ← Script setup database
└── README.md        ← Dokumentasi ini
```

---

## Cara Menjalankan

```bash
# Pastikan kamu sudah di folder go-api
cd go-api

# Download semua dependency
go mod tidy

# Jalankan server
go run main.go
```

Output yang akan muncul:
```
✅ Database berhasil terhubung!
🚀 Server berjalan di http://localhost:8080
📋 Endpoints tersedia:
   GET  /                 → Health check
   POST /api/register     → Daftar akun baru
   POST /api/login        → Login
   GET  /api/dashboard    → Dashboard (butuh token)
   POST /api/logout       → Logout (butuh token)
```

---

## Testing API — Cara Mencoba

Gunakan **Postman**, **Insomnia**, atau **curl** di terminal.

### ✅ 1. Health Check
```bash
curl http://localhost:8080/
```
Response:
```json
{
  "message": "🚀 Go API berjalan dengan baik!",
  "version": "1.0.0"
}
```

---

### ✅ 2. Register (Daftar Akun)
```bash
curl -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Budi Santoso",
    "email": "budi@email.com",
    "password": "password123"
  }'
```
Response sukses:
```json
{
  "success": true,
  "message": "Registrasi berhasil! Silakan login.",
  "data": {
    "id": 1,
    "name": "Budi Santoso",
    "email": "budi@email.com"
  }
}
```

---

### ✅ 3. Login
```bash
curl -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "budi@email.com",
    "password": "password123"
  }'
```
Response sukses:
```json
{
  "success": true,
  "message": "Login berhasil!",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": 1,
      "name": "Budi Santoso",
      "email": "budi@email.com"
    }
  }
}
```
**⚠️ Simpan token ini!** Kamu butuh ini untuk mengakses endpoint yang dilindungi.

---

### ✅ 4. Dashboard (Butuh Token)
Ganti `TOKEN_DARI_LOGIN` dengan token yang didapat saat login.
```bash
curl http://localhost:8080/api/dashboard \
  -H "Authorization: Bearer TOKEN_DARI_LOGIN"
```
Response sukses:
```json
{
  "success": true,
  "message": "Selamat datang di dashboard!",
  "data": {
    "user": {
      "id": 1,
      "name": "Budi Santoso",
      "email": "budi@email.com",
      "member_since": "15 December 2024"
    },
    "stats": {
      "total_login": 42,
      "last_active": "15 December 2024, 10:30"
    }
  }
}
```

---

### ✅ 5. Logout
```bash
curl -X POST http://localhost:8080/api/logout \
  -H "Authorization: Bearer TOKEN_DARI_LOGIN"
```
Response:
```json
{
  "success": true,
  "message": "Logout berhasil. Sampai jumpa!",
  "hint": "Hapus token dari penyimpanan lokal (localStorage/cookie) di sisi frontend."
}
```

---

## Konsep Penting

### 🔐 Kenapa Password Di-hash dengan bcrypt?
Password TIDAK PERNAH disimpan dalam bentuk teks biasa.
```
Password asli : "password123"
Setelah bcrypt: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
```
Bahkan jika database dibobol, hacker tidak bisa tahu password aslinya!

### 🎫 Bagaimana JWT Bekerja?
```
1. User login → Server buat token JWT
2. Token dikirim ke user
3. User simpan token (di localStorage atau memory)
4. Setiap request ke endpoint privat, user kirim token di header
5. Server verifikasi token → jika valid, izinkan akses
```

Token JWT terdiri dari 3 bagian (dipisah titik):
```
header.payload.signature
eyJhbGc...  .  eyJ1c2Vy...  .  SflKxwRJ...
```

### 🛡️ HTTP Status Code yang Digunakan
| Kode | Arti |
|------|------|
| 200 | OK - Sukses |
| 201 | Created - Data berhasil dibuat |
| 400 | Bad Request - Data yang dikirim salah |
| 401 | Unauthorized - Belum login / token salah |
| 404 | Not Found - Data tidak ditemukan |
| 409 | Conflict - Misalnya email sudah ada |
| 500 | Internal Server Error - Error di server |

---

## 🚀 Tips Selanjutnya (Setelah Paham Dasar)

1. **Pisahkan kode** ke beberapa file (handlers/, models/, middleware/)
2. **Tambah validasi** lebih ketat dengan library validator
3. **Implementasi refresh token** agar tidak perlu login ulang
4. **Tambah rate limiting** untuk mencegah brute force
5. **Deploy ke cloud** (Railway, Render, Fly.io — semuanya gratis untuk start)
6. **Tambah logging** yang proper dengan library seperti zerolog
7. **Tulis unit test** untuk setiap handler

---

Selamat belajar! 🎉
