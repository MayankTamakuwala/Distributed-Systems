from locust import FastHttpUser, task, between
import random

GOOD_ID = 1          # an ID you know exists (seeded / created)
BAD_ID = 999999      # should not exist -> 404

GOOD_PRODUCT = {
    "product_id": GOOD_ID,
    "sku": "SKU-123",
    "manufacturer": "Acme",
    "category_id": 10,
    "weight": 50,
    "some_other_id": 77
}

def bad_product_body_mismatch(path_id: int):
    # product_id mismatch => 400
    return {
        "product_id": path_id + 1,
        "sku": "SKU-123",
        "manufacturer": "Acme",
        "category_id": 10,
        "weight": 50,
        "some_other_id": 77
    }

class ProductUser(FastHttpUser):
    wait_time = between(0.1, 0.5)

    @task(7)
    def get_product_200_or_404(self):
        # mostly fetch existing, sometimes non-existing
        pid = GOOD_ID if random.random() < 0.8 else BAD_ID
        with self.client.get(f"/products/{pid}", name="GET /products/{id}", catch_response=True) as r:
            if pid == GOOD_ID and r.status_code != 200:
                r.failure(f"Expected 200, got {r.status_code}")
            elif pid == BAD_ID and r.status_code != 404:
                r.failure(f"Expected 404, got {r.status_code}")
            else:
                r.success()

    @task(2)
    def post_details_204(self):
        # update details for existing product => 204
        with self.client.post(f"/products/{GOOD_ID}/details", json=GOOD_PRODUCT,
                                name="POST /products/{id}/details (204)", catch_response=True) as r:
            if r.status_code != 204:
                r.failure(f"Expected 204, got {r.status_code}")
            else:
                r.success()

    @task(1)
    def post_details_400(self):
        # mismatch product_id vs path => 400
        body = bad_product_body_mismatch(GOOD_ID)
        with self.client.post(f"/products/{GOOD_ID}/details", json=body,
                                name="POST /products/{id}/details (400)", catch_response=True) as r:
            if r.status_code != 400:
                r.failure(f"Expected 400, got {r.status_code}")
            else:
                r.success()
