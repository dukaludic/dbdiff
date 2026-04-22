package main

import (
	"flag"
	"fmt"
	"os"
)

type Config struct {
	Flavor   string
	Host     string
	Port     uint16
	User     string
	Password string
	Schema   string
}

func readArguments() Config {
	flavor := flag.String("flavor", "mysql", "database flavor (mysql, mariadb)")
	host := flag.String("host", "127.0.0.1", "database host")
	port := flag.Int("port", 3306, "database port")
	user := flag.String("user", "root", "database user")
	password := flag.String("password", "", "database password")
	schema := flag.String("schema", "sakila", "database schema to watch")

	flag.Parse()

	if *schema == "" {
		fmt.Fprintln(os.Stderr, "error: -schema is required")
		os.Exit(1)
	}

	return Config{
		Flavor:   *flavor,
		Host:     *host,
		Port:     uint16(*port),
		User:     *user,
		Password: *password,
		Schema:   *schema,
	}
}
