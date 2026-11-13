package pdf

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/go-pdf/fpdf"

	"recibosgo/internal/models"
)

// Recibo en hoja A4 vertical (210x297mm) que contiene el recibo DOS veces
// apiladas: original + copia (como el sistema PHP original).
// Cada recibo es un A5 horizontal (210x148mm).

const (
	pw = 210.0 // ancho A4/ A5 landscape (mm)
	ph = 148.0 // alto de cada recibo (A5 landscape, mm)
)

func Recibo(rec models.Recibo) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(0, 0, 0)
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddPage()

	renderRecibo(pdf, 0.0, rec) // original
	renderRecibo(pdf, ph, rec)  // copia

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// logoDevuelve la ruta del logo si existe. FPDF no soporta PNG con
// transparencia; acepta PNG-RGB o JPEG (el JPEG siempre anda).
func logoPath() string {
	cands := []string{
		"static/images/logo.png",
		"static/images/logo.jpg",
		"logo.png",
		"logo.jpg",
	}
	for _, p := range cands {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func renderRecibo(pdf *fpdf.Fpdf, y0 float64, rec models.Recibo) {
	// --- cabecera ---
	pdf.SetFillColor(190, 190, 190)
	pdf.Rect(0, y0, pw, 25, "F")
	if lp := logoPath(); lp != "" {
		pdf.Image(lp, 10, y0+3, 60, 18, false, "", 0, "")
	}

	cliente := ""
	if rec.Titular != nil {
		cliente = rec.Titular.Nombre
	} else if rec.Departamento.Dueño.ID != 0 {
		cliente = rec.Departamento.Dueño.Nombre
	}

	pdf.SetFont("Arial", "B", 13)
	pdf.SetTextColor(0, 0, 0)
	pdf.SetY(y0 + 5)
	pdf.SetX(100)
	pdf.CellFormat(0, 5, "Recibo nro: "+rec.NumeroRecibo, "", 0, "L", false, 0, "")
	pdf.SetY(y0 + 10)
	pdf.SetX(100)
	pdf.CellFormat(0, 5, "Cliente : "+cliente, "", 0, "L", false, 0, "")
	pdf.SetY(y0 + 15)
	pdf.SetX(100)
	pdf.CellFormat(0, 5, "Fecha de emision : "+rec.Fecha.Time.Format("02/01/2006"), "", 0, "L", false, 0, "")

	// --- títulos de la tabla ---
	pdf.SetY(y0 + 28)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFillColor(79, 78, 77)
	pdf.CellFormat(30, 6, "Codigo", "", 0, "C", true, 0, "")
	pdf.CellFormat(110, 6, "Descripcion", "", 0, "L", true, 0, "")
	pdf.CellFormat(20, 6, "Pago", "", 0, "C", true, 0, "")
	pdf.CellFormat(30, 6, "Importe", "", 0, "R", true, 0, "")

	// --- detalle ---
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFillColor(255, 255, 255)
	pdf.SetFont("times", "", 12)
	pdf.SetLineWidth(0.2)

	const rowH = 6.0
	total := 0.0
	for i, d := range rec.Detalles {
		// máximo 16 filas por recibo para que quepa en los 148mm
		if i >= 16 {
			break
		}
		y := y0 + 35 + float64(i)*rowH
		concepto := d.Servicio.Nombre
		if d.Descripcion != nil && *d.Descripcion != "" {
			concepto += ". " + *d.Descripcion
		}
		importe := d.Precio * float64(d.Cantidad)
		pago := "Transf."
		if d.FormaPago == "E" {
			pago = "Efect."
		}
		pdf.SetY(y)
		pdf.SetX(10)
		pdf.CellFormat(30, rowH, fmt.Sprintf("%d", d.Servicio.ID), "1", 0, "C", true, 0, "")
		pdf.CellFormat(110, rowH, concepto, "1", 0, "L", true, 0, "")
		pdf.CellFormat(20, rowH, pago, "1", 0, "R", true, 0, "")
		pdf.CellFormat(30, rowH, "$"+arMoney(importe), "1", 0, "R", true, 0, "")
		total += importe
	}

	// --- total ---
	y := y0 + 35 + rowH*16
	pdf.SetY(y)
	pdf.SetX(10)
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(190, rowH+2, "Importe Total : $"+arMoney(total), "1", 0, "R", true, 0, "")

	// --- pie / sello ---
	pdf.SetFillColor(60, 60, 60)
	pdf.Rect(0, y0+ph-8, pw, 8, "F")
	pdf.SetFont("Arial", "", 12)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetY(y0 + ph - 7)
	pdf.SetX(120)
	pdf.CellFormat(0, 5, "Recibido. Sello", "", 0, "L", false, 0, "")
}

func arMoney(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	neg := ""
	if strings.HasPrefix(s, "-") {
		neg = "-"
		s = s[1:]
	}
	parts := strings.Split(s, ".")
	var b strings.Builder
	for i, c := range parts[0] {
		if i > 0 && (len(parts[0])-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	return neg + b.String() + "," + parts[1]
}
