package platform

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strconv"
)

type LocalizedPDFRenderer interface {
	RenderLanguage(context.Context, *Order, string) ([]byte, error)
}

// downloadRemedyPDF applies current language and wording to the immutable purchased snapshot, without
// replacing its original Hindi file or creating another order. One bounded render
// can run at a time; entitlement is checked again after the potentially slow render.
func (a *App) downloadRemedyPDF(w http.ResponseWriter, r *http.Request, purchased *Order, userID, language string) {
	renderer, ok := a.PDF.(LocalizedPDFRenderer)
	if !ok || a.remedyPDFSlots == nil || purchased == nil || purchased.Snapshot == nil {
		fail(w, 503, fmt.Errorf("The remedy PDF could not be prepared. Please try again."))
		return
	}
	if r.Method == http.MethodHead {
		w.Header().Set("Content-Type", "application/pdf")
		return
	}
	select {
	case a.remedyPDFSlots <- struct{}{}:
		defer func() { <-a.remedyPDFSlots }()
	default:
		fail(w, 429, fmt.Errorf("Remedy PDF is busy. Please try again shortly."))
		return
	}
	data, err := renderer.RenderLanguage(r.Context(), purchased, language)
	if r.Context().Err() != nil {
		return
	}
	if err != nil || !bytes.HasPrefix(data, []byte("%PDF-")) {
		fail(w, 503, fmt.Errorf("The remedy PDF could not be prepared. Please try again."))
		return
	}
	if err = a.Store.View(r.Context(), func(s *State) error {
		o := s.Orders[purchased.ID]
		if o == nil || o.UserID != userID || o.Status != "paid" || o.Product != "pdf" {
			return fmt.Errorf("PDF not found")
		}
		return nil
	}); err != nil {
		fail(w, 404, err)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="Mishra4You-remedies-%s.pdf"`, language))
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	_, _ = w.Write(data)
}
