package handlers

import (
	"fmt"
	"strings"
	"time"

	"recibosgo/internal/models"
)

// Orden natural por número legacy y nombre; compatible con PostgreSQL.
const ordenUnidades = "CAST(NULLIF(regexp_replace(numero, '[^0-9]', '', 'g'), '') AS INTEGER) NULLS LAST, nombre ASC"

func FmtMoney(v float64) string {
	return fmt.Sprintf("%.2f", v)
}

func FmtFecha(t time.Time) string {
	return t.Format("02/01/2006")
}

func FmtFechaDate(f models.Fecha) string {
	return f.String()
}

func TrimNumero(s string) string {
	return strings.TrimSpace(s)
}
