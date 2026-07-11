# Dinsos Kubu Raya Backend API

> REST API backend untuk sistem informasi Dinas Sosial Kabupaten Kubu Raya, dibangun menggunakan Go dengan framework Gin, dilengkapi upload file via Cloudinary, notifikasi real-time via WebSocket, dan integrasi Firebase.

---

## Tentang Proyek

DinsosBackend adalah layanan API untuk mendukung operasional digital Dinas Sosial Kabupaten Kubu Raya. Sistem ini mengelola manajemen pengguna (admin & staff), dokumen masuk dan keluar, disposisi dokumen kepada staf, notifikasi, serta pencatatan aktivitas — semua melalui REST API yang aman dengan autentikasi JWT, rate limiting, proteksi CORS, dan perlindungan XSS.

Proyek ini dilengkapi dengan **WebSocket** untuk notifikasi real-time dan **Firebase Admin SDK** untuk layanan autentikasi dan cloud tambahan. Background worker berjalan otomatis untuk membersihkan log aktivitas dan notifikasi yang sudah kedaluwarsa.

---

## Fitur & Modul

| Modul | Deskripsi |
|---|---|
| **Pengguna** | Manajemen akun admin dan staff |
| **Autentikasi** | Login & logout berbasis JWT dengan SecretToken |
| **Dokumen** | CRUD dokumen masuk (surat/berkas), upload PDF & gambar ke Cloudinary |
| **Dokumen Staf** | Dokumen yang dimiliki atau dikirim oleh staf |
| **Superior Orders** | Disposisi dokumen dari admin ke satu atau beberapa staf |
| **Notifikasi** | Sistem notifikasi dengan dukungan real-time via WebSocket |
| **Log Aktivitas** | Pencatatan aktivitas pengguna secara otomatis |
| **WebSocket** | Komunikasi real-time untuk notifikasi live |

---

## Tech Stack

