from flask import Flask, request, jsonify
import boto3
import re
from collections import Counter
import json

app = Flask(__name__)
s3 = boto3.client("s3")

@app.route("/map", methods=["GET"])
def map_chunk():
    bucket = request.args.get("bucket")
    key = request.args.get("key")

    obj = s3.get_object(Bucket=bucket, Key=key)
    text = obj["Body"].read().decode("utf-8").lower()

    words = re.findall(r"\b[a-z0-9']+\b", text)
    counts = Counter(words)

    output_key = key.replace("chunks/", "maps/").replace(".txt", ".json")
    s3.put_object(
        Bucket=bucket,
        Key=output_key,
        Body=json.dumps(counts)
    )

    return jsonify({"map_output": output_key})

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
