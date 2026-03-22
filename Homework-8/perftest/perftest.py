"""
Simple perf test script for the shopping cart API.
Runs 150 operations (50 create + 50 add items + 50 get) and dumps results to JSON.
Usage:
  python perftest.py --base-url http://<alb-dns> --output mysql_test_results.json
"""

import requests
import json
import time
import random
import argparse
from datetime import datetime, timezone


def run_test(base_url, output_file):
    results = []
    cart_ids = []

    # create 50 carts
    print("Phase 1: Creating 50 carts...")
    for i in range(1, 51):
        start = time.time()
        try:
            resp = requests.post(f"{base_url}/shopping-carts", json={"customer_id": i}, timeout=30)
            elapsed_ms = (time.time() - start) * 1000
            ok = resp.status_code == 201
            if ok:
                cart_ids.append(resp.json()["shopping_cart_id"])
            results.append(make_result("create_cart", elapsed_ms, ok, resp.status_code))
        except Exception as e:
            elapsed_ms = (time.time() - start) * 1000
            results.append(make_result("create_cart", elapsed_ms, False, 0))
            print(f"  Error: {e}")

    print(f"  Created {len(cart_ids)} carts")

    # add items to those carts
    print("Phase 2: Adding items to 50 carts...")
    for i in range(50):
        cid = cart_ids[i % len(cart_ids)]
        payload = {"product_id": (i % 5) + 1, "quantity": random.randint(1, 3)}
        start = time.time()
        try:
            resp = requests.post(f"{base_url}/shopping-carts/{cid}/items", json=payload, timeout=30)
            elapsed_ms = (time.time() - start) * 1000
            results.append(make_result("add_items", elapsed_ms, resp.status_code == 204, resp.status_code))
        except Exception as e:
            elapsed_ms = (time.time() - start) * 1000
            results.append(make_result("add_items", elapsed_ms, False, 0))
            print(f"  Error: {e}")

    # fetch carts back
    print("Phase 3: Getting 50 carts...")
    for i in range(50):
        cid = cart_ids[i % len(cart_ids)]
        start = time.time()
        try:
            resp = requests.get(f"{base_url}/shopping-carts/{cid}", timeout=30)
            elapsed_ms = (time.time() - start) * 1000
            results.append(make_result("get_cart", elapsed_ms, resp.status_code == 200, resp.status_code))
        except Exception as e:
            elapsed_ms = (time.time() - start) * 1000
            results.append(make_result("get_cart", elapsed_ms, False, 0))
            print(f"  Error: {e}")

    with open(output_file, "w") as f:
        json.dump(results, f, indent=2)
    print(f"Done! {len(results)} results -> {output_file}")


def make_result(op, elapsed_ms, success, status_code):
    return {
        "operation": op,
        "response_time": round(elapsed_ms, 2),
        "success": success,
        "status_code": status_code,
        "timestamp": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
    }


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--base-url", default="http://localhost:8080")
    parser.add_argument("--output", default="test_results.json")
    args = parser.parse_args()
    run_test(args.base_url, args.output)
