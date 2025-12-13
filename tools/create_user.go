package main

import (
	"fmt"
	"log"
	"os"

	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/common/config"
	"github.com/sunthewhat/easy-cert-api/common/gorm"
	"github.com/sunthewhat/easy-cert-api/type/shared/model"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	if len(os.Args) < 5 {
		fmt.Println("Usage: go run create_user.go <username> <password> <firstname> <lastname>")
		fmt.Println("Example: go run create_user.go admin admin123 Admin User")
		os.Exit(1)
	}

	username := os.Args[1]
	password := os.Args[2]
	firstname := os.Args[3]
	lastname := os.Args[4]

	// Load config and connect to database
	config.LoadConfig()
	gorm.InitGorm()

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Failed to hash password:", err)
	}

	// Create user
	user := &model.User{
		Username:  username,
		Password:  string(hashedPassword),
		Firstname: firstname,
		Lastname:  lastname,
	}

	err = common.Gorm.User.Create(user)
	if err != nil {
		log.Fatal("Failed to create user:", err)
	}

	fmt.Printf("✓ User created successfully!\n")
	fmt.Printf("  Username: %s\n", username)
	fmt.Printf("  Firstname: %s\n", firstname)
	fmt.Printf("  Lastname: %s\n", lastname)
	fmt.Printf("\nYou can now login with:\n")
	fmt.Printf("  Username: %s\n", username)
	fmt.Printf("  Password: %s\n", password)
}