| Teknologi | Versi | Fungsi |
|---|---|---|
| [Go](https://go.dev) | 1.24.4 | Bahasa pemrograman |
| [Gin](https://gin-gonic.com) | v1.11.0 | HTTP framework |
| [GORM](https://gorm.io) | v1.31.1 | ORM |
| [MySQL](https://mysql.com) | (via GORM driver) | Database |
| [JWT](https://jwt.io) | v5.3.0 | Autentikasi token |
| [bcrypt](https://pkg.go.dev/golang.org/x/crypto) | v0.45.0 | Hashing password |
| [UUID](https://github.com/google/uuid) | v1.6.0 | Generate ID unik |
| [Cloudinary](https://cloudinary.com) | (via utils) | Upload & manajemen file |
| [WebSocket](https://github.com/gorilla/websocket) | v1.5.3 | Notifikasi real-time |
| [Firebase Admin SDK](https://firebase.google.com/docs/admin/setup) | v3.13.0 | Layanan Firebase |
| [Rate Limiter](https://github.com/ulule/limiter) | v3.11.2 | Pembatasan request API |
| [CORS Middleware](https://github.com/gin-contrib/cors) | v1.7.6 | Konfigurasi CORS |
| [godotenv](https://github.com/joho/godotenv) | v1.5.1 | Manajemen env variables |

---

## Struktur Proyek

```
DinsosBackend/
├── config/           # Koneksi database & inisialisasi Firebase
├── controllers/      # Handler untuk setiap route
├── middleware/       # RateLimiter, CORS, XSSBlocker
├── models/           # Definisi struct & skema database
├── routes/           # Pendaftaran semua route
├── services/         # Business logic & integrasi Cloudinary
├── utils/            # Fungsi utilitas, background cleaner
├── websocket/        # Hub & handler WebSocket
├── main.go           # Entry point & inisialisasi server
├── go.mod
└── go.sum
```

---

## Model Database

```
User            — Akun pengguna (admin & staff)
Document        — Dokumen masuk/surat dinas
SecretToken     — Token sesi autentikasi JWT
DocumentStaff   — Dokumen milik atau yang dikirim staf
Notification    — Notifikasi untuk pengguna
ActivityLog     — Riwayat aktivitas pengguna
```

Relasi `SuperiorOrder` (disposisi dokumen ke staf) dikelola secara relasional melalui `Document` dan `User`.

Server melakukan **AutoMigrate** otomatis saat pertama kali dijalankan.

---

## Middleware

| Middleware | Fungsi |
|---|---|
| `RateLimiter` | Membatasi jumlah request per IP untuk mencegah abuse |
| `CORSMiddleware` | Mengatur izin akses lintas origin |
| `XSSBlocker` | Memblokir request yang mengandung payload XSS |

---

## Background Workers

Dua goroutine berjalan otomatis di background sejak server pertama kali dijalankan:

| Worker | Fungsi |
|---|---|
| `StartActivityLogCleaner` | Menghapus log aktivitas yang sudah kedaluwarsa secara berkala |
| `StartNotificationCleaner` | Menghapus notifikasi lama secara berkala |

---

## API Routes

Semua endpoint diawali dengan prefix `/api`.

### Users

| Method | Endpoint | Deskripsi |
|---|---|---|
| `POST` | `/api/users/admin` | Buat akun admin baru |
| `POST` | `/api/users/staff` | Buat akun staff baru |
| `GET` | `/api/users` | Ambil semua pengguna |
| `GET` | `/api/users/:id` | Ambil pengguna berdasarkan ID |
| `PUT` | `/api/users/:id` | Perbarui data pengguna |
| `DELETE` | `/api/users/:id` | Hapus pengguna |

### Autentikasi

| Method | Endpoint | Deskripsi |
|---|---|---|
| `POST` | `/api/login` | Login, mengembalikan `token_id` |
| `POST` | `/api/logout` | Logout, menghapus sesi token |

### Dokumen

| Method | Endpoint | Deskripsi |
|---|---|---|
| `POST` | `/api/documents` | Upload dokumen baru (PDF/gambar) |
| `GET` | `/api/documents` | Ambil semua dokumen |
| `GET` | `/api/documents/:id` | Ambil dokumen berdasarkan ID |
| `PUT` | `/api/documents/:id` | Perbarui dokumen |
| `DELETE` | `/api/documents/:id` | Hapus dokumen |

### Dokumen Staf

| Method | Endpoint | Deskripsi |
|---|---|---|
| `POST` | `/api/document_staff` | Upload dokumen staf baru |
| `GET` | `/api/document_staff` | Ambil semua dokumen staf |
| `GET` | `/api/document_staff/:id` | Ambil dokumen staf berdasarkan ID |
| `PUT` | `/api/document_staff/:id` | Perbarui dokumen staf |
| `DELETE` | `/api/document_staff/:id` | Hapus dokumen staf |

### Superior Orders (Disposisi)

| Method | Endpoint | Deskripsi |
|---|---|---|
| `POST` | `/api/superior_orders` | Disposisikan dokumen ke beberapa staf |
| `GET` | `/api/superior_orders` | Ambil semua disposisi |
| `GET` | `/api/superior_orders/:document_id` | Ambil disposisi berdasarkan dokumen |
| `PUT` | `/api/superior_orders/:document_id` | Perbarui disposisi |
| `DELETE` | `/api/superior_orders/:document_id` | Hapus semua disposisi dokumen |

### Notifikasi & Log Aktivitas

| Method | Endpoint | Deskripsi |
|---|---|---|
| `GET/POST/DELETE` | `/api/notifications/...` | Manajemen notifikasi |
| `GET/DELETE` | `/api/activity_logs/...` | Riwayat aktivitas |

### WebSocket

| Endpoint | Deskripsi |
|---|---|
| `WS /api/ws` | Koneksi WebSocket untuk notifikasi real-time |

---

## Environment Variables

Buat file `.env` di root project:

```env
# Server
PORT=8000

# Database
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=your_password
DB_NAME=dinsos_kuburaya

# JWT
JWT_SECRET=your_jwt_secret

# Cloudinary
CLOUDINARY_CLOUD_NAME=your_cloud_name
CLOUDINARY_API_KEY=your_api_key
CLOUDINARY_API_SECRET=your_api_secret

# Firebase
FIREBASE_CREDENTIALS_PATH=./firebase-credentials.json
```

---

## Memulai (Development)

### Prasyarat

- Go >= 1.21
- MySQL (lokal atau cloud)
- Akun [Cloudinary](https://cloudinary.com) (gratis tersedia)
- Akun [Firebase](https://firebase.google.com) dengan service account JSON

### Instalasi

```bash
# Clone repository
git clone https://github.com/DinsosKubuRaya/DinsosBackend.git
cd DinsosBackend

# Download dependencies
go mod tidy

# Buat file .env dan isi sesuai konfigurasi
cp .env.example .env
```

### Menjalankan Server

```bash
go run main.go
```

Server akan berjalan di port `8000` (default). AutoMigrate, WebSocket hub, dan background worker akan diinisialisasi otomatis.

### Build Binary

```bash
go build -o dinsos-backend main.go
./dinsos-backend
```

---
