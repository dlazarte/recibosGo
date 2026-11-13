package database

import (
	"log"

	"gorm.io/gorm"

	"recibosgo/internal/models"
	"recibosgo/internal/parser"
)

// BackfillV21 populates new v2.1 fields from CSV and legacy data.
// Idempotent: safe to run multiple times without duplicating data.
func BackfillV21(db *gorm.DB, csvPath string) {
	backfillFromCSV(db, csvPath)
	backfillDepartamentoNombre(db)
	log.Printf("backfill v2.1 completado")
}

// backfillFromCSV creates personas and links them to departamentos using the CSV.
func backfillFromCSV(db *gorm.DB, csvPath string) {
	rows, err := parser.ParseCSV(csvPath)
	if err != nil {
		log.Printf("backfill: no se pudo leer CSV (%s), saltando: %v", csvPath, err)
		return
	}

	for _, row := range rows {
		// Find the departamento by matching the unit name in Propietario legacy field
		var deptos []models.Departamento
		db.Where("propietario LIKE ?", "%"+row.Unidad+"%").Find(&deptos)
		if len(deptos) == 0 {
			// Try matching by nombre if already set
			db.Where("nombre = ?", row.Unidad).Find(&deptos)
		}
		if len(deptos) == 0 {
			log.Printf("backfill: departamento no encontrado para unidad %q", row.Unidad)
			continue
		}

		for _, d := range deptos {
			// Skip if already linked
			if d.IDDueño != 0 {
				continue
			}

			// Create or find propietario persona
			normName := parser.NormalizeName(row.Propietario)
			var persona models.Persona
			result := db.Where("nombre = ? AND tipo = ?", normName, "propietario").First(&persona)
			if result.Error == gorm.ErrRecordNotFound {
				persona = models.Persona{
					Nombre:   normName,
					Telefono: row.TelefonoProp,
					Tipo:     "propietario",
					Activo:   true,
				}
				if err := db.Create(&persona).Error; err != nil {
					log.Printf("backfill: error creando persona %q: %v", normName, err)
					continue
				}
			}

			// Link propietario
			db.Model(&d).Update("id_dueño", persona.ID)

			// Set nombre from CSV
			if d.Nombre == "" {
				db.Model(&d).Update("nombre", row.Unidad)
			}

			// Create or find inquilino if present
			if row.Inquilino != "" && row.Inquilino != "-" && row.Inquilino != "PROP" && row.Inquilino != "PROP." {
				inqNorm := parser.NormalizeName(row.Inquilino)
				var inq models.Persona
				result := db.Where("nombre = ? AND tipo = ?", inqNorm, "inquilino").First(&inq)
				if result.Error == gorm.ErrRecordNotFound {
					inq = models.Persona{
						Nombre:   inqNorm,
						Telefono: row.TelefonoInq,
						Tipo:     "inquilino",
						Activo:   true,
					}
					if err := db.Create(&inq).Error; err != nil {
						log.Printf("backfill: error creando inquilino %q: %v", inqNorm, err)
						continue
					}
				}
				db.Model(&d).Update("id_inquilino", inq.ID)
			}

			log.Printf("backfill: depto %s → dueño=%s, inquilino=%s", row.Unidad, normName, row.Inquilino)
		}
	}
}

// backfillDepartamentoNombre sets departamento.nombre from the legacy propietario field
// for departments not covered by the CSV.
func backfillDepartamentoNombre(db *gorm.DB) {
	var deptos []models.Departamento
	db.Where("nombre = '' OR nombre IS NULL").Find(&deptos)

	for _, d := range deptos {
		unidad, _ := parser.ParseNombre(d.Propietario)
		if unidad == "" {
			// Fallback: use the numero legacy field
			unidad = d.Numero
		}
		if unidad != "" {
			db.Model(&d).Update("nombre", unidad)
		}
	}
}
