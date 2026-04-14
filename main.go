package main

// ================================================================
// 🗺️  PETA PROGRAM INI (Baca ini dulu sebelum lihat kodenya!)
// ================================================================
//
// Program ini adalah sebuah REST API — yaitu program yang menerima
// permintaan dari luar (seperti Postman atau aplikasi mobile) dan
// membalasnya dengan data dalam format JSON.
//
// Bayangkan seperti sebuah WARUNG:
//   - Kamu (Postman/Frontend) = Pelanggan
//   - API ini               = Kasir/Pelayan
//   - Database PostgreSQL   = Gudang stok barang
//
// Alur lengkap program:
//
//   [Pelanggan kirim request]
//           │
//           ▼
//   [Router] ← Seperti "papan menu", menentukan handler mana yang dipanggil
//           │
//           ├─── POST /api/register  →  handler: daftarUser()
//           ├─── POST /api/login     →  handler: loginUser()
//           │
//           └─── [Penjaga/Middleware] ← Cek token dulu sebelum masuk
//                       │
//                       ├─── GET  /api/dashboard  →  handler: dashboard()
//                       └─── POST /api/logout     →  handler: logoutUser()
//
// ================================================================
// 📦 DAFTAR TOOLS YANG KITA PAKAI
// ================================================================
//
//  fiber    → Framework untuk membuat API (seperti Express.js di Node.js)
//  gorm     → Alat bantu untuk akses database tanpa nulis SQL manual
//  postgres → "Jembatan" antara GORM dan database PostgreSQL
//  bcrypt   → Untuk mengacak/mengenkripsi password
//  jwt      → Untuk membuat "kartu tanda masuk" digital setelah login
//  godotenv → Untuk membaca file .env (file konfigurasi rahasia)
//
// ================================================================

import (
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ================================================================
// 📌 BAGIAN 1: MODEL (Cetakan Data)
// ================================================================
//
// "Struct" di Go itu seperti FORMULIR KOSONG.
// Kita mendefinisikan kolom apa saja yang ada, baru nanti diisi datanya.
//
// Struct User di bawah ini = formulir data pengguna.
// GORM akan otomatis membuat tabel "users" di PostgreSQL
// berdasarkan struct ini. Jadi kita tidak perlu buat tabel manual!
//
// Penjelasan tag:
//   `gorm:"primaryKey"`   → ini kolom ID utama (unik, auto increment)
//   `gorm:"unique"`       → nilai di kolom ini tidak boleh sama/duplikat
//   `gorm:"not null"`     → kolom ini wajib diisi, tidak boleh kosong
//   `json:"nama_field"`   → nama yang tampil saat data dikirim sebagai JSON
//   `json:"-"`            → field ini DISEMBUNYIKAN dari response JSON
//                           (dipakai untuk Password agar tidak bocor!)
//
// ================================================================

type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"not null"   json:"name"`
	Email     string         `gorm:"unique;not null" json:"email"`
	Password  string         `gorm:"not null"   json:"-"`       // ← password tidak ikut tampil
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`            // ← untuk soft delete
}

// ================================================================
// 📌 BAGIAN 2: INPUT (Apa yang Diterima dari Pelanggan)
// ================================================================
//
// Kita perlu struct terpisah untuk menerima data dari request.
// Kenapa tidak pakai struct User langsung?
// → Karena saat register, kita tidak butuh semua field User.
//   Misalnya, ID dan CreatedAt tidak perlu diisi oleh user.
//
// RegisterInput = data yang dikirim saat mendaftar
// LoginInput    = data yang dikirim saat login
//
// ================================================================

type RegisterInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ================================================================
// 📌 BAGIAN 3: VARIABEL GLOBAL
// ================================================================
//
// Variabel global = variabel yang bisa diakses dari SEMUA fungsi
// di file ini. Kita simpan di sini karena dibutuhkan di banyak tempat.
//
// DB          → koneksi ke database. Dibuat sekali, dipakai terus.
// jwtSecret   → "kunci rahasia" untuk membuat & memverifikasi token JWT.
//               Harus dijaga kerahasiaannya! Simpan di file .env.
// tokenHitam  → daftar token yang sudah tidak berlaku (sudah logout).
//               Bentuknya: map[tokenString]waktuExpired
//               Contoh: {"abc123": 1703000000, "xyz789": 1703001000}
//
// ================================================================

var DB *gorm.DB
var jwtSecret []byte
var tokenHitam = map[string]int64{} // Token yang sudah logout disimpan di sini

// ================================================================
// 📌 BAGIAN 4: KONEKSI DATABASE
// ================================================================
//
// Fungsi ini bertugas "membuka pintu" ke database PostgreSQL.
// Dipanggil sekali saja saat program pertama kali dijalankan.
//
// DSN (Data Source Name) = alamat + kredensial database.
// Format: "host=... user=... password=... dbname=... port=..."
//
// db.AutoMigrate(&User{}) → GORM otomatis cek tabel "users":
//   - Jika belum ada → dibuat
//   - Jika sudah ada tapi ada kolom baru → ditambahkan
//   - Jika sudah sesuai → tidak diubah
// Jadi kita tidak perlu buka pgAdmin atau psql untuk buat tabel!
//
// ================================================================

func koneksiDatabase() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// Nilai default jika .env tidak ada (untuk development)
		dsn = "host=localhost user=postgres password=postgres dbname=go_api_db port=5432 sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		// log.Fatal = tampilkan error lalu HENTIKAN program.
		// Tidak ada gunanya lanjut jika tidak bisa konek ke database.
		log.Fatal("❌ Gagal konek ke database! Pastikan PostgreSQL berjalan. Error:", err)
	}

	// Buat tabel otomatis berdasarkan struct User
	if err := db.AutoMigrate(&User{}); err != nil {
		log.Fatal("❌ Gagal membuat tabel:", err)
	}

	DB = db
	log.Println("✅ Berhasil terhubung ke database!")
}

// ================================================================
// 📌 BAGIAN 5: JWT — KARTU TANDA MASUK DIGITAL
// ================================================================
//
// JWT (JSON Web Token) itu seperti KARTU IDENTITAS di konser musik:
//
//   1. Kamu beli tiket (login dengan email + password)
//   2. Panitia kasih gelang/kartu (server kasih token JWT)
//   3. Setiap mau masuk area VIP, kamu tunjukkan kartu (kirim token)
//   4. Panitia cek kartu — valid? Boleh masuk. Tidak valid? Diusir.
//
// Token JWT bentuknya seperti ini (3 bagian dipisah titik):
//   eyJhbGc.eyJ1c2Vy.SflKxwRJ
//   [header].[payload].[signature]
//
// Di dalam payload kita simpan:
//   user_id → siapa pemilik token ini
//   exp     → kapan token kedaluwarsa (24 jam dari sekarang)
//
// Fungsi buatToken() → membuat token baru saat login berhasil
// Fungsi cekToken()  → memverifikasi token yang dikirim user
//
// ================================================================

// buatToken: membuat token JWT untuk user yang berhasil login
// Mengembalikan: (string token, waktu expired, error jika gagal)
func buatToken(userID uint) (string, int64, error) {
	waktuExpired := time.Now().Add(24 * time.Hour).Unix() // 24 jam dari sekarang

	// "Claims" = data yang kita simpan di dalam token
	isiToken := jwt.MapClaims{
		"user_id": userID,        // ID user pemilik token
		"exp":     waktuExpired,  // kapan token kedaluwarsa
	}

	// Buat token dengan algoritma HS256 (standar industri)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, isiToken)

	// "Tanda tangani" token dengan jwtSecret agar tidak bisa dipalsukan
	tokenString, err := token.SignedString(jwtSecret)

	return tokenString, waktuExpired, err
}

