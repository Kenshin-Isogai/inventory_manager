package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"backend/internal/config"
	"backend/internal/platform/database"
	"backend/internal/platform/migrate"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: go run ./cmd/migrate [up|status]")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx := context.Background()
	db, err := database.Open(ctx, cfg.Database)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	runner := migrate.NewRunner(db)

	switch os.Args[1] {
	case "up":
		count, err := runner.Up(ctx)
		if err != nil {
			log.Fatalf("apply migrations: %v", err)
		}
		fmt.Printf("applied %d migration(s)\n", count)
	case "status":
		status, err := runner.Status(ctx)
		if err != nil {
			log.Fatalf("migration status: %v", err)
		}
		for _, row := range status {
			state := "pending"
			if row.Applied {
				state = "applied"
			}
			fmt.Printf("%s\t%s\n", state, row.Name)
		}
	default:
		log.Fatalf("unknown command: %s", os.Args[1])
	}
}
