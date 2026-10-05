package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"mime"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"github.com/Swek09/lct-fcp-2026/backend/internal/db"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

type ctxKey int

const sessionKey ctxKey = 1

type Session struct {
	UserID  string `json:"user_id"`
	CSRF    string `json:"csrf"`
	Version int32  `json:"version"`
}
type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	Verified bool   `json:"verified"`
	CSRF     string `json:"csrf"`
}
type AuthResponse struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
	Body      User
}

func token() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
func digest(s string) string { v := sha256.Sum256([]byte(s)); return hex.EncodeToString(v[:]) }
func publicUser(u db.User, s Session) User {
	return User{u.ID, u.Email, u.Name, u.Role, u.Verified, s.CSRF}
}
func (a *App) sessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("fsp_session"); err == nil && len(c.Value) == 64 {
			raw, err := a.Redis.Get(r.Context(), "session:"+digest(c.Value)).Bytes()
			if err != nil && !errors.Is(err, redis.Nil) {
				http.Error(w, "Сервис сессий временно недоступен", 503)
				return
			}
			var s Session
			if err == nil && json.Unmarshal(raw, &s) == nil {
				publicAuth := r.URL.Path == "/api/auth/login" || r.URL.Path == "/api/auth/register" || r.URL.Path == "/api/auth/verify" || r.URL.Path == "/api/auth/reset" || r.URL.Path == "/api/auth/forgot"
				if r.Method != "GET" && r.Method != "HEAD" && !publicAuth && r.Header.Get("X-CSRF-Token") != s.CSRF {
					http.Error(w, "Обновите страницу и повторите действие", 403)
					return
				}
				r = r.WithContext(context.WithValue(r.Context(), sessionKey, s))
			}
		}
		next.ServeHTTP(w, r)
	})
}
func (a *App) require(ctx context.Context, role string) (db.User, error) {
	s, ok := ctx.Value(sessionKey).(Session)
	if !ok {
		return db.User{}, huma.Error401Unauthorized("Войдите в аккаунт")
	}
	u, err := a.Q.GetUser(ctx, s.UserID)
	if isMissing(err) {
		return u, huma.Error401Unauthorized("Войдите снова")
	}
	if err != nil {
		return u, internal(err)
	}
	if !u.Verified || u.AuthVersion != s.Version {
		return u, huma.Error401Unauthorized("Подтвердите почту и войдите снова")
	}
	if role != "" && u.Role != role {
		return u, huma.Error403Forbidden("Недостаточно прав")
	}
	return u, nil
}

var rateScript = redis.NewScript("local n=redis.call('INCR',KEYS[1]); if n==1 then redis.call('EXPIRE',KEYS[1],ARGV[1]); end; return n")

