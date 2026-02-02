from locust import FastHttpUser, task, between
import random


class AlbumUser(FastHttpUser):
    # Add a small wait so your CPU doesn't instantly max out at high user counts
    wait_time = between(0.1, 0.3)

    def on_start(self):
        # Cache known album IDs for GET /albums/:id
        self.album_ids = []
        self.refresh_album_ids()

    def refresh_album_ids(self):
        # Fetch albums list and store IDs
        with self.client.get("/albums", name="GET /albums", catch_response=True) as resp:
            if resp.status_code != 200:
                resp.failure(f"GET /albums failed: {resp.status_code}")
                return
            try:
                data = resp.json()
                self.album_ids = [a["id"] for a in data if "id" in a]
            except Exception as e:
                resp.failure(f"Bad JSON from /albums: {e}")

    # --- GET tasks (3 parts) ---

    @task(2)  # 2 parts out of total 3 GET weight
    def get_albums(self):
        with self.client.get("/albums", name="GET /albums", catch_response=True) as resp:
            if resp.status_code != 200:
                resp.failure(f"GET /albums failed: {resp.status_code}")

    @task(1)  # 1 part out of total 3 GET weight
    def get_album_by_id(self):
        if not self.album_ids:
            self.refresh_album_ids()

        if not self.album_ids:
            # If still empty, don't cause failures; just skip this round
            return

        album_id = random.choice(self.album_ids)
        with self.client.get(f"/albums/{album_id}", name="GET /albums/:id", catch_response=True) as resp:
            if resp.status_code != 200:
                resp.failure(f"GET /albums/{album_id} failed: {resp.status_code}")

    # --- POST task (1 part) ---

    @task(1)
    def post_album(self):
        # Valid payload; ID can be omitted and server will auto-generate
        payload = {
            "title": f"Test Album {random.randint(1, 10_000_000)}",
            "artist": "Locust User",
            "price": round(random.uniform(1.00, 200.00), 2),
        }

        with self.client.post("/albums", json=payload, name="POST /albums", catch_response=True) as resp:
            if resp.status_code not in (200, 201):
                resp.failure(f"POST /albums failed: {resp.status_code}")
                return

        # After adding, refresh IDs sometimes so GET /albums/:id stays valid as data grows
        if random.random() < 0.1:
            self.refresh_album_ids()
