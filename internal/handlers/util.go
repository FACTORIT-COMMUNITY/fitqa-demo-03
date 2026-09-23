package handlers

import "strings"

func boolAInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// esViolacionUnicidad detecta el error de constraint UNIQUE que devuelve modernc.org/sqlite.
func esViolacionUnicidad(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// esViolacionFK detecta el error de FK que devuelve modernc.org/sqlite.
func esViolacionFK(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "FOREIGN KEY constraint failed")
}
