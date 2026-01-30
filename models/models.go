package models

import (
	"database/sql"
)

// PetInsuranceComparison represents the insurance provider details
type PetInsuranceComparison struct {
	ID                            int             `json:"id" db:"id"`
	InsuranceProvider             string          `json:"insurance_provider" db:"insurance_provider"`
	ProviderKey                   string          `json:"provider_key" db:"provider_key"`
	Category                      string          `json:"category" db:"category"`
	Subcategory                   string          `json:"subcategory" db:"subcategory"`
	CoveragePercentage            string          `json:"coverage_percentage" db:"coverage_percentage"`
	CancerCash                    sql.NullFloat64 `json:"cancer_cash" db:"cancer_cash"`
	CancerCashNotes               sql.NullString  `json:"cancer_cash_notes" db:"cancer_cash_notes"`
	AdditionalCriticalCashBenefit sql.NullFloat64 `json:"additional_critical_cash_benefit" db:"additional_critical_cash_benefit"`
}

// CoverageLimit represents specific coverage limits
type CoverageLimit struct {
	ID                int            `json:"id" db:"id"`
	LimitItem         string         `json:"limit_item" db:"limit_item"`
	ProviderKey       string         `json:"provider_key" db:"provider_key"`
	Category          string         `json:"category" db:"category"`
	Subcategory       string         `json:"subcategory" db:"subcategory"`
	Level             string         `json:"level" db:"level"`
	CoverageAmountHKD sql.NullString `json:"coverage_amount_hkd" db:"coverage_amount_hkd"` // Stored as string to preserve formatting or "Nil"
	Notes             sql.NullString `json:"notes" db:"notes"`
}
