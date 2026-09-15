# NusaMedia Social Screen Specification

Semua halaman sosial mengikuti satu standar: **Layout → Feature → Function → Frontend Route → API Route → Data → Output → Loading/Error → E2E**.

## 1. Feed `/feed`
Layout: header, story strip, composer, post cards, engagement row, bottom navigation.
Fungsi: membaca feed publik, membuat post, like/unlike.
API: `GET /api/v1/feed`, `POST /api/v1/posts`, `POST /api/v1/posts/{id}/like`.
Output: JSON berisi post author, caption, verified, like count, comment count, liked state.

## 2. Post detail `/post/:id`
Layout: full post, media, engagement, comments, composer.
Fungsi: melihat detail dan membuat komentar.
API: `GET /api/v1/posts/{id}`, `POST /api/v1/posts/{id}/comments`.
Output: post detail + paginated comments.

## 3. Profile `/u/:username`
Layout: avatar, display name, badge, bio, follower/following counts, tabs Posts/Reels/Shop.
Fungsi: identitas akun, follow/unfollow.
API: `GET /api/v1/users/{username}`, `POST /api/v1/users/{username}/follow`.

## 4. Reels `/reels`
Layout: vertical fullscreen video, caption, music, engagement rail.
Fungsi: playback, like, comment, share, save, follow.
API target: `/api/v1/reels`, `/api/v1/reels/{id}/like`, `/api/v1/reels/{id}/comments`.

## 5. Stories `/stories`
Layout: avatar rail + fullscreen viewer + reply/reaction.
Fungsi: create, view, expire, viewer list.
API target: `/api/v1/stories`, `/api/v1/stories/{id}/view`.

## 6. Discover `/discover`
Layout: category chips, trending creators, media grid, UMKM discovery.
API target: `/api/v1/discover`.

## 7. Search `/search`
Layout: search field + tabs Users/Posts/Reels/Shops.
API target: `GET /api/v1/search?q=&type=`.

## 8. Notifications `/notifications`
Layout: grouped activity list and unread indicator.
API target: `GET /api/v1/notifications`, `POST /api/v1/notifications/read`.

## 9. Messages `/messages`
Layout: conversations + realtime message pane.
API target: `/api/v1/conversations`, `/api/v1/conversations/{id}/messages`.

## 10. Marketplace
`/shop/:slug`, `/product/:slug`, `/cart`, `/checkout`, `/orders`.
Semua harus berakhir pada persistent shop/product/cart/order/payment records; bukan simulasi tombol.