// cekToken: memverifikasi token yang dikirim oleh user
// Mengembalikan: (isi/claims token, error jika tidak valid)
func cekToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("token tidak valid atau sudah kedaluwarsa")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("token rusak")
	}

	return claims, nil
}

// ================================================================
// 📌 BAGIAN 6: PENJAGA (MIDDLEWARE)
// ================================================================
//
// Middleware = kode yang berjalan SEBELUM handler utama.
// Fungsinya seperti SATPAM di pintu masuk gedung:
//
//   ❌ Tidak ada kartu/token      → "Silakan pergi!"
//   ❌ Kartu ada di daftar hitam  → "Kartu Anda sudah tidak berlaku!"
//   ❌ Kartu palsu atau kedaluarsa → "Kartu tidak valid!"
//   ✅ Kartu valid                → "Silakan masuk!"
//
// Urutan pemeriksaan:
//   1. Cek: apakah ada token di header?
//   2. Cek: apakah token ada di daftar hitam (sudah logout)?  ← PENTING!
//   3. Cek: apakah token valid dan belum kedaluarsa?
//   4. Jika semua lolos → simpan user_id ke "context" lalu lanjutkan
//
// c.Next() → artinya "lanjutkan ke handler berikutnya"
// c.Abort() di Gin → di Fiber kita langsung return setelah kirim response
//
// ================================================================

func penjagaLogin(c *fiber.Ctx) error {
	// Langkah 1: Ambil token dari header "Authorization"
	// Format yang benar: "Bearer eyJhbGci..."
	headerAuth := c.Get("Authorization")
	if headerAuth == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Akses ditolak. Kamu belum login. Silakan login terlebih dahulu.",
		})
	}

	// Ambil token saja, buang kata "Bearer " di depannya
	// Contoh: "Bearer abc123" → "abc123"
	if len(headerAuth) <= 7 || headerAuth[:7] != "Bearer " {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Format token salah. Harus: 'Bearer <token>'",
		})
	}
	tokenString := headerAuth[7:]

	// Langkah 2: Cek apakah token ini ada di daftar hitam
	// Token masuk daftar hitam ketika user logout.
	// Ini adalah solusi agar token lama tidak bisa dipakai setelah logout!
	if _, sudahLogout := tokenHitam[tokenString]; sudahLogout {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Token sudah tidak berlaku karena kamu sudah logout. Silakan login lagi.",
		})
	}

	// Langkah 3: Verifikasi token (cek tanda tangan & tanggal kedaluwarsa)
	isiToken, err := cekToken(tokenString)
	if err != nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Token tidak valid: " + err.Error(),
		})
	}

	// Langkah 4: Simpan data ke "context" agar bisa dipakai oleh handler
	// c.Locals() = "tas sementara" yang ikut bersama request ini
	c.Locals("user_id", uint(isiToken["user_id"].(float64)))
	c.Locals("token", tokenString)
	c.Locals("token_exp", int64(isiToken["exp"].(float64)))

	// Semua pemeriksaan lolos! Lanjutkan ke handler
	return c.Next()
}

// ================================================================
// 📌 BAGIAN 7: HANDLER REGISTER
// ================================================================
//
// Handler = fungsi yang memproses sebuah request dan mengirim response.
//
// Endpoint : POST /api/register
// Akses    : Bebas (siapa saja bisa akses, tidak perlu token)
//
// Contoh request body (JSON):
//   {
//     "name": "Budi",
//     "email": "budi@gmail.com",
//     "password": "rahasia123"
//   }
//
// Contoh response sukses:
//   {
//     "success": true,
//     "message": "Registrasi berhasil!",
//     "data": { "id": 1, "name": "Budi", "email": "budi@gmail.com" }
//   }
//
// Alur proses:
//   1. Terima data JSON dari request
//   2. Validasi: nama, email, password tidak boleh kosong
//   3. Cek: email sudah terdaftar? Jika ya → tolak
//   4. Enkripsi password dengan bcrypt
//   5. Simpan user baru ke database
//   6. Kirim response sukses
//
// ================================================================

