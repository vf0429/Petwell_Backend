package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	port        = "8000"
	jsonFile    = "vaccines.json"
	clinicsFile = "clinics.csv"
	insuranceDB = "pet_insurance.db"
)

// --- Custom Types ---

type NullJsonString struct {
	sql.NullString
}

func (ns NullJsonString) MarshalJSON() ([]byte, error) {
	if !ns.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(ns.String)
}

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
	Content      string    `json:"content"`
	ImageColor   string    `json:"imageColor"`
	Likes        int       `json:"likes"`
	Timestamp    time.Time `json:"timestamp"`
}

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

// --- New Insurance Models (pet_insurance.db) ---

type InsuranceCompany struct {
	CompanyId     int            `json:"company_id"`
	CompanyName   string         `json:"company_name"`
	CompanyNameZh NullJsonString `json:"company_name_zh,omitempty"`
	CompanyLogo   NullJsonString `json:"company_logo,omitempty"`
}

type InsuranceProduct struct {
	InsuranceId       int            `json:"insurance_id"`
	ProviderId        int            `json:"provider_id"`
	InsuranceName     string         `json:"insurance_name"`
	InsuranceNameZh   NullJsonString `json:"insurance_name_zh,omitempty"`
	Remark            NullJsonString `json:"remark,omitempty"`
	RemarkZh          NullJsonString `json:"remark_zh,omitempty"`
	MinAge            NullJsonString `json:"min_age,omitempty"`
	MinAgeZh          NullJsonString `json:"min_age_zh,omitempty"`
	MaxAge            NullJsonString `json:"max_age,omitempty"`
	MaxAgeZh          NullJsonString `json:"max_age_zh,omitempty"`
	Coinsurance       NullJsonString `json:"coinsurance,omitempty"`
	CoinsuranceZh     NullJsonString `json:"coinsurance_zh,omitempty"`
	SuitablePetType   NullJsonString `json:"suitable_pet_type,omitempty"`
	SuitablePetTypeZh NullJsonString `json:"suitable_pet_type_zh,omitempty"`
	CatBreedType      NullJsonString `json:"cat_breed_type,omitempty"`
	CatBreedTypeZh    NullJsonString `json:"cat_breed_type_zh,omitempty"`
	DogBreedType      NullJsonString `json:"dog_breed_type,omitempty"`
	DogBreedTypeZh    NullJsonString `json:"dog_breed_type_zh,omitempty"`
	BreedTypeRemark   NullJsonString `json:"breed_type_remark,omitempty"`
	BreedTypeRemarkZh NullJsonString `json:"breed_type_remark_zh,omitempty"`
	PaymentMode       NullJsonString `json:"payment_mode,omitempty"`
	PaymentModeZh     NullJsonString `json:"payment_mode_zh,omitempty"`
	WaitingPeriod     NullJsonString `json:"waiting_period,omitempty"`
	WaitingPeriodZh   NullJsonString `json:"waiting_period_zh,omitempty"`
	InformationLink   NullJsonString `json:"information_link,omitempty"`
	InformationLinkZh NullJsonString `json:"information_link_zh,omitempty"`
	UpdateTime        NullJsonString `json:"update_time,omitempty"`
	Tag               NullJsonString `json:"tag,omitempty"`
	TagZh             NullJsonString `json:"tag_zh,omitempty"`
}

type CoverageItem struct {
	CoverageId     int            `json:"coverage_id"`
	CoverageType   string         `json:"coverage_type"`
	CoverageTypeZh NullJsonString `json:"coverage_type_zh,omitempty"`
}

type CoverageLimit struct {
	CoverageId    int            `json:"coverage_id"`
	ProductId     int            `json:"product_id"`
	CoverageLimit NullJsonString `json:"coverage_limit,omitempty"`
	Remark        NullJsonString `json:"remark,omitempty"`
	RemarkZh      NullJsonString `json:"remark_zh,omitempty"`
}

type SubCoverageLimit struct {
	SubCoverageId       int            `json:"sub_coverage_id"`
	ParentCoverageId    int            `json:"parent_coverage_id"`
	ProductId           int            `json:"product_id"`
	SubCoverageName     NullJsonString `json:"sub_coverage_name,omitempty"`
	SubCoverageNameZh   NullJsonString `json:"sub_coverage_name_zh,omitempty"`
	SubLimit            NullJsonString `json:"sub_limit,omitempty"`
	SubCoverageRemark   NullJsonString `json:"sub_coverage_remark,omitempty"`
	SubCoverageRemarkZh NullJsonString `json:"sub_coverage_remark_zh,omitempty"`
}

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

// --- Helpers ---

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	(*w).Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func openInsuranceDB() (*sql.DB, error) {
	if _, err := os.Stat(insuranceDB); err == nil {
		return sql.Open("sqlite3", insuranceDB)
	}

	ex, err := os.Executable()
	if err == nil {
		path := filepath.Join(filepath.Dir(ex), insuranceDB)
		if _, err := os.Stat(path); err == nil {
			return sql.Open("sqlite3", path)
		}
	}

	return sql.Open("sqlite3", insuranceDB)
}

// --- Handlers ---

func vaccinesHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	if r.Method == http.MethodOptions {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ex, err := os.Executable()
	if err != nil {
		http.Error(w, "Server Error", http.StatusInternalServerError)
		return
	}
	exPath := filepath.Dir(ex)
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

func clinicsHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("[]"))
}

func emergencyClinicsHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("[]"))
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
	usersMutex.Lock()
	users[user.ID] = user
	usersMutex.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
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
		postsMutex.Lock()
		posts = append(posts, post)
		postsMutex.Unlock()
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(post)
	}
}

// --- New Insurance Handlers ---

func insuranceCompaniesHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	if r.Method == http.MethodOptions {
		return
	}

	db, err := openInsuranceDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT company_id, company_name, company_name_zh, company_logo FROM insurance_provider")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var companies []InsuranceCompany
	for rows.Next() {
		var c InsuranceCompany
		if err := rows.Scan(&c.CompanyId, &c.CompanyName, &c.CompanyNameZh, &c.CompanyLogo); err == nil {
			companies = append(companies, c)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(companies)
}

func insuranceProductsHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	if r.Method == http.MethodOptions {
		return
	}

	db, err := openInsuranceDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query(`SELECT insurance_id, provider_id, insurance_name, insurance_name_zh, remark, remark_zh, 
		min_age, min_age_zh, max_age, max_age_zh, coinsurance, coinsurance_zh, suitable_pet_type, suitable_pet_type_zh,
		cat_breed_type, cat_breed_type_zh, dog_breed_type, dog_breed_type_zh, breed_type_remark, breed_type_remark_zh,
		payment_mode, payment_mode_zh, waiting_period, waiting_period_zh, information_link, information_link_zh, update_time,
		tag, tag_zh
		FROM product`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var products []InsuranceProduct
	for rows.Next() {
		var p InsuranceProduct
		if err := rows.Scan(&p.InsuranceId, &p.ProviderId, &p.InsuranceName, &p.InsuranceNameZh, &p.Remark, &p.RemarkZh,
			&p.MinAge, &p.MinAgeZh, &p.MaxAge, &p.MaxAgeZh, &p.Coinsurance, &p.CoinsuranceZh, &p.SuitablePetType, &p.SuitablePetTypeZh,
			&p.CatBreedType, &p.CatBreedTypeZh, &p.DogBreedType, &p.DogBreedTypeZh, &p.BreedTypeRemark, &p.BreedTypeRemarkZh,
			&p.PaymentMode, &p.PaymentModeZh, &p.WaitingPeriod, &p.WaitingPeriodZh, &p.InformationLink, &p.InformationLinkZh, &p.UpdateTime,
			&p.Tag, &p.TagZh); err == nil {
			products = append(products, p)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
}

func coverageListHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	if r.Method == http.MethodOptions {
		return
	}

	db, err := openInsuranceDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT coverage_id, coverage_type, coverage_type_zh FROM coverage_list")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var list []CoverageItem
	for rows.Next() {
		var item CoverageItem
		if err := rows.Scan(&item.CoverageId, &item.CoverageType, &item.CoverageTypeZh); err == nil {
			list = append(list, item)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func coverageLimitsHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	if r.Method == http.MethodOptions {
		return
	}

	db, err := openInsuranceDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT coverage_id, product_id, coverage_limit, remark, remark_zh FROM coverage_limit")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var limits []CoverageLimit
	for rows.Next() {
		var l CoverageLimit
		if err := rows.Scan(&l.CoverageId, &l.ProductId, &l.CoverageLimit, &l.Remark, &l.RemarkZh); err == nil {
			limits = append(limits, l)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(limits)
}

func subCoverageLimitsHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	if r.Method == http.MethodOptions {
		return
	}

	db, err := openInsuranceDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query(`SELECT sub_coverage_id, parent_coverage_id, product_id, sub_coverage_name, sub_coverage_name_zh, 
		sub_limit, sub_coverage_remark, sub_coverage_remark_zh FROM sub_coverage_limit`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var limits []SubCoverageLimit
	for rows.Next() {
		var l SubCoverageLimit
		if err := rows.Scan(&l.SubCoverageId, &l.ParentCoverageId, &l.ProductId, &l.SubCoverageName, &l.SubCoverageNameZh,
			&l.SubLimit, &l.SubCoverageRemark, &l.SubCoverageRemarkZh); err == nil {
			limits = append(limits, l)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(limits)
}

// Legacy Handlers - Return empty
func insuranceProvidersHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("[]"))
}
func serviceSubcategoriesHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("[]"))
}

func main() {
	posts = []BlogPost{
		{ID: "1", AuthorName: "System", Title: "Welcome to PetWell Blog", Content: "This is the start of our community.", Likes: 10, ImageColor: "green", Timestamp: time.Now()},
	}

	http.HandleFunc("/vaccines", vaccinesHandler)
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/posts", postsHandler)
	http.HandleFunc("/clinics", clinicsHandler)
	http.HandleFunc("/emergency-clinics", emergencyClinicsHandler)

	http.HandleFunc("/insurance-companies", insuranceCompaniesHandler)
	http.HandleFunc("/insurance-products", insuranceProductsHandler)
	http.HandleFunc("/coverage-list", coverageListHandler)
	http.HandleFunc("/coverage-limits", coverageLimitsHandler)
	http.HandleFunc("/sub-coverage-limits", subCoverageLimitsHandler)

	http.HandleFunc("/insurance-providers", insuranceProvidersHandler)
	http.HandleFunc("/service-subcategories", serviceSubcategoriesHandler)

	fmt.Printf("PetWell Backend running at http://localhost:%s\n", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		fmt.Printf("Server failed to start: %v\n", err)
	}
}
