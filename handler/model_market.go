package handler

import (
	"encoding/json"
	"net/http"

	"github.com/tigerowo/infinite-canvas/model"
	"github.com/tigerowo/infinite-canvas/service"
)

func ModelMarket(w http.ResponseWriter, r *http.Request) { items, err := service.ListMarketModels(r.URL.Query().Get("category"), r.URL.Query().Get("featured") == "true"); if err != nil { FailError(w, err); return }; OK(w, items) }
func AdminModelProviders(w http.ResponseWriter, r *http.Request) { items, err := service.AdminModelProviders(); if err != nil { FailError(w, err); return }; OK(w, items) }
func AdminTestModelProvider(w http.ResponseWriter, r *http.Request, id string) { item, err := service.TestModelProviderConnection(id); if err != nil { FailError(w, err); return }; OK(w, item) }
func AdminModelRoutes(w http.ResponseWriter, r *http.Request) { items, err := service.AdminModelRoutes(); if err != nil { FailError(w, err); return }; OK(w, items) }
func AdminModelVariants(w http.ResponseWriter, r *http.Request) { items, err := service.AdminModelVariants(); if err != nil { FailError(w, err); return }; OK(w, items) }
func AdminMarketModels(w http.ResponseWriter, r *http.Request) { items, err := service.AdminMarketModels(); if err != nil { FailError(w, err); return }; OK(w, items) }
func AdminModelReadiness(w http.ResponseWriter, r *http.Request) { item, err := service.AdminModelReadiness(); if err != nil { FailError(w, err); return }; OK(w, item) }
func AdminSaveModelProvider(w http.ResponseWriter, r *http.Request) { var item model.ModelProvider; _ = json.NewDecoder(r.Body).Decode(&item); saved, err := service.SaveModelProvider(item); if err != nil { FailError(w, err); return }; saved.APIKey = ""; OK(w, saved) }
func AdminSaveMarketModel(w http.ResponseWriter, r *http.Request) { var item model.MarketModel; _ = json.NewDecoder(r.Body).Decode(&item); saved, err := service.SaveMarketModel(item); if err != nil { FailError(w, err); return }; OK(w, saved) }
func AdminSaveModelRoute(w http.ResponseWriter, r *http.Request) { var item model.ModelRoute; _ = json.NewDecoder(r.Body).Decode(&item); saved, err := service.SaveModelRoute(item); if err != nil { FailError(w, err); return }; OK(w, saved) }
func AdminSaveModelVariant(w http.ResponseWriter, r *http.Request) { var item model.ModelVariant; _ = json.NewDecoder(r.Body).Decode(&item); saved, err := service.SaveModelVariant(item); if err != nil { FailError(w, err); return }; OK(w, saved) }
