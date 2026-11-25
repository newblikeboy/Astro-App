package platform

// gurujiWording selects display labels retained in calculation and legacy snapshot schemas.
func (l pdfLocale) gurujiWording(wording GurujiWording) string {
	if l.language == "en" {
		return wording.English
	}
	return wording.Hindi
}
