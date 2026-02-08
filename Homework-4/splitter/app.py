from flask import Flask, request, jsonify
import boto3

app = Flask(__name__)
s3 = boto3.client("s3")

@app.route("/split", methods=["GET"])
def split_file():
    bucket = request.args.get("bucket")
    key = request.args.get("key")
    chunks = int(request.args.get("chunks", 3))

    obj = s3.get_object(Bucket=bucket, Key=key)
    text = obj["Body"].read().decode("utf-8")

    lines = text.splitlines()
    size = len(lines) // chunks
    chunk_keys = []

    for i in range(chunks):
        part = lines[i*size:(i+1)*size] if i < chunks - 1 else lines[i*size:]
        chunk_key = f"chunks/chunk{i}.txt"
        s3.put_object(
            Bucket=bucket,
            Key=chunk_key,
            Body="\n".join(part)
        )
        chunk_keys.append(chunk_key)

    return jsonify({"chunks": chunk_keys})

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
