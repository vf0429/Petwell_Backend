package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	port        = "8000"
	jsonFile    = "vaccines.json"
	clinicsFile = "clinics.csv"
	insuranceDB = "insurance.db"
)

// --- Models ---

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"` // "developer", "user"
}

type BlogPost struct {
	ID           string    `json:"id"`
	AuthorName   string    `json:"authorName"`
	AuthorAvatar string    `json:"authorAvatar"`
	Title        string    `json:"title"`
	Content      string    `json:"content"`    // Added content field
	ImageColor   string    `json:"imageColor"` // Hex or system name
	Likes        int       `json:"likes"`
	Timestamp    time.Time `json:"timestamp"`
}

// Clinic represents a veterinary clinic
type Clinic struct {
	ClinicID       string `json:"clinic_id"`
	Name           string `json:"name"`
	Address        string `json:"address"`
	PhoneRegular   string `json:"phone_regular"`
	PhoneEmergency string `json:"phone_emergency"`
	Whatsapp       string `json:"whatsapp"`
	OpeningHours   string `json:"opening_hours"`
	Emergency24h   string `json:"emergency_24h"`
	WebsiteURL     string `json:"website_url"`
	ApplemapURL    string `json:"applemap_url"`
	Latitude       string `json:"latitude"`
	Longitude      string `json:"longitude"`
	Rating         string `json:"rating"`
}

// InsuranceProvider represents a pet insurance provider/product
type InsuranceProvider struct {
	ProviderKey       string  `json:"provider_key"`
	InsuranceProvider string  `json:"insurance_provider"`
	CompanyName       string  `json:"company_name"`
	PlanName          string  `json:"plan_name"`
	Category          string  `json:"category"`
	Subcategory       string  `json:"subcategory"`
	CoveragePercent   string  `json:"coverage_percentage"`
	CancerCashHKD     float64 `json:"cancer_cash_hkd,omitempty"`
	CancerCashNotes   string  `json:"cancer_cash_notes,omitempty"`
	AdditionalBenefit float64 `json:"additional_critical_cash_benefit,omitempty"`
	CoverageMode      string  `json:"coverage_mode"`
}

// CoverageLimit represents a coverage limit entry
type CoverageLimit struct {
	ID                int    `json:"id"`
	LimitItem         string `json:"limit_item"`
	ProviderKey       string `json:"provider_key"`
	Level             string `json:"level"`
	Category          string `json:"category"`
	Subcategory       string `json:"subcategory"`
	CoverageAmountHKD string `json:"coverage_amount_hkd"`
	Notes             string `json:"notes"`
}

// ServiceSubcategory represents a service type
type ServiceSubcategory struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	DisplayOrder int    `json:"display_order"`
}

// --- In-Memory Storage ---

var (
	users      = make(map[string]User)
	posts      = []BlogPost{}
	postsMutex sync.RWMutex
	usersMutex sync.RWMutex
)

