package main

import (
	"log"
	"net/http"

	"recibosgo/internal/config"
	"recibosgo/internal/database"
	"recibosgo/internal/handlers"
	"recibosgo/internal/middleware"
	"recibosgo/internal/server"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("error conectando a la base: %v", err)
	}

	tmpls, err := server.LoadTemplates(".")
	if err != nil {
		log.Fatalf("error cargando templates: %v", err)
	}

	app := handlers.NewApp(db, tmpls, cfg.SessionSecret)
	auth := middleware.New(db, cfg.SessionSecret)
	r := server.NewRouter(app, auth, ".")

	addr := ":" + cfg.Port
	log.Printf("RecibosGo escuchando en http://localhost%s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}
