import sqlite3
import os
import shutil

# Paths
REPO_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DB_PATH = os.path.join(REPO_ROOT, "pet_insurance.db")
BACKUP_PATH = os.path.join(REPO_ROOT, "pet_insurance_backup.db")

def backup_db():
    if os.path.exists(DB_PATH):
        shutil.copy2(DB_PATH, BACKUP_PATH)
        print(f"Backup created at {BACKUP_PATH}")
    else:
        print("Database not found!")
        exit(1)

def refactor_db():
    conn = sqlite3.connect(DB_PATH)
    cursor = conn.cursor()
    
    # Enable foreign keys
    cursor.execute("PRAGMA foreign_keys = OFF;")
    
    # 1. insurance_provider
    print("Migrating insurance_provider...")
    cursor.execute("ALTER TABLE insurance_provider RENAME TO insurance_provider_old")
    cursor.execute("""
        CREATE TABLE insurance_provider (
            company_id INTEGER PRIMARY KEY AUTOINCREMENT,
            company_name TEXT NOT NULL,
            company_name_zh TEXT,
            company_logo TEXT
        )
    """)
    cursor.execute("""
        INSERT INTO insurance_provider (company_id, company_name, company_logo)
        SELECT company_id, company_name, company_logo FROM insurance_provider_old
    """)
    cursor.execute("DROP TABLE insurance_provider_old")

    # 2. product
    print("Migrating product...")
    cursor.execute("ALTER TABLE product RENAME TO product_old")
    cursor.execute("""
        CREATE TABLE product (
            insurance_id INTEGER PRIMARY KEY AUTOINCREMENT,
            provider_id INTEGER,
            insurance_name TEXT,
            insurance_name_zh TEXT,
            min_age TEXT,
            min_age_zh TEXT,
            max_age TEXT,
            max_age_zh TEXT,
            suitable_pet_type TEXT,
            suitable_pet_type_zh TEXT,
            cat_breed_type TEXT,
            cat_breed_type_zh TEXT,
            dog_breed_type TEXT,
            dog_breed_type_zh TEXT,
            breed_type_remark TEXT,
            breed_type_remark_zh TEXT,
            payment_mode TEXT,
            payment_mode_zh TEXT,
            waiting_period TEXT,
            waiting_period_zh TEXT,
            information_link TEXT,
            information_link_zh TEXT,
            update_time TEXT,
            FOREIGN KEY(provider_id) REFERENCES insurance_provider(company_id)
        )
    """)
    # Construct INSERT statement explicitly mapping old columns to new standard columns
    cursor.execute("""
        INSERT INTO product (
            insurance_id, provider_id, insurance_name, min_age, max_age, 
            suitable_pet_type, cat_breed_type, dog_breed_type, breed_type_remark, 
            payment_mode, waiting_period, information_link, update_time
        )
        SELECT 
            insurance_id, provider_id, insurance_name, min_age, max_age, 
            suitable_pet_type, cat_breed_type, dog_breed_type, breed_type_remark, 
            payment_mode, waiting_period, information_link, update_time
        FROM product_old
    """)
    cursor.execute("DROP TABLE product_old")

    # 3. coverage
    print("Migrating coverage...")
    cursor.execute("ALTER TABLE coverage RENAME TO coverage_old")
    cursor.execute("""
        CREATE TABLE coverage (
            coverage_id INTEGER PRIMARY KEY,
            product_id INTEGER,
            coverage_type TEXT,
            coverage_type_zh TEXT,
            coverage_limit TEXT,
            coverage_limit_zh TEXT,
            coverage_remark TEXT,
            coverage_remark_zh TEXT,
            FOREIGN KEY(product_id) REFERENCES product(insurance_id)
        )
    """)
    cursor.execute("""
        INSERT INTO coverage (coverage_id, product_id, coverage_type, coverage_limit, coverage_remark)
        SELECT coverage_id, product_id, coverage_type, coverage_limit, coverage_remark FROM coverage_old
    """)
    cursor.execute("DROP TABLE coverage_old")

    # 4. sub_coverage
    print("Migrating sub_coverage...")
    cursor.execute("ALTER TABLE sub_coverage RENAME TO sub_coverage_old")
    cursor.execute("""
        CREATE TABLE sub_coverage (
            sub_coverage_id INTEGER PRIMARY KEY AUTOINCREMENT,
            parent_coverage_id INTEGER NOT NULL,
            sub_coverage_remark TEXT,
            sub_coverage_remark_zh TEXT,
            Field4 INTEGER,
            FOREIGN KEY(parent_coverage_id) REFERENCES coverage(coverage_id)
        )
    """)
    cursor.execute("""
        INSERT INTO sub_coverage (sub_coverage_id, parent_coverage_id, sub_coverage_remark, Field4)
        SELECT sub_coverage_id, parent_coverage_id, sub_coverage_remark, Field4 FROM sub_coverage_old
    """)
    cursor.execute("DROP TABLE sub_coverage_old")

    # 5. coinsurance_info
    print("Migrating coinsurance_info...")
    cursor.execute("ALTER TABLE coinsurance_info RENAME TO coinsurance_info_old")
    cursor.execute("""
        CREATE TABLE coinsurance_info (
            provider_id INTEGER,
            min_age TEXT,
            min_age_zh TEXT,
            max_age TEXT,
            max_age_zh TEXT,
            vet_type TEXT,
            vet_type_zh TEXT,
            coinsurance_percentage NUMERIC,
            FOREIGN KEY(provider_id) REFERENCES insurance_provider(company_id)
        )
    """)
    cursor.execute("""
        INSERT INTO coinsurance_info (provider_id, min_age, max_age, coinsurance_percentage, vet_type)
        SELECT provider_id, min_age, max_age, coinsurance_percentage, vet_type FROM coinsurance_info_old
    """)
    cursor.execute("DROP TABLE coinsurance_info_old")

    conn.commit()
    conn.execute("PRAGMA foreign_keys = ON;")
    conn.close()
    print("Database refactor complete.")

if __name__ == "__main__":
    backup_db()
    refactor_db()
