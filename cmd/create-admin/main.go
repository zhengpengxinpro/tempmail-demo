package main

import (
	"fmt"
	"os"
	"time"

	"tempmail/backend/internal/auth"
	"tempmail/backend/internal/config"
	"tempmail/backend/internal/domain"
	"tempmail/backend/internal/storage/memory"

	"github.com/google/uuid"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: create-admin <email> <password> <username> [super|admin]")
		os.Exit(1)
	}

	email := os.Args[1]
	password := os.Args[2]
	username := os.Args[3]
	roleStr := "admin"
	if len(os.Args) >= 5 {
		roleStr = os.Args[4]
	}

	var role domain.UserRole
	if roleStr == "super" {
		role = domain.RoleSuper
	} else {
		role = domain.RoleAdmin
	}

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 创建存储
	store := memory.NewStore(cfg.Mailbox.DefaultTTL)

	// 验证邮箱
	if !auth.ValidateEmail(email) {
		fmt.Println("Invalid email format")
		os.Exit(1)
	}

	// 验证密码
	if err := auth.ValidatePassword(password); err != nil {
		fmt.Printf("Invalid password: %v\n", err)
		os.Exit(1)
	}

	// 哈希密码
	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		fmt.Printf("Failed to hash password: %v\n", err)
		os.Exit(1)
	}

	// 创建管理员用户
	user := &domain.User{
		ID:              uuid.New().String(),
		Email:           email,
		Username:        username,
		PasswordHash:    hashedPassword,
		Role:            role,
		Tier:            domain.TierFree,
		IsActive:        true,
		IsEmailVerified: true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := store.CreateUser(user); err != nil {
		fmt.Printf("Failed to create user: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Admin user created successfully!\n")
	fmt.Printf("  ID:       %s\n", user.ID)
	fmt.Printf("  Email:    %s\n", user.Email)
	fmt.Printf("  Username: %s\n", user.Username)
	fmt.Printf("  Role:     %s\n", user.Role)
	fmt.Println("\nNote: This user exists only in memory. To test admin features,")
	fmt.Println("you need to manually modify the user role after registration in the running server.")
}
