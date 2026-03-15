import os
from random import randint

from locust import HttpUser, between, task


def build_order():
    return {
        "customer_id": f"cust-{randint(1, 1000)}",
        "items": [
            {"sku": "item-1", "quantity": 2},
            {"sku": "item-2", "quantity": 1},
        ],
    }


class SyncUser(HttpUser):
    """Test sync endpoint only.
    Normal:  locust --users 5  --spawn-rate 1 --run-time 30s  --headless
    Flash:   locust --users 20 --spawn-rate 10 --run-time 60s --headless
    """

    wait_time = between(0.1, 0.5)

    @task
    def create_sync_order(self):
        self.client.post("/orders/sync", json=build_order(), name="/orders/sync")


class AsyncUser(HttpUser):
    """Test async endpoint only.
    Flash:   locust --users 20 --spawn-rate 10 --run-time 60s --headless
    """

    wait_time = between(0.1, 0.5)

    @task
    def create_async_order(self):
        self.client.post("/orders/async", json=build_order(), name="/orders/async")
