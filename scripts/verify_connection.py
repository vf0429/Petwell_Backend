import requests
import json
import sys

def test_rag_connection():
    url = "http://localhost:8000/api/chat"
    
    # 1. Test OPTIONS (CORS Preflight)
    print("Testing OPTIONS (CORS)...")
    try:
        options_resp = requests.options(url)
        print(f"Status: {options_resp.status_code}")
        print(f"Headers: {dict(options_resp.headers)}")
        
        if options_resp.headers.get("Access-Control-Allow-Origin") != "*":
            print("❌ FAILURE: Missing Access-Control-Allow-Origin header")
            return False
    except requests.exceptions.ConnectionError:
        print(f"❌ FAILURE: Could not connect to {url}. Is the server running?")
        return False
    except Exception as e:
        print(f"❌ FAILURE: Error: {e}")
        return False

    # 2. Test POST (Actual Request)
    print("\nTesting POST (Chat Request)...")
    payload = {"query": "Do you cover dental?"}
    try:
        # Note: requests.post doesn't automatically send OPTIONS unless configured, 
        # but browsers will. We check if POST succeeds with payload.
        post_resp = requests.post(url, json=payload)
        print(f"Status: {post_resp.status_code}")
        
        if post_resp.status_code != 200:
            print(f"❌ FAILURE: Expected 200, got {post_resp.status_code}")
            print(f"Response: {post_resp.text}")
            return False
            
        data = post_resp.json()
        
        # Verify JSON structure
        if "answer" not in data:
            print("❌ FAILURE: Response JSON missing 'answer' field")
            return False
            
        print(f"Response Body: {json.dumps(data, indent=2)}")
        print("✅ SUCCESS: Backend is ready for frontend connection!")
        return True
        
    except Exception as e:
        print(f"❌ FAILURE: Error during POST request: {e}")
        return False

if __name__ == "__main__":
    if test_rag_connection():
        sys.exit(0)
    else:
        sys.exit(1)
