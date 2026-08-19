// Command seed-admin bootstraps the first team account (role=admin) so the
// operation team has a way into /acesso-admin — team accounts otherwise are only
// created by an existing admin from within the admin panel.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"p5wellness/backend/internal/config"
	"p5wellness/backend/internal/db"
	"p5wellness/backend/internal/domain/auth"
	"p5wellness/backend/internal/repository/postgres"
)

func main() {
	_ = godotenv.Load()

	name := flag.String("name", "", "nome do administrador")
	email := flag.String("email", "", "e-mail de login")
	password := flag.String("password", "", "senha inicial (troque após o primeiro acesso)")
	flag.Parse()

	if *name == "" || *email == "" || *password == "" {
		fmt.Println("uso: seed-admin -name \"Nome\" -email admin@p5wellness.com.br -password \"senha-forte\"")
		os.Exit(1)
	}
	if len(*password) < 8 {
		log.Fatal("senha deve ter ao menos 8 caracteres")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	hash, err := auth.HashPassword(*password, cfg.PasswordPepper)
	if err != nil {
		log.Fatalf("hash: %v", err)
	}

	team := postgres.NewTeamRepository(pool)
	member, err := team.Create(ctx, postgres.CreateTeamMemberParams{
		Name:         *name,
		Email:        *email,
		PasswordHash: hash,
		Role:         "admin",
	})
	if err != nil {
		log.Fatalf("create admin: %v", err)
	}

	fmt.Printf("Administrador criado: %s <%s> (id=%s)\n", member.Name, member.Email, member.ID)
}
