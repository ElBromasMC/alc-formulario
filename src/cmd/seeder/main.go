package main

import (
	"context"
	"log"
	"os"

	"alc/repository"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	adminPassword := os.Getenv("APP_ADMIN_PASSWORD")
	if adminPassword == "" {
		log.Fatalf("APP_ADMIN_PASSWORD environment variable required")
	}

	connStr := os.Getenv("POSTGRESQL_URL")
	dbpool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer dbpool.Close()

	repo := repository.New(dbpool)
	ctx := context.Background()

	// Hash the admin password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	// Define the admin user
	adminUserParams := repository.CreateAppUserParams{
		Name:           "Admin User",
		Email:          "admin@alc-ti.com",
		HashedPassword: string(hashedPassword),
		Role:           repository.UserRoleADMIN,
		Dni:            "00000000",
	}

	// Check if admin already exists
	_, err = repo.GetAppUserByEmail(ctx, adminUserParams.Email)
	if err == nil {
		log.Println("Admin user already exists.")
		return
	}

	// Create the user
	user, err := repo.CreateAppUser(ctx, adminUserParams)
	if err != nil {
		log.Fatalf("Failed to create admin user: %v", err)
	}

	log.Printf("Admin user created successfully! Email: %s, ID: %s\n", user.Email, user.UserID.Bytes)
}
