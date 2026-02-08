from flask import Flask, request, jsonify
import boto3
import json
from collections import Counter

app = Flask(__name__)
s3 = boto3.client("s3")

@app.route("/reduce", methods=["GET"])
def reduce_maps():
    bucket = request.args.get("bucket")
    keys = request.args.getlist("key")

    final_counts = Counter()

    for k in keys:
        obj = s3.get_object(Bucket=bucket, Key=k)
        data = json.loads(obj["Body"].read())
        final_counts.update(data)

    output_key = "final/result.json"
    s3.put_object(
        Bucket=bucket,
        Key=output_key,
        Body=json.dumps(final_counts)
    )

    return jsonify({"result": output_key})

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
