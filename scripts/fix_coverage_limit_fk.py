import sqlite3
import os

# Paths
REPO_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DB_PATH = os.path.join(REPO_ROOT, "pet_insurance.db")

def fix_fk():
    conn = sqlite3.connect(DB_PATH)
    cursor = conn.cursor()
    
    # Enable foreign keys
    cursor.execute("PRAGMA foreign_keys = OFF;")
    
    print("Migrating coverage_limit...")
    cursor.execute("ALTER TABLE coverage_limit RENAME TO coverage_limit_old")
    
    # Create new table with correct FK
    cursor.execute("""
        CREATE TABLE coverage_limit (
            coverage_id INTEGER,
            product_id INTEGER,
            coverage_limit INTEGER,
            PRIMARY KEY(coverage_id, product_id),
            FOREIGN KEY(coverage_id) REFERENCES coverage_list(coverage_id),
            FOREIGN KEY(product_id) REFERENCES product(insurance_id)
        )
    """)
    
    # Copy data
    cursor.execute("""
        INSERT INTO coverage_limit (coverage_id, product_id, coverage_limit)
        SELECT coverage_id, product_id, coverage_limit FROM coverage_limit_old
    """)
    
    cursor.execute("DROP TABLE coverage_limit_old")
    
    conn.commit()
    conn.execute("PRAGMA foreign_keys = ON;")
    conn.close()
    print("FK fix complete.")

if __name__ == "__main__":
    fix_fk()