func daftarUser(c *fiber.Ctx) error {
	// Langkah 1: Ambil data JSON dari body request
	// BodyParser = alat untuk "membongkar" JSON menjadi struct Go
	var input RegisterInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Data yang dikirim tidak terbaca. Pastikan formatnya JSON.",
		})
	}

	// Langkah 2: Validasi input
	// Kita cek satu per satu agar pesan error-nya spesifik dan jelas
	if input.Name == "" || len(input.Name) < 2 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Nama wajib diisi dan minimal 2 karakter.",
		})
	}
	if input.Email == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Email wajib diisi.",
		})
	}
	if len(input.Password) < 6 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Password wajib diisi dan minimal 6 karakter.",
		})
	}

	// Langkah 3: Cek apakah email sudah terdaftar
	// DB.Unscoped() → cek SEMUA data, termasuk yang sudah di-soft-delete
	// (soft-delete = data dihapus tapi masih ada di database, hanya ditandai)
	var userLama User
	hasil := DB.Unscoped().Where("email = ?", input.Email).First(&userLama)

	if hasil.Error == nil {
		// User dengan email ini ditemukan di database
		if userLama.DeletedAt.Valid {
			// Email ini pernah dipakai tapi akunnya sudah dihapus
			// Kita hapus permanen agar bisa daftar ulang
			DB.Unscoped().Delete(&userLama)
		} else {
			// Email masih aktif → tolak pendaftaran
			return c.Status(http.StatusConflict).JSON(fiber.Map{
				"success": false,
				"message": "Email ini sudah terdaftar. Gunakan email lain atau langsung login.",
			})
		}
	} else if !errors.Is(hasil.Error, gorm.ErrRecordNotFound) {
		// Ada error tak terduga dari database (bukan sekedar "tidak ditemukan")
		log.Println("Error database:", hasil.Error)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Terjadi masalah di server. Coba lagi nanti.",
		})
	}

	// Langkah 4: Enkripsi password
	//
	// MENGAPA password harus dienkripsi?
	// → Bayangkan database bocor ke hacker. Jika password tersimpan
	//   sebagai teks biasa "rahasia123", hacker langsung bisa login!
	// → Dengan bcrypt, yang tersimpan adalah:
	//   "$2a$10$N9qo8uLOickgx2ZMRZoMye..."  (tidak bisa dibaca!)
	// → Bahkan developer sendiri tidak tahu password asli usernya.
	//
	// bcrypt.DefaultCost = tingkat kesulitan enkripsi (10 = cukup aman & cepat)
	passwordTerenkripsi, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Gagal mengamankan password. Coba lagi.",
		})
	}

	// Langkah 5: Simpan user baru ke database
	userBaru := User{
		Name:     input.Name,
		Email:    input.Email,
		Password: string(passwordTerenkripsi), // simpan versi yang sudah dienkripsi!
	}

	if err := DB.Create(&userBaru).Error; err != nil {
		log.Println("Gagal menyimpan user:", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Gagal menyimpan data. Coba lagi.",
		})
	}

	// Langkah 6: Kirim response sukses
	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "Registrasi berhasil! Silakan login.",
		"data": fiber.Map{
			"id":    userBaru.ID,
			"name":  userBaru.Name,
			"email": userBaru.Email,
		},
	})
}

// ================================================================
// 📌 BAGIAN 8: HANDLER LOGIN
// ================================================================
//
// Endpoint : POST /api/login
// Akses    : Bebas (siapa saja bisa akses)
//
// Contoh request body:
//   {
//     "email": "budi@gmail.com",
//     "password": "rahasia123"
//   }
//
// Contoh response sukses:
//   {
//     "success": true,
//     "message": "Login berhasil!",
//     "data": {
//       "token": "eyJhbGci...",
//       "user": { "id": 1, "name": "Budi", "email": "budi@gmail.com" }
//     }
//   }
//
// Alur proses:
//   1. Terima email dan password dari request
//   2. Cari user di database berdasarkan email
//   3. Bandingkan password input dengan hash di database
//   4. Jika cocok → buat token JWT → kirim ke user
//
// ================================================================