// --- Handlers ---

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	(*w).Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func vaccinesHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	if r.Method == http.MethodOptions {
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get the directory of the executable to find the json file reliably
	ex, err := os.Executable()
	if err != nil {
		http.Error(w, "Server Error", http.StatusInternalServerError)
		return
	}
	exPath := filepath.Dir(ex)

	// Try to open local file first
	file, err := os.Open(jsonFile)
	if err != nil {
		file, err = os.Open(filepath.Join(exPath, jsonFile))
		if err != nil {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
	}
	defer file.Close()

	w.Header().Set("Content-Type", "application/json")
	io.Copy(w, file)
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	if r.Method == http.MethodOptions {
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if user.ID == "" || user.Name == "" {
		http.Error(w, "ID and Name are required", http.StatusBadRequest)
		return
	}

	usersMutex.Lock()
	users[user.ID] = user
	usersMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
	fmt.Printf("Registered user: %+v\n", user)
}

func postsHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	if r.Method == http.MethodOptions {
		return
	}

	if r.Method == http.MethodGet {
		postsMutex.RLock()
		defer postsMutex.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(posts)
		return
	}

	if r.Method == http.MethodPost {
		var post BlogPost
		if err := json.NewDecoder(r.Body).Decode(&post); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Set defaults
		post.ID = fmt.Sprintf("%d", time.Now().UnixNano())
		post.Timestamp = time.Now()
		if post.ImageColor == "" {
			post.ImageColor = "blue"
		}

		postsMutex.Lock()
		// Prepend to keep newest first
		posts = append([]BlogPost{post}, posts...)
		postsMutex.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(post)
		fmt.Printf("New post created: %s by %s\n", post.Title, post.AuthorName)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// loadClinicsFromCSV reads the clinics.csv file and returns a slice of Clinic
func loadClinicsFromCSV() ([]Clinic, error) {
	file, err := os.Open(clinicsFile)
	if err != nil {
		ex, _ := os.Executable()
		exPath := filepath.Dir(ex)
		file, err = os.Open(filepath.Join(exPath, clinicsFile))
		if err != nil {
			return nil, err
		}
	}
	defer file.Close()

	reader := csv.NewReader(file)
	// Read header
	_, err = reader.Read()
	if err != nil {
		return nil, err
	}

	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var clinics []Clinic
	for _, row := range records {
		if len(row) < 13 {
			continue
		}
		clinics = append(clinics, Clinic{
			ClinicID:       row[0],
			Name:           row[1],
			Address:        row[2],
			PhoneRegular:   row[3],
			PhoneEmergency: row[4],
			Whatsapp:       row[5],
			OpeningHours:   row[6],
			Emergency24h:   row[7],
			WebsiteURL:     row[8],
			ApplemapURL:    row[9],
			Latitude:       row[10],
			Longitude:      row[11],
			Rating:         row[12],
		})
	}
	return clinics, nil
}

func clinicsHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	if r.Method == http.MethodOptions {
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clinics, err := loadClinicsFromCSV()
	if err != nil {
		fmt.Printf("Error reading clinics CSV: %v\n", err)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(clinics)
}

func emergencyClinicsHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	if r.Method == http.MethodOptions {
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	allClinics, err := loadClinicsFromCSV()
	if err != nil {
		fmt.Printf("Error reading clinics CSV: %v\n", err)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}

	var emergencyClinics []Clinic
	for _, c := range allClinics {
		if strings.ToUpper(strings.TrimSpace(c.Emergency24h)) == "TRUE" {
			emergencyClinics = append(emergencyClinics, c)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(emergencyClinics)
}

// --- Insurance API Handlers ---

func openInsuranceDB() (*sql.DB, error) {
	// Try to open local file first
	db, err := sql.Open("sqlite3", insuranceDB)
	if err != nil {
		ex, _ := os.Executable()
		exPath := filepath.Dir(ex)
		db, err = sql.Open("sqlite3", filepath.Join(exPath, insuranceDB))
		if err != nil {
			return nil, err
		}
	}
	return db, nil
}

func insuranceProvidersHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	if r.Method == http.MethodOptions {
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	db, err := openInsuranceDB()
	if err != nil {
		fmt.Printf("Error opening insurance DB: %v\n", err)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}
	defer db.Close()

	rows, err := db.Query(`SELECT provider_key, insurance_provider, company_name, plan_name, 
		category, subcategory, coverage_percentage, cancer_cash_hkd, cancer_cash_notes, 
		additional_critical_cash_benefit, coverage_mode 
		FROM pet_insurance_comparison ORDER BY provider_key`)
	if err != nil {
		fmt.Printf("Error querying insurance providers: %v\n", err)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}
	defer rows.Close()

	var providers []InsuranceProvider
	for rows.Next() {
		var p InsuranceProvider
		var cancerCash, additionalBenefit sql.NullFloat64
		var cancerNotes, coverageMode sql.NullString
		err := rows.Scan(&p.ProviderKey, &p.InsuranceProvider, &p.CompanyName, &p.PlanName,
			&p.Category, &p.Subcategory, &p.CoveragePercent, &cancerCash, &cancerNotes,
			&additionalBenefit, &coverageMode)
		if err != nil {
			continue
		}
		if cancerCash.Valid {
			p.CancerCashHKD = cancerCash.Float64
		}
		if cancerNotes.Valid {
			p.CancerCashNotes = cancerNotes.String
		}
		if additionalBenefit.Valid {
			p.AdditionalBenefit = additionalBenefit.Float64
		}
		if coverageMode.Valid {
			p.CoverageMode = coverageMode.String
		}
		providers = append(providers, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(providers)
}

func coverageLimitsHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	if r.Method == http.MethodOptions {
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	db, err := openInsuranceDB()
	if err != nil {
		fmt.Printf("Error opening insurance DB: %v\n", err)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}
	defer db.Close()

	// Check for query params (e.g., ?provider_key=5&level=Category)
	providerKey := r.URL.Query().Get("provider_key")
	level := r.URL.Query().Get("level")

	query := `SELECT id, limit_item, provider_key, level, category, subcategory, coverage_amount_hkd, notes 
		FROM coverage_limits WHERE 1=1`
	var args []interface{}

	if providerKey != "" {
		query += " AND provider_key = ?"
		args = append(args, providerKey)
	}
	if level != "" {
		query += " AND level = ?"
		args = append(args, level)
	}
	query += " ORDER BY provider_key, level, subcategory"

	rows, err := db.Query(query, args...)
	if err != nil {
		fmt.Printf("Error querying coverage limits: %v\n", err)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}
	defer rows.Close()

	var limits []CoverageLimit
	for rows.Next() {
		var cl CoverageLimit
		var notes sql.NullString
		err := rows.Scan(&cl.ID, &cl.LimitItem, &cl.ProviderKey, &cl.Level,
			&cl.Category, &cl.Subcategory, &cl.CoverageAmountHKD, &notes)
		if err != nil {
			continue
		}
		if notes.Valid {
			cl.Notes = notes.String
		}
		limits = append(limits, cl)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(limits)
}

func serviceSubcategoriesHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	if r.Method == http.MethodOptions {
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	db, err := openInsuranceDB()
	if err != nil {
		fmt.Printf("Error opening insurance DB: %v\n", err)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}
	defer db.Close()

	rows, err := db.Query(`SELECT id, name, display_order 
		FROM service_subcategories ORDER BY display_order`)
	if err != nil {
		fmt.Printf("Error querying service subcategories: %v\n", err)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}
	defer rows.Close()

	var subcategories []ServiceSubcategory
	for rows.Next() {
		var s ServiceSubcategory
		err := rows.Scan(&s.ID, &s.Name, &s.DisplayOrder)
		if err != nil {
			continue
		}
		subcategories = append(subcategories, s)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subcategories)
}

func main() {
	// Initialize some mock data
	posts = []BlogPost{
		{ID: "1", AuthorName: "System", Title: "Welcome to PetWell Blog", Content: "This is the start of our community.", Likes: 10, ImageColor: "green", Timestamp: time.Now()},
	}

	http.HandleFunc("/vaccines", vaccinesHandler)
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/posts", postsHandler)
	http.HandleFunc("/clinics", clinicsHandler)
	http.HandleFunc("/emergency-clinics", emergencyClinicsHandler)

	// Insurance API endpoints
	http.HandleFunc("/insurance-providers", insuranceProvidersHandler)
	http.HandleFunc("/coverage-limits", coverageLimitsHandler)
	http.HandleFunc("/service-subcategories", serviceSubcategoriesHandler)

	fmt.Printf("PetWell Backend running at http://localhost:%s\n", port)
	fmt.Println("Endpoints: /vaccines, /clinics, /emergency-clinics, /posts, /register, /insurance-providers, /coverage-limits, /service-subcategories")
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		fmt.Printf("Server failed to start: %v\n", err)
	}
}
