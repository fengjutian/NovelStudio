package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

var ErrNotFound = errors.New("auth record not found")
var ErrEmailTaken = errors.New("email already registered")

type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
}

type Store interface {
	CreateUser(context.Context, User, string) error
	UserByEmail(context.Context, string) (User, string, error)
	UserBySession(context.Context, string, time.Time) (User, error)
	CreateSession(context.Context, string, string, time.Time) error
	DeleteSession(context.Context, string) error
}

type MemoryStore struct {
	mu       sync.RWMutex
	users    map[string]memoryUser
	sessions map[string]memorySession
}
type memoryUser struct {
	user         User
	passwordHash string
}
type memorySession struct {
	userID    string
	expiresAt time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{users: map[string]memoryUser{}, sessions: map[string]memorySession{}}
}
func (s *MemoryStore) CreateUser(_ context.Context, user User, hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := NormalizeEmail(user.Email)
	if _, ok := s.users[key]; ok {
		return ErrEmailTaken
	}
	s.users[key] = memoryUser{user, hash}
	return nil
}
func (s *MemoryStore) UserByEmail(_ context.Context, email string) (User, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.users[NormalizeEmail(email)]
	if !ok {
		return User{}, "", ErrNotFound
	}
	return item.user, item.passwordHash, nil
}
func (s *MemoryStore) CreateSession(_ context.Context, tokenHash, userID string, expires time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[tokenHash] = memorySession{userID, expires}
	return nil
}
func (s *MemoryStore) UserBySession(_ context.Context, tokenHash string, now time.Time) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[tokenHash]
	if !ok || !session.expiresAt.After(now) {
		return User{}, ErrNotFound
	}
	for _, item := range s.users {
		if item.user.ID == session.userID {
			return item.user, nil
		}
	}
	return User{}, ErrNotFound
}
func (s *MemoryStore) DeleteSession(_ context.Context, hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, hash)
	return nil
}

func NormalizeEmail(value string) string { return strings.ToLower(strings.TrimSpace(value)) }
func NewID(prefix string) string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return prefix + base64.RawURLEncoding.EncodeToString(b)
}
func NewToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
func TokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", sum[:])
}

// HashPassword uses PBKDF2-HMAC-SHA256 with a per-password random salt.
func HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	digest := pbkdf2([]byte(password), salt, 210000, 32)
	return fmt.Sprintf("pbkdf2_sha256$210000$%s$%s", base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(digest)), nil
}
func CheckPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2_sha256" {
		return false
	}
	rounds, err := strconv.Atoi(parts[1])
	if err != nil || rounds < 100000 {
		return false
	}
	salt, e1 := base64.RawStdEncoding.DecodeString(parts[2])
	expected, e2 := base64.RawStdEncoding.DecodeString(parts[3])
	if e1 != nil || e2 != nil {
		return false
	}
	actual := pbkdf2([]byte(password), salt, rounds, len(expected))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}
func pbkdf2(password, salt []byte, rounds, size int) []byte {
	out := make([]byte, 0, size)
	var block uint32 = 1
	for len(out) < size {
		mac := hmac.New(sha256.New, password)
		mac.Write(salt)
		mac.Write([]byte{byte(block >> 24), byte(block >> 16), byte(block >> 8), byte(block)})
		u := mac.Sum(nil)
		t := append([]byte(nil), u...)
		for i := 1; i < rounds; i++ {
			mac = hmac.New(sha256.New, password)
			mac.Write(u)
			u = mac.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		out = append(out, t...)
		block++
	}
	return out[:size]
}
