#!/usr/bin/env python3
"""
Build Insurance Database (insurance.db)

This script reads the CSV files and creates a relational SQLite database.

Tables:
- pet_insurance_comparison (provider_key as PRIMARY KEY)
- coverage_limits (FOREIGN KEY to pet_insurance_comparison.provider_key)

Usage:
    python3 scripts/build_insurance_db.py
"""

import csv
import sqlite3
import os

# Paths
SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
REPO_ROOT = os.path.dirname(SCRIPT_DIR)
DATA_DIR = os.path.join(REPO_ROOT, "Data")
DB_PATH = os.path.join(REPO_ROOT, "insurance.db")

INSURANCE_CSV = os.path.join(DATA_DIR, "Pet Insurance Comparison.csv")
LIMITS_CSV = os.path.join(DATA_DIR, "Coverage Limits.csv")


def create_tables(conn):
    """Create the relational tables."""
    cursor = conn.cursor()
    
    # Table 1: Pet Insurance Comparison
    cursor.execute("""
        CREATE TABLE IF NOT EXISTS pet_insurance_comparison (
            provider_key TEXT PRIMARY KEY,
            insurance_provider TEXT NOT NULL,
            company_name TEXT,
            plan_name TEXT,
            category TEXT,
            subcategory TEXT,
            coverage_percentage TEXT,
            cancer_cash_hkd REAL,
            cancer_cash_notes TEXT,
            additional_critical_cash_benefit REAL,
            coverage_mode TEXT
        )
    """)
    
    # Table 2: Coverage Limits (with FK to pet_insurance_comparison)
    cursor.execute("""
        CREATE TABLE IF NOT EXISTS coverage_limits (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            limit_item TEXT NOT NULL,
            provider_key TEXT NOT NULL,
            level TEXT,
            category TEXT,
            subcategory TEXT,
            coverage_amount_hkd TEXT,
            notes TEXT,
            FOREIGN KEY (provider_key) REFERENCES pet_insurance_comparison(provider_key)
        )
    """)
    
    # Create index for faster lookups
    cursor.execute("CREATE INDEX IF NOT EXISTS idx_limits_provider_key ON coverage_limits(provider_key)")
    
    # Table 3: Service Subcategories (reference table for all possible services)
    cursor.execute("""
        CREATE TABLE IF NOT EXISTS service_subcategories (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT NOT NULL UNIQUE,
            display_order INTEGER
        )
    """)
    
    conn.commit()
    print("Tables created successfully.")


def import_insurance_providers(conn):
    """Import data from Pet Insurance Comparison.csv"""
    cursor = conn.cursor()
    
    with open(INSURANCE_CSV, 'r', encoding='utf-8-sig') as f:
        reader = csv.DictReader(f)
        count = 0
        for row in reader:
            provider_key = row.get('Provider Key', '').strip()
            if not provider_key:
                continue
            
            # Parse cancer cash
            cancer_cash = row.get('Cancer Cash (HKD)', '').strip()
            cancer_cash_val = float(cancer_cash) if cancer_cash and cancer_cash.isdigit() else None
            
            # Parse additional benefit
            additional = row.get('Additional Critical Cash Benefit', '').strip()
            additional_val = float(additional) if additional and additional.isdigit() else None
            
            # Split provider into company and plan
            provider_full = row.get('Insurance Provider', '').strip()
            if ' —— ' in provider_full:
                company, plan = provider_full.split(' —— ', 1)
            elif '----' in provider_full:
                company, plan = provider_full.split('----', 1)
            else:
                company = provider_full
                plan = ''
            
            # Determine coverage_mode based on company name
            company_clean = company.strip().lower()
            if 'one degree' in company_clean:
                coverage_mode = 'big_bucket'
            elif 'blue cross' in company_clean:
                coverage_mode = 'bento_box'
            else:
                coverage_mode = 'unknown'

            cursor.execute("""
                INSERT OR REPLACE INTO pet_insurance_comparison 
                (provider_key, insurance_provider, company_name, plan_name, category, subcategory, coverage_percentage, 
                 cancer_cash_hkd, cancer_cash_notes, additional_critical_cash_benefit, coverage_mode)
                VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
            """, (
                provider_key,
                provider_full,
                company.strip(),
                plan.strip(),
                row.get('Category', '').strip(),
                row.get('Subcategory', '').strip(),
                row.get('Coverage Percentage', '').strip(),
                cancer_cash_val,
                row.get('Cancer Cash Notes', '').strip(),
                additional_val,
                coverage_mode
            ))
            count += 1
    
    conn.commit()
    print(f"Imported {count} insurance providers.")


