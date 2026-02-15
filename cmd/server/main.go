package main

import (
	"fmt"
	"net/http"

	"github.com/vf0429/Petwell_Backend/internal/config"
	"github.com/vf0429/Petwell_Backend/internal/handlers"
	"github.com/vf0429/Petwell_Backend/internal/services/places"
	"github.com/vf0429/Petwell_Backend/internal/services/rag"
)

const port = "8000"

func main() {
	// Load Configuration
	cfg := config.LoadConfig()
	ragClient := rag.NewClient(cfg)

	// Core handlers
	http.HandleFunc("/vaccines", handlers.VaccinesHandler)
	http.HandleFunc("/register", handlers.RegisterHandler)
	http.HandleFunc("/posts", handlers.PostsHandler)
	http.HandleFunc("/clinics", handlers.NewClinicsHandler(cfg))
	http.HandleFunc("/emergency-clinics", handlers.NewEmergencyClinicsHandler(cfg))

	// Insurance handlers
	http.HandleFunc("/insurance-companies", handlers.InsuranceCompaniesHandler)
	http.HandleFunc("/insurance-products", handlers.InsuranceProductsHandler)
	http.HandleFunc("/coverage-list", handlers.CoverageListHandler)
	http.HandleFunc("/coverage-limits", handlers.CoverageLimitsHandler)
	http.HandleFunc("/sub-coverage-limits", handlers.SubCoverageLimitsHandler)

	// Legacy handlers
	http.HandleFunc("/insurance-providers", handlers.InsuranceProvidersHandler)
	http.HandleFunc("/service-subcategories", handlers.ServiceSubcategoriesHandler)

	// RAG chat handler
	http.HandleFunc("/api/chat", handlers.NewRAGHandler(ragClient))

	// Vets handler
	placesClient := places.NewClient(cfg.MapsAPIKey)
	http.HandleFunc("/api/vets", handlers.NewVetsHandler(placesClient))

	fmt.Printf("PetWell Backend running at http://localhost:%s\n", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		fmt.Printf("Server failed to start: %v\n", err)
	}
}
