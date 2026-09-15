# NusaMedia Final — Public Entry UI

The public entry screen is the mobile-first `/` experience before authentication.

## UI direction
- White, elegant base with restrained NusaMedia red.
- Indonesian cultural visual language: archipelago/map, temple silhouette, landscape and red-white wave.
- Surface is intentionally simple: brand, value proposition, contextual discovery preview, account actions, and four Explore cards.
- Explore cards: Kuliner, Seni Budaya, Produk Lokal, Reels.
- The public page does not expose account-type selection.

## Authentication
- `#auth` contains Login/Register in one modern flow.
- Registration uses 5 steps: Identitas, Kontak, Minat, Lokasi, Keamanan.
- Account type is intentionally absent from registration. It is selected later during verification application.
- Username is entered without `@` and visually rendered uppercase. Backend remains case-insensitive by normalizing username storage.

## Functional routes
- `/` equivalent: hash `#home`
- `#auth`: login/register
- `#discover`: contextual discovery preview
- `#feed`: authenticated feed

## Functional API contracts
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/register`
- `GET /api/v1/discover/nearby`
- `GET /api/v1/feed`
- `POST /api/v1/posts`
- `POST /api/v1/posts/{id}/like`

The UI is an implementation surface, not a screenshot asset: controls route into the existing API layer and authenticated state.
