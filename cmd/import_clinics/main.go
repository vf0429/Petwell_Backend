package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"googlemaps.github.io/maps"
)

// Clinic matches the CSV structure
type Clinic struct {
	ClinicID       string
	Name           string
	Address        string
	PhoneRegular   string
	PhoneEmergency string
	Whatsapp       string
	OpeningHours   string
	Emergency24h   string
	WebsiteURL     string
	ApplemapURL    string
	Latitude       string
	Longitude      string
	Rating         string
	GooglePlaceID  string
	PhotoReference string
}

func main() {
	// 1. Load Environment Variables
	if err := godotenv.Load(); err != nil {
		// If .env is in root, try loading from there
		if err := godotenv.Load("../../.env"); err != nil {
			log.Println("Warning: .env file not found")
		}
	}

	apiKey := os.Getenv("MAPS_API_KEY")
	if apiKey == "" {
		log.Fatal("MAPS_API_KEY is not set")
	}

	// 2. Initialize Maps Client
	c, err := maps.NewClient(maps.WithAPIKey(apiKey))
	if err != nil {
		log.Fatalf("fatal error: %s", err)
	}

	// 3. Load Existing CSV to avoid duplicates
	csvPath := "data/clinics.csv"
	// Adjust path if running from root vs cmd folder
	if _, err := os.Stat(csvPath); os.IsNotExist(err) {
		csvPath = "../../data/clinics.csv"
	}
	
	existingClinics, maxID := loadExistingClinics(csvPath)
	fmt.Printf("Loaded %d existing clinics. Max ID: %d\n", len(existingClinics), maxID)

	// 4. Search for Veterinary Clinics in Hong Kong
	// We will try a few keywords to get good coverage
	keywords := []string{"Veterinary Clinic Hong Kong", "Animal Hospital Hong Kong", "Vet Hong Kong"}
	
	var newClinics []Clinic
	uniquePlaceIDs := make(map[string]bool)
	
	// Mark existing Place IDs as visited
	for _, row := range existingClinics {
		if len(row) > 13 && row[13] != "" {
			uniquePlaceIDs[row[13]] = true
		}
	}

	for _, keyword := range keywords {
		fmt.Printf("Searching for: %s...\n", keyword)
		
		var nextPageToken string
		
		// Pagination loop
		for {
			var searchReq *maps.TextSearchRequest
			if nextPageToken != "" {
				// Create a fresh request for pagination
				searchReq = &maps.TextSearchRequest{
					PageToken: nextPageToken,
				}
				time.Sleep(2 * time.Second) // Wait for token to become active
			} else {
				// Initial request
				searchReq = &maps.TextSearchRequest{
					Query: keyword,
				}
			}

			resp, err := c.TextSearch(context.Background(), searchReq)
			if err != nil {
				log.Printf("Error searching: %s. PageToken: %s", err, nextPageToken)
				break 
			}
			
			// If no results, stop
			if len(resp.Results) == 0 {
				break
			}

			for _, result := range resp.Results {
				// Deduplicate
				if uniquePlaceIDs[result.PlaceID] {
					continue
				}
				uniquePlaceIDs[result.PlaceID] = true
				
				// Fetch details
				details, err := c.PlaceDetails(context.Background(), &maps.PlaceDetailsRequest{
					PlaceID: result.PlaceID,
				})
				
				if err != nil {
					fmt.Printf("Error fetching details for %s: %v\n", result.Name, err)
					continue
				}

				maxID++
				
				// Safe extraction
				lat := details.Geometry.Location.Lat
				lng := details.Geometry.Location.Lng
				
				openingHoursStr := ""
				is24h := "FALSE"
				if details.OpeningHours != nil && len(details.OpeningHours.WeekdayText) > 0 {
					openingHoursStr = strings.Join(details.OpeningHours.WeekdayText, "; ")
					for _, t := range details.OpeningHours.WeekdayText {
						if strings.Contains(strings.ToLower(t), "open 24 hours") || strings.Contains(strings.ToLower(t), "24-hour") {
							is24h = "TRUE"
							break
						}
					}
				}

				photoRef := ""
				if len(details.Photos) > 0 {
					photoRef = details.Photos[0].PhotoReference
				}

				newClinic := Clinic{
					ClinicID:       strconv.Itoa(maxID),
					Name:           details.Name,
					Address:        details.FormattedAddress,
					PhoneRegular:   details.InternationalPhoneNumber,
					WebsiteURL:     details.Website,
					OpeningHours:   openingHoursStr,
					Emergency24h:   is24h,
					Latitude:       fmt.Sprintf("%f", lat),
					Longitude:      fmt.Sprintf("%f", lng),
					Rating:         fmt.Sprintf("%.1f", details.Rating),
					GooglePlaceID:  result.PlaceID,
					PhotoReference: photoRef,
					ApplemapURL:    "https://maps.apple.com/?q=" + strings.ReplaceAll(details.Name, " ", "+"),
				}

				newClinics = append(newClinics, newClinic)
				fmt.Printf("Found: %s\n", newClinic.Name)
			}
			
			nextPageToken = resp.NextPageToken
			if nextPageToken == "" {
				break
			}
		}
	}

	// 5. Append to CSV
	if len(newClinics) > 0 {
		appendToCSV(csvPath, newClinics)
		fmt.Printf("Successfully added %d new clinics to %s\n", len(newClinics), csvPath)
	} else {
		fmt.Println("No new clinics found.")
	}
}

func loadExistingClinics(path string) ([][]string, int) {
	file, err := os.Open(path)
	if err != nil {
		return [][]string{}, 0
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return [][]string{}, 0
	}

	maxID := 0
	for i, row := range records {
		if i == 0 || len(row) == 0 { continue }
		if id, err := strconv.Atoi(row[0]); err == nil && id > maxID {
			maxID = id
		}
	}
	return records, maxID
}

func appendToCSV(path string, newClinics []Clinic) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("failed to open file: %s", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	for _, c := range newClinics {
		record := []string{
			c.ClinicID,
			c.Name,
			c.Address,
			c.PhoneRegular,
			c.PhoneEmergency,
			c.Whatsapp,
			c.OpeningHours,
			c.Emergency24h,
			c.WebsiteURL,
			c.ApplemapURL,
			c.Latitude,
			c.Longitude,
			c.Rating,
			c.GooglePlaceID,
			c.PhotoReference,
		}
		if err := writer.Write(record); err != nil {
			log.Printf("error writing record: %s", err)
		}
	}
}
