package database

import (
	"database/sql"
	"log"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"

	"recibosgo/internal/config"
	"recibosgo/internal/models"
)

func Connect(cfg config.Config) (*gorm.DB, error) {
	connConfig, err := pgx.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	connector := stdlib.GetConnector(*connConfig)
	sqlDB := sql.OpenDB(connector)
	dialector := postgres.New(postgres.Config{Conn: sqlDB})

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&models.User{}); err != nil {
		return nil, err
	}

	if !db.Migrator().HasTable("departamento") {
		if err := db.AutoMigrate(
			&models.Departamento{},
			&models.Servicio{},
			&models.Recibo{},
			&models.DetalleRecibo{},
			&models.ConsumoLuz{},
		); err != nil {
			return nil, err
		}
	}

	// v2.1 — migración aditiva (idempotente)
	migrateV21(db)
	BackfillV21(db, cfg.CSVPath)

	seedAdmin(db, cfg)
	return db, nil
}

func migrateV21(db *gorm.DB) {
	m := db.Migrator()

	// Tabla nueva: persona
	if !m.HasTable("persona") {
		m.CreateTable(&models.Persona{})
	}

	// Columnas nuevas en departamento (fallback SQL si AddColumn falla)
	if !m.HasColumn("departamento", "nombre") {
		if err := m.AddColumn(&models.Departamento{}, "Nombre"); err != nil {
			db.Exec("ALTER TABLE departamento ADD COLUMN nombre VARCHAR NOT NULL DEFAULT ''")
		}
	}
	if !m.HasColumn("departamento", "id_dueño") {
		if err := m.AddColumn(&models.Departamento{}, "IDDueño"); err != nil {
			db.Exec("ALTER TABLE departamento ADD COLUMN id_dueño INTEGER")
		}
	}
	if !m.HasColumn("departamento", "id_inquilino") {
		if err := m.AddColumn(&models.Departamento{}, "IDInquilino"); err != nil {
			db.Exec("ALTER TABLE departamento ADD COLUMN id_inquilino INTEGER")
		}
	}
	if !m.HasColumn("departamento", "coeficiente") {
		if err := m.AddColumn(&models.Departamento{}, "Coeficiente"); err != nil {
			db.Exec("ALTER TABLE departamento ADD COLUMN coeficiente REAL DEFAULT 0")
		}
		db.Exec("UPDATE departamento SET coeficiente = 0 WHERE coeficiente IS NULL")
	}

	// Columna nueva en recibo
	if !m.HasColumn("recibo", "id_persona") {
		if err := m.AddColumn(&models.Recibo{}, "IDPersona"); err != nil {
			db.Exec("ALTER TABLE recibo ADD COLUMN id_persona INTEGER")
		}
	}
	if !m.HasColumn("recibo", "periodo") {
		if err := m.AddColumn(&models.Recibo{}, "Periodo"); err != nil {
			db.Exec("ALTER TABLE recibo ADD COLUMN periodo VARCHAR DEFAULT ''")
		}
		db.Exec("UPDATE recibo SET periodo = '' WHERE periodo IS NULL")
	}
	if !m.HasColumn("recibo", "tipo") {
		if err := m.AddColumn(&models.Recibo{}, "Tipo"); err != nil {
			db.Exec("ALTER TABLE recibo ADD COLUMN tipo VARCHAR DEFAULT ''")
		}
		db.Exec("UPDATE recibo SET tipo = '' WHERE tipo IS NULL")
	}
	if !m.HasTable("consumo_luz") {
		if err := m.CreateTable(&models.ConsumoLuz{}); err != nil {
			log.Printf("error creando consumo_luz: %v", err)
		}
	}
}

func seedAdmin(db *gorm.DB, cfg config.Config) {
	var count int64
	db.Model(&models.User{}).Count(&count)
	if count > 0 {
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("error hashing admin password: %v", err)
	}
	admin := models.User{Email: cfg.AdminEmail, Password: string(hash), Role: "administrador"}
	if err := db.Create(&admin).Error; err != nil {
		log.Fatalf("error seeding admin user: %v", err)
	}
	log.Printf("usuario administrador creado: %s", cfg.AdminEmail)
}