func loginUser(c *fiber.Ctx) error {
	var input LoginInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Data yang dikirim tidak terbaca. Pastikan formatnya JSON.",
		})
	}

	if input.Email == "" || input.Password == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Email dan password wajib diisi.",
		})
	}

	// Cari user berdasarkan email di database
	var user User
	if err := DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		// Sengaja pesan error dibuat umum (tidak bilang "email tidak ditemukan")
		// Tujuannya: agar hacker tidak tahu email mana yang terdaftar
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Email atau password salah.",
		})
	}

	// Bandingkan password yang diketik user dengan hash di database
	//
	// bcrypt.CompareHashAndPassword secara cerdas bisa membandingkan:
	//   "rahasia123"  ←→  "$2a$10$N9qo8uLOickgx2ZMRZoMye..."
	// Tanpa perlu "membuka" enkripsinya. Aman!
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Email atau password salah.",
		})
	}

	// Password cocok! Buat token JWT sebagai "kartu tanda masuk"
	tokenString, waktuExpired, err := buatToken(user.ID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Gagal membuat token. Coba lagi.",
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Login berhasil! Simpan token di bawah ini.",
		"data": fiber.Map{
			// Token ini harus disimpan oleh user (di Postman, localStorage, dll)
			// dan dikirim di setiap request ke endpoint yang butuh login
			"token":          tokenString,
			"token_berlaku_sampai": time.Unix(waktuExpired, 0).Format("02 Jan 2006, 15:04 WIB"),
			"user": fiber.Map{
				"id":    user.ID,
				"name":  user.Name,
				"email": user.Email,
			},
		},
	})
}

// ================================================================
// 📌 BAGIAN 9: HANDLER DASHBOARD
// ================================================================
//
// Endpoint : GET /api/dashboard
// Akses    : PRIVAT — harus login dulu (ada penjagaLogin di depannya)
//
// Cara akses:
//   Tambahkan di header request:
//   Authorization: Bearer eyJhbGci...  (token dari hasil login)
//
// Jika tidak ada token atau token salah → penjagaLogin akan menolak
// dan handler ini tidak akan pernah dipanggil.
//
// ================================================================

func dashboard(c *fiber.Ctx) error {
	// Ambil user_id yang sudah disimpan oleh penjagaLogin
	// c.Locals() = "tas sementara" yang diisi middleware sebelumnya
	userID := c.Locals("user_id").(uint)

	// Cari data lengkap user dari database berdasarkan ID
	var user User
	if err := DB.First(&user, userID).Error; err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Akun tidak ditemukan.",
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Selamat datang, " + user.Name + "! 👋",
		"data": fiber.Map{
			"id":           user.ID,
			"name":         user.Name,
			"email":        user.Email,
			"bergabung":    user.CreatedAt.Format("02 January 2006"),
			"waktu_akses":  time.Now().Format("02 Jan 2006, 15:04"),
		},
	})
}

// ================================================================
// 📌 BAGIAN 10: HANDLER LOGOUT
// ================================================================
//
// Endpoint : POST /api/logout
// Akses    : PRIVAT — harus login dulu
//
// KENAPA LOGOUT DI JWT RUMIT?
// JWT itu seperti fotokopi KTP. Saat kamu bilang "saya tidak mau pakai
// KTP ini lagi", fotokopian yang sudah beredar tetap bisa dipakai orang lain.
//
// Server tidak menyimpan daftar token yang aktif, jadi server tidak
// bisa "menghapus" token begitu saja.
//
// SOLUSINYA: Daftar Hitam (tokenHitam)
// Kita catat token mana yang sudah logout. Setiap request masuk,
// penjagaLogin akan cek: "Apakah token ini ada di daftar hitam?"
// Jika iya → ditolak, meskipun token secara teknis masih valid!
//
// ================================================================

