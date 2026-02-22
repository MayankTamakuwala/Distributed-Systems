from locust import FastHttpUser, task, between
import random

SEARCH_TERMS = [
    "Electronics", "Books", "Home", "Clothing", "Sports", "Beauty", "Toys", "Grocery",
    "Alpha", "Nimbus", "Orion", "Vertex", "Pulse", "Apex", "Nova", "Summit",
    "Product",
]


class SearchUser(FastHttpUser):
    wait_time = between(0, 0.01)

    @task(10)
    def search_products(self):
        """Primary load: search endpoint - this drives CPU usage."""
        q = random.choice(SEARCH_TERMS)
        with self.client.get(
            f"/products/search?q={q}",
            name="GET /products/search",
            catch_response=True,
        ) as r:
            if r.status_code == 200:
                r.success()
            else:
                r.failure(f"Expected 200, got {r.status_code}")

    @task(1)
    def health_check(self):
        """Lightweight probe — keeps baseline traffic realistic."""
        with self.client.get(
            "/health",
            name="GET /health",
            catch_response=True,
        ) as r:
            if r.status_code == 200:
                r.success()
            else:
                r.failure(f"Expected 200, got {r.status_code}")
