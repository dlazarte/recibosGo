package handlers

import "recibosgo/internal/models"

// personaDelUsuario resuelve la persona del usuario: por FK si existe,
// si no por email (compatibilidad con usuarios creados antes del auto-registro).
func (a *App) personaDelUsuario(user *models.User) (models.Persona, bool) {
	if user == nil {
		return models.Persona{}, false
	}
	var p models.Persona
	if user.IDPersona != nil && *user.IDPersona != 0 {
		if a.DB.First(&p, *user.IDPersona).Error == nil {
			return p, true
		}
	}
	if a.DB.Where("email = ?", user.Email).First(&p).Error == nil {
		return p, true
	}
	return models.Persona{}, false
}
