//go:build ignore

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"ai-ats-platform/backend/internal/auth"
	"ai-ats-platform/backend/internal/config"
	"ai-ats-platform/backend/internal/database"
	"ai-ats-platform/backend/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	email := envOrDefault("EMAIL", "admin@ats.local")
	password := envOrDefault("PASSWORD", "Admin@12345")

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	pool, err := database.ConnectDB(cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "db connect: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	ctx := context.Background()
	users := repository.NewUserRepository(pool)

	user, err := users.GetByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "user lookup (%s): %v\n", email, err)
		os.Exit(1)
	}

	hash := user.PasswordHash
	fmt.Printf("email=%s role=%s is_active=%v\n", user.Email, user.Role, user.IsActive)
	fmt.Printf("hash_prefix=%s hash_len=%d\n", safePrefix(hash, 7), len(hash))

	if _, err := bcrypt.Cost([]byte(hash)); err != nil {
		fmt.Printf("bcrypt_format_valid=false err=%v\n", err)
	} else {
		fmt.Println("bcrypt_format_valid=true")
	}

	newHash, err := auth.HashPassword(password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hash password: %v\n", err)
		os.Exit(1)
	}

	_, err = pool.Exec(ctx, `UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`, newHash, user.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "update password: %v\n", err)
		os.Exit(1)
	}

	updated, err := users.GetByEmail(ctx, user.Email)
	if err != nil {
		fmt.Fprintf(os.Stderr, "re-fetch user: %v\n", err)
		os.Exit(1)
	}

	if !auth.CheckPassword(password, updated.PasswordHash) {
		fmt.Fprintln(os.Stderr, "verification failed after update")
		os.Exit(1)
	}

	fmt.Printf("password_reset_ok=true email=%s password=%s\n", user.Email, password)
}

func envOrDefault(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func safePrefix(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
