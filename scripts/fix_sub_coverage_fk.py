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
    
    print("Migrating sub_coverage...")
    cursor.execute("ALTER TABLE sub_coverage RENAME TO sub_coverage_old")
    
    # Create new table with correct FK and NO Field4 (as verified it's gone)
    cursor.execute("""
        CREATE TABLE sub_coverage (
            sub_coverage_id INTEGER PRIMARY KEY AUTOINCREMENT,
            parent_coverage_id INTEGER,
            sub_coverage_remark TEXT,
            sub_coverage_remark_zh TEXT,
            FOREIGN KEY(parent_coverage_id) REFERENCES coverage_list(coverage_id)
        )
    """)
    
    # Copy data
    cursor.execute("""
        INSERT INTO sub_coverage (sub_coverage_id, parent_coverage_id, sub_coverage_remark, sub_coverage_remark_zh)
        SELECT sub_coverage_id, parent_coverage_id, sub_coverage_remark, sub_coverage_remark_zh FROM sub_coverage_old
    """)
    
    cursor.execute("DROP TABLE sub_coverage_old")
    
    conn.commit()
    conn.execute("PRAGMA foreign_keys = ON;")
    conn.close()
    print("FK fix complete.")

if __name__ == "__main__":
    fix_fk()
