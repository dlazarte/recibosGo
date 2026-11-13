package parser

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"unicode"
)

// DeptoCSV represents one row from dptos_dueños_inquilinos.csv
type DeptoCSV struct {
	Unidad     string
	Propietario string
	Inquilino  string
	TelefonoProp string
	TelefonoInq  string
}

// ParseCSV reads the CSV file and returns cleaned department data.
func ParseCSV(path string) ([]DeptoCSV, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("abriendo CSV: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("leyendo CSV: %w", err)
	}

	var result []DeptoCSV
	for i, row := range records {
		if i == 0 { // skip header
			continue
		}
		if len(row) < 2 {
			continue
		}
		depto := strings.TrimSpace(row[0])
		propietario := strings.TrimSpace(row[1])
		inquilino := ""
		if len(row) > 2 {
			inquilino = strings.TrimSpace(row[2])
		}
		telProp := ""
		if len(row) > 3 {
			telProp = strings.TrimSpace(row[3])
		}
		telInq := ""
		if len(row) > 4 {
			telInq = strings.TrimSpace(row[4])
		}

		if depto == "" || propietario == "" {
			continue
		}

		result = append(result, DeptoCSV{
			Unidad:      depto,
			Propietario: propietario,
			Inquilino:   inquilino,
			TelefonoProp: telProp,
			TelefonoInq:  telInq,
		})
	}
	return result, nil
}

// ParseNombre splits "departamento.propietario" legacy text into (unidad, nombre).
// Order of rules from migraciones.md:
//  1. Sufijo *XX* → unidad
//  2. Prefijo \d{1,2}[A-Ha-h] → unidad
//  3. Prefijo SS → unidad
//  4. Prefijo PB → unidad
//  5. Sin unidad → nombre = texto completo
func ParseNombre(text string) (unidad, nombre string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", ""
	}

	// Rule 1: suffix *XX* — find last pair of * delimiters
	if strings.Contains(text, "*") {
		lastStar := strings.LastIndex(text, "*")
		if lastStar > 0 {
			before := text[:lastStar]
			openStar := strings.LastIndex(before, "*")
			if openStar >= 0 {
				suffix := strings.TrimSpace(text[openStar+1 : lastStar])
				prefix := strings.TrimSpace(text[:openStar])
				// Strip trailing (silvana) or similar after closing *
				after := text[lastStar+1:]
				if len(after) > 0 && after[0] == '(' {
					endParen := strings.Index(after, ")")
					if endParen > 0 {
						prefix = prefix + " (" + after[1:endParen] + ")"
						prefix = strings.TrimSpace(prefix)
					}
				}
				if suffix != "" {
					// Strip redundant unit prefix from name (e.g. "1C  DR CARAN" → "DR CARAN")
					prefix = stripLeadingUnit(prefix, suffix)
					return suffix, prefix
				}
			}
		}
	}

	// Rule 2: prefix \d{1,2}[A-Ha-h]
	if len(text) >= 2 {
		r0 := rune(text[0])
		if r0 >= '0' && r0 <= '9' {
			end := 1
			if len(text) > 1 {
				r1 := rune(text[1])
				if r1 >= '0' && r1 <= '9' {
					end = 2
				}
			}
			if end < len(text) {
				rLetter := rune(text[end])
				if (rLetter >= 'A' && rLetter <= 'H') || (rLetter >= 'a' && rLetter <= 'h') {
					unidad = text[:end+1]
					nombre = strings.TrimSpace(text[end+1:])
					return strings.ToUpper(unidad), nombre
				}
			}
		}
	}

	// Rule 3: prefix SS
	upper := strings.ToUpper(text)
	if strings.HasPrefix(upper, "SS") {
		rest := strings.TrimLeft(text[2:], " \t")
		// SS + digit(s) + letter
		if len(rest) >= 2 {
			end := 0
			for end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
				end++
			}
			if end > 0 && end < len(rest) {
				// After digits, expect a letter (unit) or space+letter
				restEnd := rest[end]
				if restEnd == ' ' && end+1 < len(rest) && unicode.IsLetter(rune(rest[end+1])) {
					// "SS 2 D ALDERETE" → unit is SS + digits + next letter
					letter := strings.ToUpper(string(rest[end+1]))
					unidad = "SS" + rest[:end] + letter
					nombre = strings.TrimSpace(rest[end+2:])
					nombre = strings.TrimLeft(nombre, "- ")
					return unidad, strings.ToUpper(nombre)
				}
				if unicode.IsLetter(rune(restEnd)) {
					unidad = "SS" + rest[:end+1]
					nombre = strings.TrimSpace(rest[end+1:])
					nombre = strings.TrimLeft(nombre, "- ")
					return strings.ToUpper(unidad), strings.ToUpper(nombre)
				}
			}
		}
		// SS without clear unit pattern
		cleaned := strings.ToUpper(strings.ReplaceAll(text, " ", ""))
		return cleaned, ""
	}

	// Rule 4: prefix PB
	if strings.HasPrefix(upper, "PB") {
		rest := strings.TrimLeft(text[2:], " \t")
		if len(rest) > 0 {
			parts := strings.SplitN(rest, " ", 2)
			unitPart := strings.TrimSpace(parts[0])
			nombre = ""
			if len(parts) > 1 {
				nombre = strings.TrimSpace(parts[1])
			}
			if unitPart == "" {
				return "PB", nombre
			}
			return "PB " + strings.ToUpper(unitPart), nombre
		}
		return "PB", ""
	}

	// Rule 5: no unit found
	return "", text
}

// stripLeadingUnit removes a redundant unit prefix from the name.
// e.g. "1C  DR CARAN" with unit "1C" → "DR CARAN"
func stripLeadingUnit(name, unit string) string {
	upper := strings.ToUpper(name)
	u := strings.ToUpper(unit)
	if strings.HasPrefix(upper, u) {
		rest := strings.TrimSpace(name[len(u):])
		if rest != "" {
			return rest
		}
	}
	return name
}

// NormalizeName cleans a person's name for deduplication.
func NormalizeName(name string) string {
	name = strings.ToUpper(strings.TrimSpace(name))
	// remove common noise
	name = strings.ReplaceAll(name, "*", "")
	name = strings.ReplaceAll(name, ".", "")
	name = strings.ReplaceAll(name, ",", "")
	// collapse whitespace
	fields := strings.Fields(name)
	return strings.Join(fields, " ")
}
