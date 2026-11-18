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
	migrateServicios(db)
	migrateIndexes(db)
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
	// Tabla nueva: registro de accesos
	if !m.HasTable("registro") {
		if err := m.CreateTable(&models.Registro{}); err != nil {
			db.Exec(`CREATE TABLE registro (
				id BIGSERIAL PRIMARY KEY, email TEXT, nombre TEXT, unidad TEXT,
				telefono TEXT, password TEXT, estado TEXT, rol TEXT,
				id_persona BIGINT, ip TEXT, creado_en TIMESTAMPTZ)`)
			db.Exec("CREATE INDEX IF NOT EXISTS idx_registro_email ON registro(email)")
		}
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
	if !m.HasColumn("recibo", "fecha_corte") {
		if err := m.AddColumn(&models.Recibo{}, "FechaCorte"); err != nil {
			db.Exec("ALTER TABLE recibo ADD COLUMN fecha_corte DATE")
		}
	}
	if !m.HasColumn("recibo", "total_con_interes") {
		if err := m.AddColumn(&models.Recibo{}, "TotalConInteres"); err != nil {
			db.Exec("ALTER TABLE recibo ADD COLUMN total_con_interes DOUBLE PRECISION DEFAULT 0")
		}
	}
	if !m.HasColumn("recibo", "fecha_pago") {
		if err := m.AddColumn(&models.Recibo{}, "FechaPago"); err != nil {
			db.Exec("ALTER TABLE recibo ADD COLUMN fecha_pago DATE")
		}
	}
	if !m.HasColumn("recibo", "monto_pagado") {
		if err := m.AddColumn(&models.Recibo{}, "MontoPagado"); err != nil {
			db.Exec("ALTER TABLE recibo ADD COLUMN monto_pagado DOUBLE PRECISION DEFAULT 0")
		}
	}
	if !m.HasColumn("recibo", "pago_parcial") {
		if err := m.AddColumn(&models.Recibo{}, "PagoParcial"); err != nil {
			db.Exec("ALTER TABLE recibo ADD COLUMN pago_parcial BOOLEAN DEFAULT FALSE")
		}
	}
	if !m.HasColumn("recibo", "id_recibo_origen") {
		if err := m.AddColumn(&models.Recibo{}, "IDReciboOrigen"); err != nil {
			db.Exec("ALTER TABLE recibo ADD COLUMN id_recibo_origen BIGINT")
		}
	}
	if !m.HasColumn("recibo", "saldo_favor") {
		if err := m.AddColumn(&models.Recibo{}, "SaldoFavor"); err != nil {
			db.Exec("ALTER TABLE recibo ADD COLUMN saldo_favor DOUBLE PRECISION DEFAULT 0")
		}
	}
	if !m.HasColumn("users", "id_persona") {
		if err := m.AddColumn(&models.User{}, "IDPersona"); err != nil {
			db.Exec("ALTER TABLE users ADD COLUMN id_persona BIGINT")
		}
	}
}

// migrateIndexes crea los índices usados por las consultas de cuenta y deudas.
func migrateIndexes(db *gorm.DB) {
	for _, stmt := range []string{
		"CREATE INDEX IF NOT EXISTS idx_recibo_id_persona ON recibo(id_persona)",
		"CREATE INDEX IF NOT EXISTS idx_recibo_id_departamento ON recibo(id_departamento)",
		"CREATE INDEX IF NOT EXISTS idx_recibo_periodo ON recibo(periodo)",
		"CREATE INDEX IF NOT EXISTS idx_recibo_tipo ON recibo(tipo)",
		"CREATE INDEX IF NOT EXISTS idx_departamento_id_dueno ON departamento(id_dueño)",
		"CREATE INDEX IF NOT EXISTS idx_departamento_id_inquilino ON departamento(id_inquilino)",
		"CREATE INDEX IF NOT EXISTS idx_detallerecibo_id_recibo ON detallerecibo(id_recibo)",
	} {
		db.Exec(stmt)
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

// migrateServicios unifica los servicios viejos ("Pago Final"/"Pago Parcial")
// a los nombres base usados por la generación de deuda.
func migrateServicios(db *gorm.DB) {
	pairs := [][2]string{
		{"Expensas Comunes. Pago Final", "Expensas Comunes"},
		{"Expensas Comunes. Pago Parcial", "Expensas Comunes"},
		{"Servicios. Pago Final", "Servicios"},
		{"Servicios. Pago Parcial", "Servicios"},
		{"Expensas Extraordinarias. Pago Final", "Expensas Extraordinarias"},
		{"Expensas Extraordinarias. Pago Parcial", "Expensas Extraordinarias"},
		{"Deudas Anteriores. Pago", "Deudas Anteriores"},
		{"Luz (EDET)", "Servicios"},
	}
	for _, p := range pairs {
		var old models.Servicio
		if db.Where("nombre = ?", p[0]).First(&old).Error != nil {
			continue
		}
		var target models.Servicio
		if db.Where("nombre = ?", p[1]).First(&target).Error == nil {
			db.Model(&models.DetalleRecibo{}).Where("id_servicio = ?", old.ID).Update("id_servicio", target.ID)
			db.Delete(&old)
		} else {
			db.Model(&old).Update("nombre", p[1])
		}
	}
}
