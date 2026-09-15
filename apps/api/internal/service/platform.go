package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Platform struct{ DB *sql.DB }

func (p Platform) Profile(ctx context.Context, id string) (map[string]any, error) {
	var username, email, name, account, role, bio, avatar, interests, loc string
	var verified bool
	e := p.DB.QueryRowContext(ctx, `SELECT username,email,display_name,account_type,role,verified,bio,avatar_url,interests,location_text FROM users WHERE id=$1`, id).Scan(&username, &email, &name, &account, &role, &verified, &bio, &avatar, &interests, &loc)
	if e != nil {
		return nil, e
	}
	return map[string]any{"id": id, "username": username, "email": email, "displayName": name, "accountType": account, "role": role, "verified": verified, "bio": bio, "avatarUrl": avatar, "interests": interests, "location": loc}, nil
}
func (p Platform) UpdateProfile(ctx context.Context, id string, v map[string]string) error {
	_, e := p.DB.ExecContext(ctx, `UPDATE users SET display_name=COALESCE(NULLIF($2,''),display_name),bio=COALESCE($3,bio),avatar_url=COALESCE($4,avatar_url),interests=COALESCE($5,interests),location_text=COALESCE($6,location_text),phone=COALESCE(NULLIF($7,''),phone) WHERE id=$1`, id, v["displayName"], v["bio"], v["avatarUrl"], v["interests"], v["location"], v["phone"])
	return e
}
func (p Platform) Notifications(ctx context.Context, id string) ([]map[string]any, error) {
	rows, e := p.DB.QueryContext(ctx, `SELECT id,type,title,body,read_at,created_at FROM notifications WHERE user_id=$1 ORDER BY created_at DESC LIMIT 50`, id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var i, t, ti, b string
		var read, created sql.NullString
		if e = rows.Scan(&i, &t, &ti, &b, &read, &created); e != nil {
			return nil, e
		}
		out = append(out, map[string]any{"id": i, "type": t, "title": ti, "body": b, "readAt": read.String, "createdAt": created.String})
	}
	return out, rows.Err()
}
func (p Platform) Save(ctx context.Context, post, user string) (bool, error) {
	var ex bool
	if e := p.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM saves WHERE post_id=$1 AND user_id=$2)`, post, user).Scan(&ex); e != nil {
		return false, e
	}
	var err error
	if ex {
		_, err = p.DB.ExecContext(ctx, `DELETE FROM saves WHERE post_id=$1 AND user_id=$2`, post, user)
	} else {
		_, err = p.DB.ExecContext(ctx, `INSERT INTO saves(post_id,user_id) VALUES($1,$2)`, post, user)
	}
	return !ex, err
}
func (p Platform) Share(ctx context.Context, post, user string) error {
	_, e := p.DB.ExecContext(ctx, `INSERT INTO shares(post_id,user_id) VALUES($1,$2)`, post, user)
	return e
}
func (p Platform) Comments(ctx context.Context, post string) ([]map[string]any, error) {
	rows, e := p.DB.QueryContext(ctx, `SELECT c.id,u.username,u.display_name,c.body,c.created_at::text FROM comments c JOIN users u ON u.id=c.user_id WHERE c.post_id=$1 ORDER BY c.created_at ASC`, post)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var i, u, n, b, c string
		if e = rows.Scan(&i, &u, &n, &b, &c); e != nil {
			return nil, e
		}
		out = append(out, map[string]any{"id": i, "username": u, "displayName": n, "body": b, "createdAt": c})
	}
	return out, rows.Err()
}
func (p Platform) Search(ctx context.Context, q string) ([]map[string]any, error) {
	q = "%" + strings.ToLower(q) + "%"
	rows, e := p.DB.QueryContext(ctx, `SELECT id::text,display_name,username,'USER' FROM users WHERE lower(display_name) LIKE $1 OR lower(username) LIKE $1 UNION ALL SELECT id::text,caption,'post','POST' FROM posts WHERE lower(caption) LIKE $1 ORDER BY 2 LIMIT 30`, q)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, title, sub, typ string
		if e = rows.Scan(&id, &title, &sub, &typ); e != nil {
			return nil, e
		}
		out = append(out, map[string]any{"id": id, "title": title, "subtitle": sub, "type": typ})
	}
	return out, rows.Err()
}
func (p Platform) CreateVerification(ctx context.Context, user, typ, doc string) (map[string]any, error) {
	if typ == "" || typ == "PERSONAL" {
		return nil, fmt.Errorf("verification type required")
	}
	var id string
	e := p.DB.QueryRowContext(ctx, `INSERT INTO verification_requests(user_id,requested_type,document_url) VALUES($1,$2,$3) ON CONFLICT(user_id) DO UPDATE SET requested_type=EXCLUDED.requested_type,document_url=EXCLUDED.document_url,status='PENDING',updated_at=now() RETURNING id::text`, user, typ, doc).Scan(&id)
	return map[string]any{"id": id, "status": "PENDING", "requestedType": typ}, e
}
func (p Platform) Products(ctx context.Context) ([]map[string]any, error) {
	rows, e := p.DB.QueryContext(ctx, `SELECT p.id::text,p.name,p.description,p.price,p.stock,s.name FROM products p JOIN shops s ON s.id=p.shop_id ORDER BY p.created_at DESC LIMIT 100`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, n, d, s string
		var price int64
		var stock int
		if e = rows.Scan(&id, &n, &d, &price, &stock, &s); e != nil {
			return nil, e
		}
		out = append(out, map[string]any{"id": id, "name": n, "description": d, "price": price, "stock": stock, "seller": s})
	}
	return out, rows.Err()
}
func (p Platform) CreateOrder(ctx context.Context, buyer, product string, qty int) (map[string]any, error) {
	tx, e := p.DB.BeginTx(ctx, nil)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	var price int64
	var stock int
	if e = tx.QueryRowContext(ctx, `SELECT price,stock FROM products WHERE id=$1 FOR UPDATE`, product).Scan(&price, &stock); e != nil {
		return nil, e
	}
	if qty < 1 || stock < qty {
		return nil, fmt.Errorf("insufficient stock")
	}
	total := price * int64(qty)
	var id string
	if e = tx.QueryRowContext(ctx, `UPDATE products SET stock=stock-$1 WHERE id=$2 RETURNING $2::text`, qty, product).Scan(&product); e != nil {
		return nil, e
	}
	if e = tx.QueryRowContext(ctx, `INSERT INTO orders(buyer_id,product_id,quantity,total_amount) VALUES($1,$2,$3,$4) RETURNING id::text`, buyer, product, qty, total).Scan(&id); e != nil {
		return nil, e
	}
	if e = tx.Commit(); e != nil {
		return nil, e
	}
	return map[string]any{"id": id, "total": total, "status": "PENDING_PAYMENT", "paymentStatus": "UNPAID"}, nil
}
func (p Platform) PayOrder(ctx context.Context, user, order string) (map[string]any, error) {
	tx, e := p.DB.BeginTx(ctx, nil)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	var total int64
	var buyer string
	if e = tx.QueryRowContext(ctx, `SELECT total_amount,buyer_id::text FROM orders WHERE id=$1 FOR UPDATE`, order).Scan(&total, &buyer); e != nil {
		return nil, e
	}
	if buyer != user {
		return nil, fmt.Errorf("forbidden")
	}
	var bal int64
	if e = tx.QueryRowContext(ctx, `SELECT balance FROM wallet_accounts WHERE user_id=$1 FOR UPDATE`, user).Scan(&bal); e != nil {
		if e == sql.ErrNoRows {
			_, _ = tx.ExecContext(ctx, `INSERT INTO wallet_accounts(user_id,balance) VALUES($1,0)`, user)
			bal = 0
		} else {
			return nil, e
		}
	}
	if bal < total {
		return nil, fmt.Errorf("insufficient balance")
	}
	if _, e = tx.ExecContext(ctx, `UPDATE wallet_accounts SET balance=balance-$1,updated_at=now() WHERE user_id=$2`, total, user); e != nil {
		return nil, e
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO wallet_transactions(user_id,type,amount,description) VALUES($1,'PAYMENT',$2,$3)`, user, -total, "Pembayaran order "+order)
	if e != nil {
		return nil, e
	}
	_, e = tx.ExecContext(ctx, `UPDATE orders SET status='PAID',payment_status='PAID',updated_at=now() WHERE id=$1`, order)
	if e != nil {
		return nil, e
	}
	if e = tx.Commit(); e != nil {
		return nil, e
	}
	return map[string]any{"id": order, "status": "PAID", "paymentStatus": "PAID", "amount": total}, nil
}
func (p Platform) Wallet(ctx context.Context, user string) (map[string]any, error) {
	var bal int64
	e := p.DB.QueryRowContext(ctx, `SELECT balance FROM wallet_accounts WHERE user_id=$1`, user).Scan(&bal)
	if e == sql.ErrNoRows {
		_, e = p.DB.ExecContext(ctx, `INSERT INTO wallet_accounts(user_id,balance) VALUES($1,0)`, user)
		bal = 0
	}
	return map[string]any{"balance": bal}, e
}
func (p Platform) Topup(ctx context.Context, user string, amount int64) (map[string]any, error) {
	if amount <= 0 || amount > 100000000 {
		return nil, fmt.Errorf("invalid amount")
	}
	tx, e := p.DB.BeginTx(ctx, nil)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	_, e = tx.ExecContext(ctx, `INSERT INTO wallet_accounts(user_id,balance) VALUES($1,$2) ON CONFLICT(user_id) DO UPDATE SET balance=wallet_accounts.balance+$2,updated_at=now()`, user, amount)
	if e != nil {
		return nil, e
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO wallet_transactions(user_id,type,amount,description) VALUES($1,'TOPUP',$2,'Top up')`, user, amount)
	if e != nil {
		return nil, e
	}
	var bal int64
	e = tx.QueryRowContext(ctx, `SELECT balance FROM wallet_accounts WHERE user_id=$1`, user).Scan(&bal)
	if e != nil {
		return nil, e
	}
	if e = tx.Commit(); e != nil {
		return nil, e
	}
	return map[string]any{"balance": bal, "amount": amount}, nil
}
func (p Platform) AdminDashboard(ctx context.Context) (map[string]any, error) {
	out := map[string]any{}
	for _, q := range []struct{ k, q string }{{"users", "SELECT count(*) FROM users"}, {"posts", "SELECT count(*) FROM posts"}, {"products", "SELECT count(*) FROM products"}, {"orders", "SELECT count(*) FROM orders"}, {"reports", "SELECT count(*) FROM reports WHERE status='OPEN'"}, {"verification", "SELECT count(*) FROM verification_requests WHERE status='PENDING'"}} {
		var n int64
		if e := p.DB.QueryRowContext(ctx, q.q).Scan(&n); e != nil {
			return nil, e
		}
		out[q.k] = n
	}
	out["generatedAt"] = time.Now().UTC().Format(time.RFC3339)
	return out, nil
}
func JSON(v any) string { b, _ := json.Marshal(v); return string(b) }
