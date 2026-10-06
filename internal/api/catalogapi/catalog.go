// Package catalogapi — каталог услуг: GET /api/services, без авторизации.
package catalogapi

import (
	"booking/internal/domain/service"
	"booking/internal/httpx"
	"booking/internal/usecase"
	"net/http"
)

type Module struct {
	catalog *usecase.Catalog
}

func New(catalog *usecase.Catalog) *Module {
	return &Module{catalog: catalog}
}

func (m *Module) Register(mux *http.ServeMux, _ httpx.Middlewares) {
	mux.HandleFunc("GET /api/services", m.list)
}

type serviceResponse struct {
	ID     string       `json:"id"`
	Name   string       `json:"name"`
	Kind   service.Kind `json:"kind"`
	Active bool         `json:"active"`
}

func (m *Module) list(w http.ResponseWriter, r *http.Request) {
	services, err := m.catalog.Active(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	out := make([]serviceResponse, 0, len(services))
	for _, s := range services {
		out = append(out, serviceResponse{ID: s.ID, Name: s.Name, Kind: s.Kind, Active: s.Active})
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}