func (a *App) limit(ctx context.Context, key string, max int64, d time.Duration) error {
	n, err := rateScript.Run(ctx, a.Redis, []string{"rate:" + a.rateLimitPrefix + key}, int(d.Seconds())).Int64()
	if err != nil {
		return huma.Error503ServiceUnavailable("Попробуйте позже")
	}
	if n > max {
		return huma.Error429TooManyRequests("Слишком много попыток. Попробуйте позже.")
	}
	return nil
}
func validEmail(s string) bool {
	e, err := mail.ParseAddress(s)
	return err == nil && e.Address == s && !strings.ContainsAny(s, "\r\n")
}
func (a *App) cookie(value string, maxAge int) http.Cookie {
	return http.Cookie{Name: "fsp_session", Value: value, Path: "/", MaxAge: maxAge, HttpOnly: true, Secure: strings.HasPrefix(a.Config.AppURL, "https://"), SameSite: http.SameSiteLaxMode}
}
func (a *App) authRoutes(api huma.API) {
	type RegisterInput struct {
		Body struct {
			Email    string `json:"email" maxLength:"254"`
			Password string `json:"password" minLength:"10" maxLength:"72"`
			Name     string `json:"name" minLength:"2" maxLength:"100"`
			Role     string `json:"role" enum:"candidate,employer"`
			Consent  bool   `json:"consent"`
		}
	}
	huma.Post(api, "/api/auth/register", func(ctx context.Context, in *RegisterInput) (*Response[Message], error) {
		b := in.Body
		b.Email = strings.ToLower(strings.TrimSpace(b.Email))
		b.Name = strings.TrimSpace(b.Name)
		if !validEmail(b.Email) || len(b.Password) > 72 || len(b.Name) < 2 || !b.Consent {
			return nil, huma.Error400BadRequest("Проверьте почту, имя и согласие на обработку данных")
		}
		if err := a.limit(ctx, "register:"+digest(b.Email), 5, time.Hour); err != nil {
			return nil, err
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(b.Password), 12)
		if err != nil {
			return nil, internal(err)
		}
		tx, err := a.DB.Begin(ctx)
		if err != nil {
			return nil, internal(err)
		}
		defer tx.Rollback(ctx)
		id := uuid.NewString()
		err = db.New(tx).CreateUser(ctx, db.CreateUserParams{ID: id, Email: b.Email, PasswordHash: string(hash), Role: b.Role, Name: b.Name})
		if err != nil {
			var pg *pgconn.PgError
			if errors.As(err, &pg) && pg.Code == "23505" {
				return msg("Если адрес доступен для регистрации, письмо придёт в ближайшее время."), nil
			}
			return nil, internal(err)
		}
		if b.Role == "candidate" {
			_, err = tx.Exec(ctx, "INSERT INTO profiles(user_id) VALUES($1)", id)
		} else {
			_, err = tx.Exec(ctx, "INSERT INTO companies(user_id,name) VALUES($1,$2)", id, b.Name)
		}
		if err != nil {
			return nil, internal(err)
		}
		t := token()
		_, err = tx.Exec(ctx, "INSERT INTO email_tokens(digest,user_id,kind,expires_at) VALUES($1,$2,'verify',now()+interval '24 hours')", digest(t), id)
		if err != nil {
			return nil, internal(err)
		}
		_, err = tx.Exec(ctx, "INSERT INTO mail_outbox(recipient,subject,body) VALUES($1,$2,$3)", b.Email, "Подтвердите почту · ФСП Карьера", "Подтвердите регистрацию по ссылке:\n"+a.Config.AppURL+"/verify?token="+t+"\nСсылка действует 24 часа.")
		if err != nil {
			return nil, internal(err)
		}
		if err = tx.Commit(ctx); err != nil {
			return nil, internal(err)
		}
		return msg("Если адрес доступен для регистрации, письмо придёт в ближайшее время."), nil
	})
	type TokenInput struct {
		Body struct {
			Token string `json:"token" minLength:"64" maxLength:"64"`
		}
	}
	huma.Post(api, "/api/auth/verify", func(ctx context.Context, in *TokenInput) (*Response[Message], error) {
		tx, err := a.DB.Begin(ctx)
		if err != nil {
			return nil, internal(err)
		}
		defer tx.Rollback(ctx)
		var id string
		err = tx.QueryRow(ctx, "DELETE FROM email_tokens WHERE digest=$1 AND kind='verify' AND expires_at>now() RETURNING user_id", digest(in.Body.Token)).Scan(&id)
		if isMissing(err) {
			return nil, huma.Error400BadRequest("Ссылка недействительна или уже использована")
		}
		if err != nil {
			return nil, internal(err)
		}
		if err = db.New(tx).VerifyUser(ctx, id); err != nil {
			return nil, internal(err)
		}
		if err = tx.Commit(ctx); err != nil {
			return nil, internal(err)
		}
		return msg("Почта подтверждена. Теперь можно войти."), nil
	})
	type LoginInput struct {
		Body struct {
			Email    string `json:"email" maxLength:"254"`
			Password string `json:"password" maxLength:"72"`
		}
	}
	huma.Post(api, "/api/auth/login", func(ctx context.Context, in *LoginInput) (*AuthResponse, error) {
		email := strings.ToLower(strings.TrimSpace(in.Body.Email))
		if err := a.limit(ctx, "login:"+digest(email), 15, 15*time.Minute); err != nil {
			return nil, err
		}
		u, err := a.Q.GetUserByEmail(ctx, email)
		if err != nil || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(in.Body.Password)) != nil {
			return nil, huma.Error401Unauthorized("Неверная почта или пароль")
		}
		if !u.Verified {
			return nil, huma.Error403Forbidden("Сначала подтвердите почту")
		}
		s := Session{u.ID, token(), u.AuthVersion}
		raw, _ := json.Marshal(s)
		t := token()
		if err = a.Redis.Set(ctx, "session:"+digest(t), raw, 7*24*time.Hour).Err(); err != nil {
			return nil, internal(err)
		}
		return &AuthResponse{SetCookie: a.cookie(t, 7*86400), Body: publicUser(u, s)}, nil
	})
	register(api, "GET", "/api/auth/me", "Текущий пользователь", func(ctx context.Context, in *struct{}) (*Response[User], error) {
		u, err := a.require(ctx, "")
		if err != nil {
			return nil, err
		}
		s := ctx.Value(sessionKey).(Session)
		return ok(publicUser(u, s)), nil
	})
	type LogoutInput struct {
		Cookie string `cookie:"fsp_session"`
	}
	register(api, "POST", "/api/auth/logout", "Завершить сессию", func(ctx context.Context, in *LogoutInput) (*AuthResponse, error) {
		if in.Cookie != "" {
			if err := a.Redis.Del(ctx, "session:"+digest(in.Cookie)).Err(); err != nil {
				return nil, internal(err)
			}
		}
		return &AuthResponse{SetCookie: a.cookie("", -1)}, nil
	})
	type ForgotInput struct {
		Body struct {
			Email string `json:"email" maxLength:"254"`
		}
	}
	huma.Post(api, "/api/auth/forgot", func(ctx context.Context, in *ForgotInput) (*Response[Message], error) {
		email := strings.ToLower(strings.TrimSpace(in.Body.Email))
		if err := a.limit(ctx, "forgot:"+digest(email), 3, time.Hour); err != nil {
			return nil, err
		}
		result := msg("Если аккаунт существует, письмо с инструкцией отправлено.")
		u, err := a.Q.GetUserByEmail(ctx, email)
		if isMissing(err) {
			return result, nil
		}
		if err != nil {
			return nil, internal(err)
		}
		tx, err := a.DB.Begin(ctx)
		if err != nil {
			return nil, internal(err)
		}
		defer tx.Rollback(ctx)
		t := token()
		_, err = tx.Exec(ctx, "INSERT INTO email_tokens(digest,user_id,kind,expires_at) VALUES($1,$2,'reset',now()+interval '30 minutes')", digest(t), u.ID)
		if err != nil {
			return nil, internal(err)
		}
		_, err = tx.Exec(ctx, "INSERT INTO mail_outbox(recipient,subject,body) VALUES($1,$2,$3)", email, "Восстановление доступа · ФСП Карьера", "Задайте новый пароль:\n"+a.Config.AppURL+"/reset?token="+t+"\nСсылка действует 30 минут.")
		if err != nil {
			return nil, internal(err)
		}
		if err = tx.Commit(ctx); err != nil {
			return nil, internal(err)
		}
		return result, nil
	})
	type ResetInput struct {
		Body struct {
			Token    string `json:"token" minLength:"64" maxLength:"64"`
			Password string `json:"password" minLength:"10" maxLength:"72"`
		}
	}
	huma.Post(api, "/api/auth/reset", func(ctx context.Context, in *ResetInput) (*Response[Message], error) {
		if len(in.Body.Password) > 72 {
			return nil, huma.Error400BadRequest("Пароль слишком длинный")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(in.Body.Password), 12)
		if err != nil {
			return nil, internal(err)
		}
		tx, err := a.DB.Begin(ctx)
		if err != nil {
			return nil, internal(err)
		}
		defer tx.Rollback(ctx)
		var id string
		err = tx.QueryRow(ctx, "DELETE FROM email_tokens WHERE digest=$1 AND kind='reset' AND expires_at>now() RETURNING user_id", digest(in.Body.Token)).Scan(&id)
		if isMissing(err) {
			return nil, huma.Error400BadRequest("Ссылка недействительна или уже использована")
		}
		if err != nil {
			return nil, internal(err)
		}
		_, err = tx.Exec(ctx, "UPDATE users SET password_hash=$2,auth_version=auth_version+1 WHERE id=$1", id, string(hash))
		if err != nil {
			return nil, internal(err)
		}
		_, err = tx.Exec(ctx, "DELETE FROM email_tokens WHERE user_id=$1 AND kind='reset'", id)
		if err != nil {
			return nil, internal(err)
		}
		if err = tx.Commit(ctx); err != nil {
			return nil, internal(err)
		}
		return msg("Пароль изменён. Войдите снова."), nil
	})
}
func (a *App) deliverMail(ctx context.Context) error {
	tx, err := a.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var id int64
	var recipient, subject, body string
	err = tx.QueryRow(ctx, "SELECT id,recipient,subject,body FROM mail_outbox WHERE sent_at IS NULL AND attempts<10 ORDER BY id FOR UPDATE SKIP LOCKED LIMIT 1").Scan(&id, &recipient, &subject, &body)
	if isMissing(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var auth smtp.Auth
	if a.Config.SMTPUser != "" {
		auth = smtp.PlainAuth("", a.Config.SMTPUser, a.Config.SMTPPassword, a.Config.SMTPHost)
	}
	data := []byte("From: " + a.Config.SMTPFrom + "\r\nTo: " + recipient + "\r\nSubject: " + mime.QEncoding.Encode("UTF-8", subject) + "\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + body)
	sendErr := a.sendMail(auth, recipient, data)
	if sendErr != nil {
		_, err = tx.Exec(ctx, "UPDATE mail_outbox SET attempts=attempts+1 WHERE id=$1", id)
	} else {
		_, err = tx.Exec(ctx, "UPDATE mail_outbox SET sent_at=now(),body='' WHERE id=$1", id)
	}
	if err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	return sendErr
}

func (a *App) sendMail(auth smtp.Auth, recipient string, data []byte) error {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(a.Config.SMTPHost, a.Config.SMTPPort), 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err = conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return err
	}
	c, err := smtp.NewClient(conn, a.Config.SMTPHost)
	if err != nil {
		return err
	}
	defer c.Close()
	if supported, _ := c.Extension("STARTTLS"); supported {
		if err = c.StartTLS(&tls.Config{ServerName: a.Config.SMTPHost, MinVersion: tls.VersionTLS12}); err != nil {
			return err
		}
	} else if a.Config.Env == "production" {
		return errors.New("SMTP server must support STARTTLS in production")
	}
	if auth != nil {
		if err = c.Auth(auth); err != nil {
			return err
		}
	}
	if err = c.Mail(a.Config.SMTPFrom); err != nil {
		return err
	}
	if err = c.Rcpt(recipient); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err = w.Write(data); err != nil {
		return err
	}
	if err = w.Close(); err != nil {
		return err
	}
	return c.Quit()
}
