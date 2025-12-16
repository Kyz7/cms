# Swagger Documentation Setup

Swagger UI telah berhasil diintegrasikan ke dalam aplikasi CMS Anda. Berikut adalah cara mengakses dan menggunakan dokumentasi API:

## Akses Swagger UI

Setelah menjalankan aplikasi CMS, Anda dapat mengakses Swagger UI melalui:

- **Swagger UI**: `http://localhost:8080/swagger/`
- **OpenAPI Spec (YAML)**: `http://localhost:8080/swagger/openapi.yaml`
- **OpenAPI Spec (JSON)**: `http://localhost:8080/swagger/openapi.json`

## Fitur Swagger UI

### 1. Dokumentasi API Lengkap
- Semua endpoint REST API terdokumentasi dengan detail
- Request/response schemas yang jelas
- Parameter dan body request yang diperlukan
- Status code dan error responses

### 2. Testing API Langsung
- **Try it out**: Klik tombol "Try it out" pada setiap endpoint
- **Execute**: Jalankan request langsung dari browser
- **Response**: Lihat response langsung dengan format JSON

### 3. Authentication
- **JWT Token**: Masukkan token JWT untuk mengakses endpoint yang memerlukan autentikasi
- **Token Input**: Gunakan input field di pojok kanan atas untuk memasukkan token
- **Auto-include**: Token akan otomatis disertakan dalam header Authorization

### 4. Kategori API
Swagger UI mengorganisir API berdasarkan kategori:
- **Health**: Health check endpoint
- **Auth**: Autentikasi dan autorisasi
- **Users**: Manajemen pengguna (Admin only)
- **Roles**: Manajemen role dan permission
- **Content**: Manajemen konten dan content types
- **Media**: Upload dan manajemen media
- **Workflow**: Workflow dan approval process
- **Search**: Pencarian dan filtering

## Cara Menggunakan

### 1. Akses Swagger UI
```
http://localhost:8080/swagger/
```

### 2. Login untuk Mendapatkan Token
1. Gunakan endpoint `/auth/login` untuk login
2. Copy `access_token` dari response
3. Paste token ke input field di pojok kanan atas Swagger UI
4. Klik "Set Token"

### 3. Test Endpoint
1. Pilih endpoint yang ingin ditest
2. Klik "Try it out"
3. Isi parameter yang diperlukan
4. Klik "Execute"
5. Lihat response

## Contoh Penggunaan

### Login
```json
POST /auth/login
{
  "email": "admin@example.com",
  "password": "password123"
}
```

### Create Content Type
```json
POST /content/types
{
  "name": "Article",
  "slug": "article"
}
```

### Upload Media
```
POST /media/upload
Content-Type: multipart/form-data

file: [binary file]
folder: "images"
alt: "Article image"
```

## Troubleshooting

### Token Expired
- Jika token expired, gunakan endpoint `/auth/refresh` untuk memperbarui token
- Atau login ulang menggunakan `/auth/login`

### Permission Denied
- Pastikan user memiliki role yang sesuai
- Check permission pada role user

### File Upload Issues
- Pastikan file size tidak melebihi limit (100MB)
- Gunakan content-type yang sesuai

## Development Notes

- Swagger UI files di-embed dalam aplikasi menggunakan Go's embed feature
- OpenAPI spec di-load dari file `openapi.yaml` di root project
- Semua static files (CSS, JS, images) di-serve dari embedded filesystem
- Custom JavaScript ditambahkan untuk token management

## File Structure

```
internal/swagger/
├── handler.go              # Swagger handler implementation
└── swagger-ui/             # Embedded Swagger UI files
    ├── index.html          # Main HTML file
    ├── swagger-ui.css      # Swagger UI styles
    ├── swagger-ui-bundle.js # Swagger UI JavaScript
    ├── swagger-ui-standalone-preset.js
    ├── favicon-32x32.png
    └── favicon-16x16.png
```

