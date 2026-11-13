package parser

import "testing"

func TestParseNombre(t *testing.T) {
	tests := []struct {
		input   string
		unidad  string
		nombre  string
	}{
		// Rule 1: suffix *XX*
		{"1C  DR CARAN *1C*", "1C", "DR CARAN"},
		{"3G CECILIA GUTIRREZ  *3G*(silvana)", "3G", "CECILIA GUTIRREZ (silvana)"},
		{"8A VERO ANGELILLI *8A*", "8A", "VERO ANGELILLI"},

		// Rule 2: prefix digit+letter
		{"1A DONADIO", "1A", "DONADIO"},
		{"2B TORRES RICARDO", "2B", "TORRES RICARDO"},
		{"10A CECILIA GUTIERRE", "10A", "CECILIA GUTIERRE"},

		// Rule 3: prefix SS
		{"SS2A", "SS2A", ""},
		{"ss1f  gettar", "SS1F", "GETTAR"},
		{"SS 2 D ALDERETE", "SS2D", "ALDERETE"},
		{"SS1J -RAMONA DEL CARMEN ORTIZ", "SS1J", "RAMONA DEL CARMEN ORTIZ"},

		// Rule 4: prefix PB
		{"PB C ALDERETE", "PB C", "ALDERETE"},
		{"PB D AGOTE PABLO", "PB D", "AGOTE PABLO"},

		// Rule 5: no unit
		{"ILIAL CONSULTORIO", "", "ILIAL CONSULTORIO"},
		{"GRANDI", "", "GRANDI"},

		// Edge cases
		{"", "", ""},
		{"DONADIO", "", "DONADIO"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			unidad, nombre := ParseNombre(tt.input)
			if unidad != tt.unidad {
				t.Errorf("unidad: got %q, want %q", unidad, tt.unidad)
			}
			if nombre != tt.nombre {
				t.Errorf("nombre: got %q, want %q", nombre, tt.nombre)
			}
		})
	}
}

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"DONADIO", "DONADIO"},
		{"  donadio  ", "DONADIO"},
		{"DR CARAN", "DR CARAN"},
		{"VERO *ANGELILLI*", "VERO ANGELILLI"},
		{"SILVANA,", "SILVANA"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeName(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