def import_coverage_limits(conn):
    """Import data from Coverage Limits.csv"""
    cursor = conn.cursor()
    
    with open(LIMITS_CSV, 'r', encoding='utf-8-sig') as f:
        reader = csv.DictReader(f)
        count = 0
        for row in reader:
            provider_key = row.get('Provider Key', '').strip()
            limit_item = row.get('Limit Item', '').strip()
            if not provider_key or not limit_item:
                continue
            
            cursor.execute("""
                INSERT INTO coverage_limits 
                (limit_item, provider_key, level, category, subcategory, coverage_amount_hkd, notes)
                VALUES (?, ?, ?, ?, ?, ?, ?)
            """, (
                limit_item,
                provider_key,
                row.get('Level', '').strip(),
                row.get('Category', '').strip(),
                row.get('Subcategory', '').strip(),
                row.get('Coverage Amount (HKD)', '').strip(),
                row.get('Notes', '').strip()
            ))
            count += 1
    
    conn.commit()
    print(f"Imported {count} coverage limits.")


def verify_relations(conn):
    """Verify the relational integrity."""
    cursor = conn.cursor()
    
    # Check FK integrity
    cursor.execute("""
        SELECT cl.limit_item, cl.provider_key 
        FROM coverage_limits cl
        LEFT JOIN pet_insurance_comparison pic ON cl.provider_key = pic.provider_key
        WHERE pic.provider_key IS NULL
    """)
    orphans = cursor.fetchall()
    
    if orphans:
        print(f"WARNING: Found {len(orphans)} coverage limits without matching provider:")
        for o in orphans[:5]:
            print(f"  - {o}")
    else:
        print("All coverage limits have valid provider references.")
    
    # Summary
    cursor.execute("SELECT COUNT(*) FROM pet_insurance_comparison")
    provider_count = cursor.fetchone()[0]
    
    cursor.execute("SELECT COUNT(*) FROM coverage_limits")
    limits_count = cursor.fetchone()[0]
    
    cursor.execute("SELECT COUNT(*) FROM service_subcategories")
    subcategory_count = cursor.fetchone()[0]
    
    print(f"\nDatabase Summary:")
    print(f"  - Providers: {provider_count}")
    print(f"  - Coverage Limits: {limits_count}")
    print(f"  - Service Subcategories: {subcategory_count}")


def import_service_subcategories(conn):
    """Create reference table of all possible service subcategories.
    
    Based on Blue Cross Type A product which has the most comprehensive list.
    These are sorted alphabetically for consistent display.
    """
    cursor = conn.cursor()
    
    # All subcategories from Blue Cross Type A (the most comprehensive)
    # Sorted alphabetically
    subcategories = [
        "Anaesthetists",
        "Chemotherapy Benefit",
        "Consultation",
        "Euthanasia",
        "Hospitalization",
        "Medication",
        "Miscellaneous",
        "MRI & CT",
        "Operating Theatre",
        "Prosthesis or Wheelchair",
        "Specialist Consultation",
        "Surgery",
        "Ultrasound & Lab Tests",
        "X-rays",
    ]
    
    for i, name in enumerate(subcategories, start=1):
        cursor.execute("""
            INSERT OR IGNORE INTO service_subcategories (name, display_order)
            VALUES (?, ?)
        """, (name, i))
    
    conn.commit()
    print(f"Imported {len(subcategories)} service subcategories.")


def main():
    # Remove old DB if exists
    if os.path.exists(DB_PATH):
        os.remove(DB_PATH)
        print(f"Removed old database: {DB_PATH}")
    
    # Connect and build
    conn = sqlite3.connect(DB_PATH)
    conn.execute("PRAGMA foreign_keys = ON")
    
    try:
        create_tables(conn)
        import_insurance_providers(conn)
        import_coverage_limits(conn)
        import_service_subcategories(conn)
        verify_relations(conn)
        print(f"\nDatabase created: {DB_PATH}")
    finally:
        conn.close()


if __name__ == "__main__":
    main()

