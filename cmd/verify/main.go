package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"

	"recibosgo/internal/config"
	"recibosgo/internal/database"
	"recibosgo/internal/models"
)

func main() {
	cfg := config.Load()
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("error conectando: %v", err)
	}

	var deptos []models.Departamento
	db.Preload("Dueño").Preload("Inquilino").Order("nombre ASC").Find(&deptos)

	f, err := os.Create("verificacion_deptos.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// BOM for Excel
	f.Write([]byte{0xEF, 0xBB, 0xBF})

	w := csv.NewWriter(f)
	defer w.Flush()

	w.Write([]string{"ID", "Nombre (Unidad)", "ID Dueño", "Dueño", "ID Inquilino", "Inquilino", "Activo"})

	for _, d := range deptos {
		dueño := ""
		var idDueño string
		if d.Dueño.ID != 0 {
			dueño = d.Dueño.Nombre
			idDueño = fmt.Sprintf("%d", d.Dueño.ID)
		}

		inquilino := ""
		var idInq string
		if d.Inquilino != nil && d.Inquilino.ID != 0 {
			inquilino = d.Inquilino.Nombre
			idInq = fmt.Sprintf("%d", d.Inquilino.ID)
		}

		activo := "NO"
		if d.Activo {
			activo = "SI"
		}

		w.Write([]string{
			fmt.Sprintf("%d", d.ID),
			d.Nombre,
			idDueño,
			dueño,
			idInq,
			inquilino,
			activo,
		})
	}

	fmt.Printf("Exportados %d departamentos a verificacion_deptos.csv\n", len(deptos))
}
