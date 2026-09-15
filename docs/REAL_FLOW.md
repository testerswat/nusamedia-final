# Real Flow Contract

## Post
UI composer → `POST /api/v1/posts` → auth middleware → PostService → PostgreSQL `posts` → response → React state → post muncul di Feed.

## Like
Button → `POST /api/v1/posts/{id}/like` → authenticated user → transaction → insert/delete `post_likes` → recount → response `{liked,likes_count}` → UI state update.

## Comment
Composer → `POST /api/v1/posts/{id}/comments` → validation → insert `comments` → returned comment → UI append.

## Follow
Profile follow button → `POST /api/v1/users/{username}/follow` → resolve user → insert/delete `follows` → response `{following}` → profile state update.

No feature is accepted as complete until the output survives refresh and can be observed in database/API response.
