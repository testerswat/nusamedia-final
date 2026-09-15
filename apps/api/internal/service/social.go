package service

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"strings"
)

type Social struct{ DB *sql.DB }
type User struct {
	ID, Username, Email, DisplayName, AccountType, Role string
	Verified                                            bool
}
type Post struct {
	ID, Username, DisplayName, Caption string
	Verified                           bool
	Likes, Comments                    int
	Liked                              bool
	CreatedAt                          string
}

func (s Social) Register(ctx context.Context, username, email, password, name, account string) (User, error) {
	h, e := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if e != nil {
		return User{}, e
	}
	id := uuid.New()
	account = "PERSONAL"
	_, e = s.DB.ExecContext(ctx, `INSERT INTO users(id,username,email,password_hash,display_name,account_type) VALUES($1,$2,$3,$4,$5,$6)`, id, strings.ToLower(username), strings.ToLower(email), string(h), name, account)
	if e != nil {
		return User{}, e
	}
	return User{ID: id.String(), Username: username, Email: email, DisplayName: name, AccountType: account, Role: "USER"}, nil
}
func (s Social) Login(ctx context.Context, identity, password string) (User, error) {
	var u User
	var h string
	e := s.DB.QueryRowContext(ctx, `SELECT id,username,email,password_hash,display_name,account_type,role,verified FROM users WHERE username=$1 OR email=$1`, strings.ToLower(identity)).Scan(&u.ID, &u.Username, &u.Email, &h, &u.DisplayName, &u.AccountType, &u.Role, &u.Verified)
	if e != nil {
		return User{}, e
	}
	if bcrypt.CompareHashAndPassword([]byte(h), []byte(password)) != nil {
		return User{}, fmt.Errorf("invalid credentials")
	}
	return u, nil
}
func (s Social) CreatePost(ctx context.Context, userID, caption, visibility string) (Post, error) {
	id := uuid.New()
	_, e := s.DB.ExecContext(ctx, `INSERT INTO posts(id,author_id,caption,visibility) VALUES($1,$2,$3,$4)`, id, userID, caption, visibility)
	if e != nil {
		return Post{}, e
	}
	return s.GetPost(ctx, id.String(), userID)
}
func (s Social) GetPost(ctx context.Context, id, userID string) (Post, error) {
	var p Post
	var liked bool
	e := s.DB.QueryRowContext(ctx, `SELECT p.id,u.username,u.display_name,p.caption,u.verified,(SELECT count(*) FROM post_likes l WHERE l.post_id=p.id),(SELECT count(*) FROM comments c WHERE c.post_id=p.id),EXISTS(SELECT 1 FROM post_likes l2 WHERE l2.post_id=p.id AND l2.user_id=$2),p.created_at::text FROM posts p JOIN users u ON u.id=p.author_id WHERE p.id=$1`, id, userID).Scan(&p.ID, &p.Username, &p.DisplayName, &p.Caption, &p.Verified, &p.Likes, &p.Comments, &liked, &p.CreatedAt)
	p.Liked = liked
	return p, e
}
func (s Social) Feed(ctx context.Context, userID string, limit int) ([]Post, error) {
	rows, e := s.DB.QueryContext(ctx, `SELECT p.id,u.username,u.display_name,p.caption,u.verified,(SELECT count(*) FROM post_likes l WHERE l.post_id=p.id),(SELECT count(*) FROM comments c WHERE c.post_id=p.id),EXISTS(SELECT 1 FROM post_likes l2 WHERE l2.post_id=p.id AND l2.user_id=$1),p.created_at::text FROM posts p JOIN users u ON u.id=p.author_id WHERE p.visibility='PUBLIC' ORDER BY p.created_at DESC LIMIT $2`, userID, limit)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Post{}
	for rows.Next() {
		var p Post
		if e := rows.Scan(&p.ID, &p.Username, &p.DisplayName, &p.Caption, &p.Verified, &p.Likes, &p.Comments, &p.Liked, &p.CreatedAt); e != nil {
			return nil, e
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
func (s Social) ToggleLike(ctx context.Context, post, user string) (bool, int, error) {
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return false, 0, e
	}
	defer tx.Rollback()
	var exists bool
	e = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM post_likes WHERE post_id=$1 AND user_id=$2)`, post, user).Scan(&exists)
	if e != nil {
		return false, 0, e
	}
	if exists {
		_, e = tx.ExecContext(ctx, `DELETE FROM post_likes WHERE post_id=$1 AND user_id=$2`, post, user)
	} else {
		_, e = tx.ExecContext(ctx, `INSERT INTO post_likes(post_id,user_id) VALUES($1,$2)`, post, user)
	}
	if e != nil {
		return false, 0, e
	}
	var count int
	e = tx.QueryRowContext(ctx, `SELECT count(*) FROM post_likes WHERE post_id=$1`, post).Scan(&count)
	if e != nil {
		return false, 0, e
	}
	if e = tx.Commit(); e != nil {
		return false, 0, e
	}
	return !exists, count, nil
}
func (s Social) Comment(ctx context.Context, post, user, body string) (map[string]any, error) {
	id := uuid.New()
	var created string
	e := s.DB.QueryRowContext(ctx, `INSERT INTO comments(id,post_id,user_id,body) VALUES($1,$2,$3,$4) RETURNING created_at::text`, id, post, user, body).Scan(&created)
	if e != nil {
		return nil, e
	}
	return map[string]any{"id": id.String(), "body": body, "created_at": created}, nil
}
func (s Social) Follow(ctx context.Context, follower, following string) (bool, error) {
	var exists bool
	e := s.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM follows WHERE follower_id=$1 AND following_id=$2)`, follower, following).Scan(&exists)
	if e != nil {
		return false, e
	}
	if exists {
		_, e = s.DB.ExecContext(ctx, `DELETE FROM follows WHERE follower_id=$1 AND following_id=$2`, follower, following)
	} else {
		_, e = s.DB.ExecContext(ctx, `INSERT INTO follows(follower_id,following_id) VALUES($1,$2)`, follower, following)
	}
	return !exists, e
}
