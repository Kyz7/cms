# Live Preview (Strapi-style)

Panduan singkat untuk FE menggunakan fitur live preview via token.

## Ringkasan alur
1) FE (authenticated editor) memanggil BE untuk membuat token preview.
2) FE membuka halaman preview (FE) dengan query `entry_id` dan `token`.
3) Halaman preview FE memanggil endpoint preview BE memakai token (tanpa Bearer).

## Endpoint BE
- POST `/content/entries/{entry_id}/preview-token`
  - Auth: Bearer + permission `ContentEntry:read`.
  - Respon: `{ token, expires_at, preview_url, frontend_preview_url }`.
- GET `/content/entries/{entry_id}/preview?token=...`
  - Auth: memakai token preview saja (no Bearer).
  - Respon: full data entry (draft/published) untuk dirender FE.

## Env BE
- `PREVIEW_SECRET` (disarankan, min 32 chars). Jika kosong fallback `JWT_SECRET`.
- `PREVIEW_TOKEN_TTL_MINUTES` default 60.
- `PREVIEW_FRONTEND_URL` opsional; jika diisi, `frontend_preview_url` akan memakai base ini (mis: `https://fe.local/preview`).

## Alur FE yang direkomendasikan
1) Saat editor klik “Preview”:
   - Panggil POST `/content/entries/{id}/preview-token` (pakai Bearer login).
   - Ambil `frontend_preview_url` jika tersedia, jika tidak pakai `preview_url` (backend).
2) FE buka tab/window ke URL tersebut (cukup open location).
3) Halaman preview FE:
   - Baca `entry_id` + `token` dari query.
   - Panggil `GET /content/entries/{entry_id}/preview?token=...`.
   - Render layout publik dengan data yang diterima (hindari cache).

## Catatan implementasi FE
- Jangan kirim Bearer di request preview; cukup token query/header `X-Preview-Token` jika mau.
- Token kadaluarsa sesuai `expires_at`; fallback dengan refresh token (langkah 1) bila 401/403.
- Pastikan tidak membiarkan crawler mengindeks halaman preview (noindex).
- Jika `PREVIEW_FRONTEND_URL` di-set, query final akan berbentuk: `{PREVIEW_FRONTEND_URL}?entry_id=123&token=abc`.

## Contoh respons preview-token
```json
{
  "token": "signed.jwt.token",
  "expires_at": "2025-01-01T00:00:00Z",
  "preview_url": "https://api.local/content/entries/123/preview?token=...",
  "frontend_preview_url": "https://fe.local/preview?entry_id=123&token=..."
}
```

## Contoh fetch di FE (pseudo)
```js
const url = new URL(window.location.href);
const entryId = url.searchParams.get("entry_id");
const token = url.searchParams.get("token");
const resp = await fetch(`${API_BASE}/content/entries/${entryId}/preview?token=${encodeURIComponent(token)}`);
const data = await resp.json(); // { data: { ...entry }, message: "..."}
renderPage(data.data);
```

