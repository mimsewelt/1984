package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	dbURL := mustEnv("DATABASE_URL")
	direction := "up"
	if len(os.Args) > 1 {
		direction = os.Args[1]
	}

	m, err := migrate.New("file://migrations", dbURL)
	if err != nil {
		fmt.Printf("migrate init error: %v\n", err)
		os.Exit(1)
	}
	defer m.Close()

	switch direction {
	case "up":
		err = m.Up()
	case "down":
		err = m.Down()
	case "drop":
		err = m.Drop()
	default:
		fmt.Printf("unknown direction: %s (use up/down/drop)\n", direction)
		os.Exit(1)
	}

	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		fmt.Printf("migration error: %v\n", err)
		os.Exit(1)
	}

	v, dirty, _ := m.Version()
	fmt.Printf("migration %s done — version: %d dirty: %v\n", direction, v, dirty)
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("required env var not set: " + key)
	}
	return v
}