func logoutUser(c *fiber.Ctx) error {
	// Ambil token dan waktu expired dari context (diisi oleh penjagaLogin)
	tokenString := c.Locals("token").(string)
	tokenExp := c.Locals("token_exp").(int64)

	// Masukkan token ke daftar hitam
	// Mulai sekarang, token ini akan ditolak oleh penjagaLogin!
	tokenHitam[tokenString] = tokenExp

	log.Printf("🔒 User ID %v berhasil logout", c.Locals("user_id"))

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Logout berhasil! Token kamu sudah tidak berlaku. Sampai jumpa! 👋",
	})
}

// ================================================================
// 📌 BAGIAN 11: MAIN — TITIK AWAL PROGRAM
// ================================================================
//
// Fungsi main() adalah titik pertama yang dijalankan Go.
// Di sini kita:
//   1. Load konfigurasi (.env)
//   2. Konek ke database
//   3. Buat aplikasi Fiber
//   4. Daftarkan semua endpoint (route)
//   5. Jalankan server
//
// ================================================================

func main() {
	// Langkah 1: Load file .env
	// File .env berisi konfigurasi rahasia seperti password database
	// Jika file tidak ada, program tetap jalan dengan nilai default
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  File .env tidak ditemukan, menggunakan nilai default")
	}

	// Langkah 2: Setup JWT Secret
	// Secret ini harus panjang, acak, dan RAHASIA.
	// Jangan pernah tulis langsung di kode! Simpan di .env
	jwtSecret = []byte(os.Getenv("JWT_SECRET"))
	if len(jwtSecret) == 0 {
		jwtSecret = []byte("secret-default-ganti-di-env!")
		log.Println("⚠️  JWT_SECRET tidak ada di .env, menggunakan nilai default")
	}

	// Langkah 3: Konek ke database
	koneksiDatabase()

	// Langkah 4: Buat aplikasi Fiber
	app := fiber.New(fiber.Config{
		AppName: "Belajar Go API v1.0",
	})

	// Middleware: Logger → tampilkan log setiap ada request masuk
	// Contoh output: [10:30:45] 200 POST /api/login - 15ms
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} ${method} ${path} - ${latency}\n",
	}))

	// Middleware: CORS → izinkan request dari browser atau aplikasi lain
	// Tanpa ini, request dari frontend (misal React) akan diblokir browser
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE",
	}))

	// Langkah 5: Daftarkan semua endpoint (route)

	// Route bebas: tidak perlu login
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "🚀 API berjalan! Selamat datang."})
	})

	api := app.Group("/api") // Semua route di bawah ini berawalan /api

	// --- Endpoint yang bisa diakses siapa saja ---
	api.Post("/register", daftarUser)  // POST /api/register
	api.Post("/login",    loginUser)   // POST /api/login

	// --- Endpoint yang HANYA bisa diakses setelah login ---
	// penjagaLogin akan berjalan SEBELUM handler di dalamnya
	area_privat := api.Group("/", penjagaLogin)
	area_privat.Get("/dashboard", dashboard)   // GET  /api/dashboard
	area_privat.Post("/logout",   logoutUser)  // POST /api/logout

	// Langkah 6: Jalankan server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("🚀 Server jalan di → http://localhost:%s", port)
	log.Println("")
	log.Println("📋 Daftar Endpoint:")
	log.Println("   GET  /                → Cek server")
	log.Println("   POST /api/register    → Daftar akun baru")
	log.Println("   POST /api/login       → Login")
	log.Println("   GET  /api/dashboard   → Dashboard  🔒 (butuh token)")
	log.Println("   POST /api/logout      → Logout     🔒 (butuh token)")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if err := app.Listen(":" + port); err != nil {
		log.Fatal("❌ Server gagal jalan:", err)
	}
}