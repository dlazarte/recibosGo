package handlers

import "testing"

func TestCalcDeuda(t *testing.T) {
	// Datos reales de octubre 2026: 7.300.000 expensas y 1.183.000 SAT+CISI.
	exp, sat := calcDeuda(7300000, 1183000, 2.302566)
	if exp != 168087.32 {
		t.Fatalf("expensas = %v, quería 168087.32", exp)
	}
	if sat != 27239.36 {
		t.Fatalf("sat_cisi = %v, quería 27239.36", sat)
	}

	exp, sat = calcDeuda(100000, 50000, 0)
	if exp != 0 || sat != 0 {
		t.Fatalf("coeficiente 0 debe dar 0, dio %v/%v", exp, sat)
	}

	exp, _ = calcDeuda(100, 0, 33.333333)
	if exp != 33.33 {
		t.Fatalf("redondeo = %v, quería 33.33", exp)
	}
}
