import sys
import time
import requests
import numpy as np
import matplotlib.pyplot as plt

def load_test(url: str, duration_seconds: int = 30, timeout_seconds: int = 10):
    response_times_ms = []
    statuses = []
    errors = 0

    start_time = time.time()
    end_time = start_time + duration_seconds

    print(f"Starting load test for {duration_seconds} seconds...")
    i = 0

    while time.time() < end_time:
        i += 1
        try:
            t0 = time.time()
            r = requests.get(url, timeout=timeout_seconds)
            t1 = time.time()

            rt_ms = (t1 - t0) * 1000.0
            response_times_ms.append(rt_ms)
            statuses.append(r.status_code)

            if r.status_code == 200:
                print(f"Request {i}: {rt_ms:.2f}ms (200)")
            else:
                print(f"Request {i}: {rt_ms:.2f}ms (status={r.status_code})")

        except requests.exceptions.RequestException as e:
            errors += 1
            print(f"Request {i}: FAILED ({e})")

    return response_times_ms, statuses, errors

def summarize(times_ms, statuses, errors):
    ok_count = sum(1 for s in statuses if s == 200)
    total = len(statuses) + errors

    if len(times_ms) == 0:
        print("\nNo successful requests recorded.")
        print(f"Total attempts: {total}, Errors: {errors}")
        return

    arr = np.array(times_ms, dtype=float)

    print("\nStatistics:")
    print(f"Total attempts: {total}")
    print(f"Successful (200): {ok_count}")
    print(f"Errors: {errors}")
    print(f"Average: {np.mean(arr):.2f}ms")
    print(f"Median (p50): {np.median(arr):.2f}ms")
    print(f"p95: {np.percentile(arr, 95):.2f}ms")
    print(f"p99: {np.percentile(arr, 99):.2f}ms")
    print(f"Max: {np.max(arr):.2f}ms")

def plot(times_ms, out_prefix="response_times"):
    if len(times_ms) == 0:
        return

    plt.figure(figsize=(12, 8))

    # Histogram
    plt.subplot(2, 1, 1)
    plt.hist(times_ms, bins=50, alpha=0.7)
    plt.xlabel("Response Time (ms)")
    plt.ylabel("Frequency")
    plt.title("Distribution of Response Times")

    # Scatter over time
    plt.subplot(2, 1, 2)
    plt.scatter(range(len(times_ms)), times_ms, alpha=0.6)
    plt.xlabel("Request Number")
    plt.ylabel("Response Time (ms)")
    plt.title("Response Times Over Time")

    plt.tight_layout()
    # Save files for submission screenshots
    plt.savefig(f"{out_prefix}.png", dpi=200)
    plt.show()

# if __name__ == "__main__":
#     EC2_URL = "http://54.187.40.224:8080/albums"  # <-- your EC2 public IP endpoint
#     times_ms, statuses, errors = load_test(EC2_URL, duration_seconds=30)
#     summarize(times_ms, statuses, errors)
#     plot(times_ms, out_prefix="ec2_albums_30s")

if __name__ == "__main__":

    if len(sys.argv) < 2:
        print("Usage: python load_test.py <public_ip> [port] [path]")
        sys.exit(1)

    public_ip = sys.argv[1]
    port = sys.argv[2] if len(sys.argv) > 2 else "8080"
    path = sys.argv[3] if len(sys.argv) > 3 else "/albums"

    url = f"http://{public_ip}:{port}{path}"

    times_ms, statuses, errors = load_test(url)
    summarize(times_ms, statuses, errors)
    plot(times_ms)
