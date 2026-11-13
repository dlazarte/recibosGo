package models

import (
	"database/sql/driver"
	"fmt"
	"time"
)

type User struct {
	ID       uint   `gorm:"primaryKey"`
	Email    string `gorm:"uniqueIndex;not null"`
	Password string
	Role     string // administrador, dueño, inquilino, receptor
}

func (User) TableName() string { return "users" }

type Departamento struct {
	ID          uint    `gorm:"primaryKey"`
	Nombre      string  `gorm:"not null"` // unidad legible: 1A, SS2A, PB C
	Numero      string  `gorm:"not null"` // legacy, se conserva durante migración
	Piso        *string // legacy
	Propietario string  `gorm:"not null"` // legacy, reemplazado por persona
	Telefono    string  // legacy
	Email       string  // legacy
	IDDueño     uint    // FK→persona, NOT NULL
	IDInquilino *uint   // FK→persona, NULLABLE
	Coeficiente float64 `gorm:"default:0"` // % de participación para prorratear expensas (0-100)
	Activo      bool
	Dueño       Persona  `gorm:"foreignKey:IDDueño"`
	Inquilino   *Persona `gorm:"foreignKey:IDInquilino"`
}

func (Departamento) TableName() string { return "departamento" }

type Persona struct {
	ID       uint   `gorm:"primaryKey"`
	Nombre   string `gorm:"not null"`
	Telefono string
	Email    string
	Tipo     string `gorm:"not null"` // propietario | inquilino
	Activo   bool
}

func (Persona) TableName() string { return "persona" }

type Servicio struct {
	ID             uint `gorm:"primaryKey"`
	Nombre         string
	PrecioUnitario float64
	Activo         bool
}

func (Servicio) TableName() string { return "servicio" }

type Fecha struct {
	time.Time
}

var fechaLayouts = []string{
	"2006-01-02",
	time.RFC3339,
	"2006-01-02 15:04:05",
	"2006-01-02 15:04:05.999999999 -0700 MST",
}

func (f *Fecha) Scan(value interface{}) error {
	switch v := value.(type) {
	case nil:
		f.Time = time.Time{}
	case time.Time:
		f.Time = v
	case string:
		t, err := parseFecha(v)
		if err != nil {
			return err
		}
		f.Time = t
	case []byte:
		t, err := parseFecha(string(v))
		if err != nil {
			return err
		}
		f.Time = t
	default:
		return fmt.Errorf("unsupported fecha scan: %T", value)
	}
	return nil
}

func parseFecha(s string) (time.Time, error) {
	for _, layout := range fechaLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("fecha inválida: %q", s)
}

func (f Fecha) Value() (driver.Value, error) {
	if f.Time.IsZero() {
		return nil, nil
	}
	return f.Time.Format("2006-01-02"), nil
}

func (f Fecha) GormDataType() string { return "date" }

func (f Fecha) String() string { return f.Time.Format("02/01/2006") }

type Recibo struct {
	ID             uint `gorm:"primaryKey"`
	NumeroRecibo   string
	Fecha          Fecha
	Periodo        string `gorm:"default:''"` // YYYY-MM de la deuda generada; vacío en recibos manuales
	IDDepartamento uint
	IDPersona      *uint // titular al generar (nullable, NULL en legacy)
	Observaciones  *string
	Total          float64
	Departamento   Departamento    `gorm:"foreignKey:IDDepartamento"`
	Titular        *Persona        `gorm:"foreignKey:IDPersona"`
	Detalles       []DetalleRecibo `gorm:"foreignKey:IDRecibo"`
}

func (Recibo) TableName() string { return "recibo" }

type DetalleRecibo struct {
	ID          uint `gorm:"primaryKey"`
	IDRecibo    uint
	IDServicio  uint
	Precio      float64
	Cantidad    int
	Descripcion *string
	FormaPago   string   // 'E' efectivo, 'T' transferencia, ''
	Servicio    Servicio `gorm:"foreignKey:IDServicio"`
	Recibo      Recibo   `gorm:"foreignKey:IDRecibo"`
}

func (DetalleRecibo) TableName() string { return "detallerecibo" }

func (r Recibo) Pagado() bool {
	for _, d := range r.Detalles {
		if d.FormaPago == "" {
			return false
		}
	}
	return len(r.Detalles) > 0
}

func (r Recibo) FormaPago() string {
	metodos := map[string]bool{}
	for _, d := range r.Detalles {
		if d.FormaPago != "" {
			metodos[d.FormaPago] = true
		}
	}
	if metodos["E"] && metodos["T"] {
		return "Efectivo y Transferencia"
	}
	if metodos["E"] {
		return "Efectivo"
	}
	if metodos["T"] {
		return "Transferencia"
	}
	return ""
}
