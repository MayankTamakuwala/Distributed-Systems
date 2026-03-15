# Hummingbird API Debugging Assignment
## Submission Report

**Student:** Mayank Tamakuwala \
**Date:** March 7, 2026 \
**Assignment:** Debug the Hummingbird API using Claude Code \
**Environment:** AWS CloudShell, us-west-2, Claude Code v2.1.71 with AWS Bedrock (Opus 4.6) \
**Status:** All 4 tickets + Bonus completed and verified

---

## Project Context

### What is Hummingbird?

Hummingbird is a media management REST API built with Node.js and Express. It allows users to upload images, track their processing status, and download processed versions. The system runs on AWS infrastructure provisioned via Terraform.

### Architecture Overview

```
Client ---> ALB (port 80) ---> ECS Fargate Container (port 9000) ---> Express API
                                    |
                                    +---> S3 (file storage: uploads/ and resized/ prefixes)
                                    +---> DynamoDB (metadata: PK=MEDIA#<id>, SK=METADATA)
                                    +---> SNS ---> SQS ---> Worker (separate ECS task)
```

**Infrastructure components:**
- **ALB (Application Load Balancer):** Receives client traffic on port 80, forwards to ECS containers on port 9000
- **ECS Fargate:** Runs two services, the API container and a background worker container
- **S3:** Stores uploaded images under `uploads/<mediaId>/<filename>` and processed copies under `resized/<mediaId>/<filename>`
- **DynamoDB:** Single table design with composite key (PK + SK). Each media item is stored with `PK=MEDIA#<mediaId>` and `SK=METADATA`
- **SNS/SQS:** Event bus for async processing. Upload triggers a `media.v1.resize` event via SNS, which fans out to SQS for the worker to consume

### Image Lifecycle (End-to-End Flow)

1. **Upload (POST /v1/media/upload?width=N):** Client uploads a file via multipart form. The API streams it directly to S3 (no disk), saves metadata to DynamoDB with `status=PENDING`, publishes a resize event to SNS, and returns `202 { mediaId }`.

2. **Background Processing:** The worker service polls SQS, receives the resize event, sets status to `PROCESSING`, copies the file from `uploads/` to `resized/` in S3 (simulated resize), then sets status to `COMPLETE`. There is also a synchronous endpoint `PUT /v1/media/:id/resize` that does the same inline.

3. **Status Polling (GET /v1/media/:id/status):** Clients poll this endpoint to check if processing is done. Returns `{ status: "PENDING" | "PROCESSING" | "COMPLETE" | "ERROR" }`.

4. **Download (GET /v1/media/:id/download):** If status is `COMPLETE`, returns a `302` redirect to a presigned S3 URL (valid 1 hour). If not ready, returns `202` with `Retry-After: 60` and a `Location` header pointing to the status endpoint.

### Key Source Files

| File | Purpose |
|------|---------|
| `server.js` | Express app setup, health check endpoint, mounts routes, starts listening |
| `routes/media.js` | Route definitions linking URL patterns to controllers with middleware |
| `controllers/media.js` | All 6 request handlers: upload, status, download, get, resize, delete |
| `actions/uploadMedia.js` | Streaming file upload to S3 via formidable multipart parser |
| `clients/s3.js` | S3 operations: upload, generate presigned download URL, copy, delete |
| `clients/dynamodb.js` | DynamoDB CRUD: createMedia, getMedia, setMediaStatus, deleteMedia |
| `clients/sns.js` | Publishes resize and delete events to SNS topics |
| `worker/processor.js` | SQS consumer that processes resize and delete events |
| `core/constants.js` | Status enums (PENDING, PROCESSING, COMPLETE, ERROR), event types, file size limits |
| `core/responses.js` | HTTP response helper functions (sendOkResponse, sendAcceptedResponse, etc.) |
| `middlewares/validateWidth.js` | Validates width parameter is a number between 100-1024 |
| `middlewares/setMediaWidth.js` | Parses width from query/body, defaults to 500, attaches to req.hummingbirdOptions |

### Investigation Methodology

For each ticket, the debugging process followed these steps:
1. **Read the source code** using Claude Code to understand the relevant code path
2. **Reproduce the bug** by making curl requests against the live API (ALB endpoint)
3. **Check CloudWatch logs** using `aws logs tail /ecs/hummingbird/production/api --follow`
4. **Identify the root cause** by tracing the symptom through the code
5. **Apply the fix** in the source code
6. **Rebuild and redeploy** using `docker build` + `aws ecs update-service`
7. **Verify the fix** by repeating the curl request and confirming correct behavior

---

## Ticket #1: Server Crashes on Startup Without APP_PORT

**Severity:** Critical \
**File:** `server.js` \
**Line:** 35 \
**Category:** Configuration / Environment Variable Handling

### Bug Description

The Express server reads the listening port from the `APP_PORT` environment variable but provides no fallback default. If the environment variable is missing or unset, `process.env.APP_PORT` evaluates to `undefined`. Node.js then attempts to call `app.listen(undefined, ...)`, which either crashes or binds to a random port, neither of which allows the ALB health check to reach the container on the expected port.

### Detailed Investigation

The server startup code at `server.js:35` was:

```javascript
const port = process.env.APP_PORT;

app.listen(port, () => {
  logger.info(`Example app listening on port ${port}`);
});
```

The infrastructure expects the container to listen on port 9000. This is confirmed across three Terraform files:

1. **`terraform/variables.tf:25-29`** defines the default:
   ```hcl
   variable "app_port" {
     description = "Port the application listens on"
     type        = number
     default     = 9000
   }
   ```

2. **`terraform/ecs.tf:65`** injects the variable into the container environment:
   
   
   environment = [
     { name = "APP_PORT", value = tostring(var.app_port) }
   ]
   ```

3. **`terraform/ecs.tf:58-63`** maps the container port:
   ```hcl
   portMappings = [
     { containerPort = var.app_port, hostPort = var.app_port, protocol = "tcp" }
   ]
   ```

4. **`terraform/alb.tf:15`** configures the target group to health-check on port 9000:
   ```hcl
   port     = var.app_port
   protocol = "HTTP"
   ```

When the ECS task definition is correctly configured, `APP_PORT=9000` is injected and everything works. However, if the environment variable is ever missing (e.g., local development, task definition misconfiguration, or running the container outside ECS), the server fails silently.

### Root Cause

Missing JavaScript logical OR (`||`) default value operator. The developer assumed `APP_PORT` would always be set by the infrastructure, but defensive coding requires a fallback.

### Code Diff

**Before (broken):**
```javascript
const port = process.env.APP_PORT;
```

**After (fixed):**
```javascript
const port = process.env.APP_PORT || 9000;
```

### Full Fixed Code Block (server.js:32-38)

```javascript
app.use('/v1/media', mediaRoutes);

const port = process.env.APP_PORT || 9000;

app.listen(port, () => {
  logger.info(`Example app listening on port ${port}`);
});
```

### Why This Fix Is Correct

- The fallback value of `9000` matches the Terraform-defined default (`variables.tf:27`)
- The ALB target group routes to port 9000 (`alb.tf:15`)
- The ECS port mapping uses port 9000 (`ecs.tf:60`)
- Using `||` means the fallback only activates when `APP_PORT` is falsy (undefined, empty string, null)
- When deployed in ECS with proper env vars, the fallback is never used, it only protects against misconfiguration

### Impact Without Fix

- Container starts on undefined port or crashes
- ALB health checks fail (expecting port 9000)
- ECS marks the task as unhealthy and restarts it in a loop
- Service becomes completely unavailable

---

## Ticket #2: Width Field Missing from GET /v1/media/:id Response

**Severity:** High
**File:** `clients/dynamodb.js`
**Line:** 84 (inside `getMedia` function)
**Category:** Data Retrieval / Incomplete Response

### Bug Description

When a client uploads an image with a specific `width` parameter (e.g., `?width=800`), the width is correctly saved to DynamoDB. However, when the client later retrieves the media metadata via `GET /v1/media/:id`, the width field is missing from the response. The `getMedia()` function reads the DynamoDB record but omits the `width` attribute from its return value.

### Detailed Investigation

**Step 1: Trace the upload path.** When a file is uploaded, `controllers/media.js:27` (`uploadController`) calls `createMedia()` in `clients/dynamodb.js:23`. The `createMedia` function saves the following attributes to DynamoDB:

```javascript
// clients/dynamodb.js:23-36 (createMedia)
const command = new PutItemCommand({
  TableName,
  Item: {
    PK: { S: `MEDIA#${mediaId}` },
    SK: { S: 'METADATA' },
    size: { N: size.toString() },
    name: { S: name },
    mimetype: { S: mimetype },
    status: { S: MEDIA_STATUS.PENDING },
    width: { N: String(width) },          // <-- width IS saved
  },
});
```

**Step 2: Trace the retrieval path.** When `GET /v1/media/:id` is called, `controllers/media.js:132` (`getController`) calls `getMedia()` in `clients/dynamodb.js:56`. The `getMedia` function retrieves the DynamoDB record and constructs a return object:

```javascript
// clients/dynamodb.js:78-84 (getMedia - BEFORE fix)
return {
  mediaId,
  size: Number(Item.size.N),
  name: Item.name.S,
  mimetype: Item.mimetype.S,
  status: Item.status.S,
  // width is NOT included here
};
```

The width attribute exists in the DynamoDB `Item` object (as `Item.width.N`), but it was never mapped into the return value.

### Evidence from Live API

```bash
# Upload with width=800
curl -X POST "http://hummingbird-production-alb-1340824329.us-west-2.elb.amazonaws.com/v1/media/upload?width=800" \
  -F "file=@sample.png"
{"mediaId":"89f6a00d-281b-426e-88d5-609c421866d0"}

# Retrieve metadata - width is missing
curl -X GET "http://hummingbird-production-alb-1340824329.us-west-2.elb.amazonaws.com/v1/media/89f6a00d-281b-426e-88d5-609c421866d0"
{
  "mediaId": "89f6a00d-281b-426e-88d5-609c421866d0",
  "size": 700210,
  "name": "sample.png",
  "mimetype": "image/png",
  "status": "PENDING"
}
# PROBLEM: No "width" field in response, even though upload specified width=800
```

### Root Cause

The developer who wrote `createMedia` remembered to persist the width field to DynamoDB, but the developer who wrote `getMedia` (or perhaps the same developer at a different time) forgot to include it in the return object. This is a classic "write-read asymmetry" bug, the data is stored correctly but not retrieved completely.

### Code Diff

**Before (broken):**
```javascript
return {
  mediaId,
  size: Number(Item.size.N),
  name: Item.name.S,
  mimetype: Item.mimetype.S,
  status: Item.status.S,
};
```

**After (fixed):**
```javascript
return {
  mediaId,
  size: Number(Item.size.N),
  name: Item.name.S,
  mimetype: Item.mimetype.S,
  status: Item.status.S,
  width: Number(Item.width.N),
};
```

### Full Fixed Function (clients/dynamodb.js:56-87)

```javascript
async function getMedia(mediaId) {
  try {
    const command = new GetItemCommand({
      TableName,
      Key: {
        PK: { S: `MEDIA#${mediaId}` },
        SK: { S: 'METADATA' },
      },
    });

    logger.info({ mediaId, sk: 'METADATA' });

    const client = new DynamoDBClient({
      region: process.env.AWS_REGION,
    });

    const { Item } = await client.send(command);

    if (!Item) {
      return null;
    }

    return {
      mediaId,
      size: Number(Item.size.N),
      name: Item.name.S,
      mimetype: Item.mimetype.S,
      status: Item.status.S,
      width: Number(Item.width.N),   // <-- ADDED: read width from DynamoDB
    };
  } catch (error) {
    logger.error(error);
    throw error;
  }
}
```

### Why This Fix Is Correct

- `Item.width.N` accesses the DynamoDB Number attribute that was saved by `createMedia`
- `Number()` converts the DynamoDB string representation back to a JavaScript number, matching the pattern used for `size`
- The field name `width` matches what `createMedia` stores and what the middleware (`setMediaWidth.js`) originally parsed from the request
- This makes `getMedia` symmetrical with `createMedia`, every field that is written is also read back

### Impact Without Fix

- Clients cannot determine what resize width was requested for an image
- Any downstream logic depending on width (e.g., UI displaying image dimensions) breaks
- The API contract is inconsistent: the upload endpoint accepts width, but the GET endpoint doesn't return it
- Violates the principle of least surprise, users expect stored data to be retrievable

---

## Ticket #3: Location Header Missing HTTP Protocol Prefix

**Severity:** High
**File:** `controllers/media.js`
**Line:** 111
**Category:** HTTP Standards Compliance / URL Construction

### Bug Description

When the download endpoint returns a `202 Accepted` response (because the media is still processing), it includes a `Location` header pointing the client to the status endpoint. However, the Location header is constructed without the `http://` protocol prefix, producing an invalid URL that clients cannot follow.

### Detailed Investigation

**Step 1: Read the download controller.** The `downloadController` function at `controllers/media.js:98-130` handles `GET /v1/media/:id/download`. When the media is not yet `COMPLETE`, it returns a 202 with headers telling the client to retry:

```javascript
// controllers/media.js:108-119 (BEFORE fix)
if (media.status !== MEDIA_STATUS.PROCESSING) {   // (also buggy, see Ticket #4)
  const SIXTY_SECONDS = 60;
  res.set('Retry-After', SIXTY_SECONDS);
  res.set('Location', `${req.hostname}/v1/media/${mediaId}/status`);  // <-- BUG
  logger.info(
    { mediaId, currentStatus: media.status },
    'Media not ready for download, sending 202'
  );
  return sendAcceptedResponse(res, { message: 'Media processing in progress.' });
}
```

**Step 2: Understand the Express API difference.** Express provides two ways to access the request host:

| Express Method | What It Returns | Example Input: `Host: example.com:9000` |
|---------------|----------------|----------------------------------------|
| `req.hostname` | Domain name only, port stripped | `example.com` |
| `req.get('host')` | Raw Host header value, port included | `example.com:9000` |

The code used `req.hostname`, which:
1. Strips the port number
2. Does not include a protocol scheme

**Step 3: Understand the HTTP specification.** Per RFC 7231 Section 7.1.2, the `Location` header field is used to refer to a specific resource in relation to the response. When used in a 2xx response, it should be an absolute URI so that clients can unambiguously resolve it.

An absolute URI requires: `scheme "://" authority path`

Example of a valid Location: `http://example.com/v1/media/abc/status`
Example of an invalid Location: `example.com/v1/media/abc/status` (missing scheme)

### Evidence from Live API

**Before fix:**
```bash
curl -X POST "http://hummingbird-production-alb-1860718987.us-west-2.elb.amazonaws.com/v1/media/upload?width=500" \
  -F "file=@sample.png"
{"mediaId":"952d98c7-04c8-4130-9c0a-633b99576805"}

curl -i http://hummingbird-production-alb-1860718987.us-west-2.elb.amazonaws.com/v1/media/952d98c7-04c8-4130-9c0a-633b99576805/download
HTTP/1.1 202 Accepted
Date: Sat, 07 Mar 2026 05:36:47 GMT
Content-Type: application/json; charset=utf-8
Content-Length: 43
Connection: keep-alive
X-Powered-By: Express
Retry-After: 60
Location: hummingbird-production-alb-1860718987.us-west-2.elb.amazonaws.com/v1/media/952d98c7-04c8-4130-9c0a-633b99576805/status
ETag: W/"2b-DVJ8jHebx01CN9tAyv9tXcsioNc"

{"message":"Media processing in progress."}
```

Notice the Location header: `hummingbird-production-alb-1860718987.us-west-2.elb.amazonaws.com/...`
This is missing the `http://` prefix. A browser or HTTP client library trying to follow this header would either error or misinterpret it as a relative path.

**After fix:**
```bash
curl -X POST "http://hummingbird-production-alb-876316285.us-west-2.elb.amazonaws.com/v1/media/upload?width=500" \
  -F "file=@sample.png"
{"mediaId":"a67a4aec-383f-48ec-8d0c-39bd27460225"}

curl -i http://hummingbird-production-alb-876316285.us-west-2.elb.amazonaws.com/v1/media/a67a4aec-383f-48ec-8d0c-39bd27460225/download
HTTP/1.1 202 Accepted
Date: Sat, 07 Mar 2026 06:10:54 GMT
Content-Type: application/json; charset=utf-8
Content-Length: 43
Connection: keep-alive
X-Powered-By: Express
Retry-After: 60
Location: http://hummingbird-production-alb-876316285.us-west-2.elb.amazonaws.com/v1/media/a67a4aec-383f-48ec-8d0c-39bd27460225/status
ETag: W/"2b-DVJ8jHebx01CN9tAyv9tXcsioNc"

{"message":"Media processing in progress."}
```

The Location header now reads: `http://hummingbird-production-alb-876316285.us-west-2.elb.amazonaws.com/...`
This is a valid absolute URI that any HTTP client can follow.

### Root Cause

The developer used `req.hostname` instead of `req.get('host')` and forgot to prepend the protocol scheme. This is a common Express.js mistake because `req.hostname` sounds like it should return everything needed for a URL, but it actually returns only the bare domain name with the port stripped off.

### Code Diff

**Before (broken):**
```javascript
res.set('Location', `${req.hostname}/v1/media/${mediaId}/status`);
```

**After (fixed):**
```javascript
res.set('Location', `http://${req.get('host')}/v1/media/${mediaId}/status`);
```

### Why This Fix Is Correct

- `req.get('host')` returns the full Host header value including port (e.g., `localhost:9000` in dev, or `alb-dns.amazonaws.com` behind ALB)
- Prepending `http://` produces a well-formed absolute URI
- Behind the ALB, the Host header reflects the ALB's DNS name (port 80 is implicit), so the URL resolves correctly for clients
- In local development on port 9000, `req.get('host')` returns `localhost:9000`, so the URL would be `http://localhost:9000/v1/media/.../status`, also correct

### Impact Without Fix

- HTTP clients that automatically follow Location headers would fail or misroute
- Automated retry logic in client SDKs would break
- API consumers would need to manually construct the status URL instead of following the header
- Violates RFC 7231 and industry conventions for 202 responses

---

## Ticket #4: Download Endpoint Never Serves Completed Files

**Severity:** Critical
**File:** `controllers/media.js`
**Line:** 108
**Category:** Business Logic / State Machine Error

### Bug Description

The download endpoint (`GET /v1/media/:id/download`) is supposed to return a `302` redirect to a presigned S3 URL when the media processing is complete. Instead, it always returns `202 Accepted` regardless of the media's status, because the status comparison logic is backwards.

### Detailed Investigation

**Step 1: Understand the media status state machine.**

The media lifecycle has four possible states, defined in `core/constants.js:22`:
```javascript
const MEDIA_STATUS = {
  PENDING: 'PENDING',       // Just uploaded, waiting for processing
  PROCESSING: 'PROCESSING', // Worker is actively processing
  COMPLETE: 'COMPLETE',     // Processing done, ready to download
  ERROR: 'ERROR',           // Processing failed
};
```

The only state in which a file should be served is `COMPLETE`. All other states mean the file is not yet ready.

**Step 2: Read the buggy condition.**

```javascript
// controllers/media.js:108 (BEFORE fix)
if (media.status !== MEDIA_STATUS.PROCESSING) {
```

This condition reads: "If the status is anything other than PROCESSING, return 202 (not ready)."

Let's trace through each possible status:

| Status | `!== PROCESSING` | Result | Correct? |
|--------|-----------------|--------|----------|
| PENDING | true | Returns 202 | Yes (not ready) |
| PROCESSING | false | Falls through to redirect | WRONG (still processing!) |
| COMPLETE | true | Returns 202 | WRONG (should redirect!) |
| ERROR | true | Returns 202 | Acceptable (not ready) |

The logic is inverted. It serves the file only when status is PROCESSING (when it's actively being worked on) and blocks it when COMPLETE (when it's actually ready).

**Step 3: Reproduce the bug.**

```bash
# Upload an image
curl -X POST "http://hummingbird-production-alb-876316285.us-west-2.elb.amazonaws.com/v1/media/upload?width=500" \
  -F "file=@sample.png"
{"mediaId":"a67a4aec-383f-48ec-8d0c-39bd27460225"}

# Synchronously resize it (sets status to COMPLETE)
curl -X PUT "http://hummingbird-production-alb-876316285.us-west-2.elb.amazonaws.com/v1/media/a67a4aec-383f-48ec-8d0c-39bd27460225/resize?width=500"
{"mediaId":"a67a4aec-383f-48ec-8d0c-39bd27460225","status":"COMPLETE"}

# Check the status (confirms COMPLETE)
curl http://hummingbird-production-alb-876316285.us-west-2.elb.amazonaws.com/v1/media/a67a4aec-383f-48ec-8d0c-39bd27460225/status
{"status":"PENDING"}
# NOTE: Status shows PENDING here due to the Bonus bug (SK case mismatch).
# The resize endpoint hardcodes "COMPLETE" in its response but the DynamoDB
# update targets the wrong item. See Bonus section for details.

# Try to download, should be 302 but returns 202
curl -i http://hummingbird-production-alb-876316285.us-west-2.elb.amazonaws.com/v1/media/a67a4aec-383f-48ec-8d0c-39bd27460225/download
HTTP/1.1 202 Accepted
Date: Sat, 07 Mar 2026 06:13:02 GMT
Content-Type: application/json; charset=utf-8
Content-Length: 43
Connection: keep-alive
X-Powered-By: Express
Retry-After: 60
Location: http://hummingbird-production-alb-876316285.us-west-2.elb.amazonaws.com/v1/media/a67a4aec-383f-48ec-8d0c-39bd27460225/status
ETag: W/"2b-DVJ8jHebx01CN9tAyv9tXcsioNc"

{"message":"Media processing in progress."}
```

The download endpoint returns `202` even after the resize was supposed to set status to `COMPLETE`.

### Root Cause

The developer confused which status to gate on. Instead of checking "is the file NOT complete?" they checked "is the file NOT processing?" This is a semantic error, the condition should guard against incomplete states, allowing only `COMPLETE` to pass through to the redirect.

### Code Diff

**Before (broken):**
```javascript
if (media.status !== MEDIA_STATUS.PROCESSING) {
```

**After (fixed):**
```javascript
if (media.status !== MEDIA_STATUS.COMPLETE) {
```

### Corrected Status Flow

After the fix:

| Status | `!== COMPLETE` | Result | Correct? |
|--------|---------------|--------|----------|
| PENDING | true | Returns 202 | Yes |
| PROCESSING | true | Returns 202 | Yes |
| COMPLETE | false | Falls through to 302 redirect | Yes |
| ERROR | true | Returns 202 | Acceptable |

### Full Fixed downloadController (controllers/media.js:98-130)

```javascript
async function downloadController(req, res) {
  try {
    const { id: mediaId } = req.params;

    const media = await getMedia(mediaId);
    if (!media) {
      return sendNotFoundResponse(res, 'Media not found');
    }

    if (media.status !== MEDIA_STATUS.COMPLETE) {  // FIXED: was PROCESSING
      const SIXTY_SECONDS = 60;
      res.set('Retry-After', SIXTY_SECONDS);
      res.set('Location', `http://${req.get('host')}/v1/media/${mediaId}/status`);  // FIXED: was req.hostname
      logger.info(
        { mediaId, currentStatus: media.status },
        'Media not ready for download, sending 202'
      );
      return sendAcceptedResponse(res, { message: 'Media processing in progress.' });
    }

    const url = await getProcessedMediaUrl({
      mediaId,
      mediaName: media.name,
    });

    return res.redirect(302, url);
  } catch (error) {
    logger.error(error);
    return sendInternalServerErrorResponse(res);
  }
}
```

### Design Pattern: Async Processing Status Gates

When implementing download endpoints for asynchronously processed resources:
1. Always gate on the **terminal success state** (COMPLETE), not an intermediate one
2. All non-success states should return a retry response (202 with Retry-After)
3. The condition should be `status !== SUCCESS` (inclusive deny), not `status !== INTERMEDIATE` (which lets unrelated states through)
4. Consider handling ERROR explicitly with a 4xx/5xx instead of lumping it with retry

### Impact Without Fix

- No client can ever download a completed file
- The entire download flow is broken, the core purpose of the API is defeated
- The presigned S3 URL generation code (lines 122-125) is unreachable dead code
- Users upload files successfully but can never retrieve them

---

## BONUS: DynamoDB Sort Key Case Mismatch (Silent Data Corruption)

**Severity:** Critical (Data Integrity)
**File:** `clients/dynamodb.js`
**Lines:** 155, 167
**Category:** Database Key Consistency / Silent Failure

### Bug Description

The `setMediaStatus()` function uses a lowercase Sort Key value `'metadata'`, while `createMedia()` and `getMedia()` use uppercase `'METADATA'`. Because DynamoDB keys are case-sensitive, status updates are written to a completely different (phantom) item in the database. The original item's status is never changed, it remains `PENDING` forever.

### Detailed Investigation

**Step 1: Map all DynamoDB key usage across the codebase.**

Three functions in `clients/dynamodb.js` interact with the same DynamoDB record using composite keys (PK + SK):

| Function | Line | PK Value | SK Value | Operation |
|----------|------|----------|----------|-----------|
| `createMedia` | 29 | `MEDIA#${mediaId}` | `METADATA` (uppercase) | PutItem (create) |
| `getMedia` | 62 | `MEDIA#${mediaId}` | `METADATA` (uppercase) | GetItem (read) |
| `setMediaStatus` | 155 | `MEDIA#${mediaId}` | `metadata` (lowercase) | UpdateItem (update) |

The SK values don't match. `createMedia` and `getMedia` use uppercase `METADATA`, but `setMediaStatus` uses lowercase `metadata`.

**Step 2: Understand DynamoDB key behavior.**

DynamoDB treats Partition Key and Sort Key values as raw binary data. String comparisons are byte-level, meaning `METADATA` and `metadata` are entirely different keys. They point to different items in the table.

When `setMediaStatus` sends an `UpdateItemCommand` with `SK: { S: 'metadata' }`:
- DynamoDB looks for an item with `PK=MEDIA#<id>` and `SK=metadata`
- No such item exists (the real item has `SK=METADATA`)
- DynamoDB's default behavior for UpdateItem is to **create the item if it doesn't exist** (upsert)
- So it silently creates a new phantom item with `SK=metadata` and sets its status
- The original item with `SK=METADATA` is untouched, its status stays `PENDING`

**Step 3: Confirm with CloudWatch logs.**

The logger calls in `dynamodb.js` conveniently log the SK value being used:

```
# Upload creates the item with uppercase SK
2026-03-07T06:38:02.138000+00:00 api/api/78ed7d1d5aee4eb8a79ea19c7e811e41
{"level":"info","message":{"mediaId":"5457fc51-04d7-4afa-aaeb-9919bfe26d30","sk":"METADATA"},
 "service":"hummingbird","timestamp":"2026-03-07T06:38:02.138Z"}

# setMediaStatus updates with lowercase SK - WRONG ITEM
2026-03-07T06:38:38.801000+00:00 api/api/78ed7d1d5aee4eb8a79ea19c7e811e41
{"level":"info","message":{"mediaId":"5457fc51-04d7-4afa-aaeb-9919bfe26d30",
 "newStatus":"PROCESSING","sk":"metadata"},
 "service":"hummingbird","timestamp":"2026-03-07T06:38:38.800Z"}

# Another setMediaStatus with lowercase SK - still wrong item
2026-03-07T06:38:39.065000+00:00 api/api/78ed7d1d5aee4eb8a79ea19c7e811e41
{"level":"info","message":{"mediaId":"5457fc51-04d7-4afa-aaeb-9919bfe26d30",
 "newStatus":"COMPLETE","sk":"metadata"},
 "service":"hummingbird","timestamp":"2026-03-07T06:38:39.064Z"}
```

Notice the difference: the first log shows `"sk":"METADATA"` (from `createMedia`), but the subsequent logs show `"sk":"metadata"` (from `setMediaStatus`). This is the smoking gun.

**Step 4: Reproduce the cascading failure.**

```bash
# Upload an image
curl -X POST "http://$ALB/v1/media/upload?width=500" -F "file=@sample.png"
{"mediaId":"5457fc51-04d7-4afa-aaeb-9919bfe26d30"}

# Trigger synchronous resize, response claims COMPLETE
curl -X PUT "http://$ALB/v1/media/5457fc51-04d7-4afa-aaeb-9919bfe26d30/resize?width=500"
{"mediaId":"5457fc51-04d7-4afa-aaeb-9919bfe26d30","status":"COMPLETE"}

# Check actual status, still PENDING because the update went to wrong item
curl http://$ALB/v1/media/5457fc51-04d7-4afa-aaeb-9919bfe26d30/status
{"status":"PENDING"}
```

The resize endpoint returns `"status":"COMPLETE"` because the response is hardcoded in `resizeController` (line 169), it doesn't read back from DynamoDB. But when we query the actual status endpoint, it reads from DynamoDB using `getMedia` (which uses uppercase `SK=METADATA`), and that item was never updated. So it still shows `PENDING`.

**Step 5: Visualize the data corruption in DynamoDB.**

After the resize operation, the DynamoDB table contains TWO items for the same media:

| PK | SK | status | Other fields |
|----|-----|--------|-------------|
| `MEDIA#5457fc51...` | `METADATA` | `PENDING` | size, name, mimetype, width |
| `MEDIA#5457fc51...` | `metadata` | `COMPLETE` | (only status, created by upsert) |

The first row is the real item (created by `createMedia`). The second row is the phantom item (created by `setMediaStatus`'s upsert). All reads go to the first row; all updates go to the second row.

### Root Cause

A typo or inconsistency in the Sort Key value. The developer used lowercase `'metadata'` in `setMediaStatus` while the rest of the codebase uses uppercase `'METADATA'`. Since there's no compile-time check or constant reference for the SK value (it's a raw string literal in each function), this mismatch went undetected.

### Code Diff

**Change 1 - UpdateItemCommand Key (line 155):**
```diff
  Key: {
    PK: { S: `MEDIA#${mediaId}` },
-   SK: { S: 'metadata' },
+   SK: { S: 'METADATA' },
  },
```

**Change 2 - Logger message (line 167):**
```diff
- logger.info({ mediaId, sk: 'metadata', newStatus }, 'Updating media status in DynamoDB');
+ logger.info({ mediaId, sk: 'METADATA', newStatus }, 'Updating media status in DynamoDB');
```

### Full Fixed setMediaStatus Function (clients/dynamodb.js:148-171)

```javascript
async function setMediaStatus({ mediaId, newStatus }) {
  try {
    const command = new UpdateItemCommand({
      TableName,
      Key: {
        PK: { S: `MEDIA#${mediaId}` },
        SK: { S: 'METADATA' },           // FIXED: was 'metadata' (lowercase)
      },
      UpdateExpression: 'SET #status = :newStatus',
      ExpressionAttributeNames: { '#status': 'status' },
      ExpressionAttributeValues: {
        ':newStatus': { S: newStatus },
      },
    });

    const ddbClient = new DynamoDBClient({
      region: process.env.AWS_REGION,
    });

    logger.info({ mediaId, sk: 'METADATA', newStatus }, 'Updating media status in DynamoDB');  // FIXED
    await ddbClient.send(command);
  } catch (error) {
    logger.error(error);
    throw error;
  }
}
```

### Why This Bug Is Particularly Dangerous

1. **No error signals:** DynamoDB doesn't throw an error when updating a non-existent item, it silently creates one. There are no error logs, no exceptions, and no failed health checks.

2. **Misleading responses:** The resize endpoint returns `"status":"COMPLETE"` because it's hardcoded in the controller response (not read back from DynamoDB), giving the false impression that everything worked.

3. **Log-level detection only:** The only way to spot this bug is by comparing the `"sk"` values across different log lines, `createMedia` logs `"sk":"METADATA"` and `setMediaStatus` logs `"sk":"metadata"`. There's no single error line that flags the problem.

4. **Data integrity violation:** The database now contains orphaned phantom items that are never read, wasting storage and creating confusion during debugging.

5. **Cascading with Ticket #4:** Even after fixing Ticket #4 (status comparison), downloads would still fail because `getMedia` reads the original item (which is stuck at PENDING). Both bugs must be fixed together for the download flow to work correctly.

### How This Should Be Prevented

- **Use constants:** Define `const SORT_KEY = 'METADATA'` once and reference it everywhere instead of using string literals
- **Write integration tests:** A test that creates, updates, and reads back a record would catch this immediately
- **Add a ConditionExpression:** Use `ConditionExpression: 'attribute_exists(PK)'` in the UpdateItemCommand to make it fail (instead of upsert) when the item doesn't exist

---

## Combined Impact Analysis

These five bugs interact with each other to create a completely broken download flow:

```
1. Ticket #1 (Port): If APP_PORT is missing, server doesn't start at all
   |
   v
2. Ticket #2 (Width): Upload works, but clients can't see their requested width
   |
   v
3. Bonus (SK mismatch): Worker/resize sets status, but update goes to phantom item
   |                     Real item stays PENDING forever
   v
4. Ticket #4 (Status gate): Even if status were COMPLETE, the condition checks
   |                         the wrong value and blocks download anyway
   v
5. Ticket #3 (Location header): Even the "retry later" response gives clients
                                  an invalid URL to poll
```

After all fixes:
- Server starts reliably on port 9000
- Upload saves and returns all metadata including width
- Status updates correctly target the real DynamoDB item
- Download endpoint correctly identifies COMPLETE status and redirects
- The retry Location header is a valid absolute URL

---

## Summary of All Fixes

| Ticket | File | Line(s) | Bug | Fix Applied |
|--------|------|---------|-----|-------------|
| #1 | server.js | 35 | No fallback port, server crashes if APP_PORT unset | Added `\|\| 9000` to match Terraform default |
| #2 | clients/dynamodb.js | 84 | getMedia omits width field that createMedia stores | Added `width: Number(Item.width.N)` to return object |
| #3 | controllers/media.js | 111 | Location header missing `http://` protocol prefix | Changed `req.hostname` to `http://${req.get('host')}` |
| #4 | controllers/media.js | 108 | Status gate checks `!== PROCESSING` instead of `!== COMPLETE` | Changed comparison to `!== MEDIA_STATUS.COMPLETE` |
| BONUS | clients/dynamodb.js | 155, 167 | SK uses lowercase `'metadata'` vs uppercase `'METADATA'` everywhere else | Changed to uppercase `'METADATA'` in both key and logger |

---

## Technical Learnings

### 1. Defensive Configuration
Always provide fallback values for critical environment variables. Infrastructure-as-code (Terraform) defines defaults, but application code should also be resilient to missing configuration. Use `process.env.VAR || default` or validation at startup.

### 2. Write-Read Symmetry
When persisting data to a database, ensure every field that is written is also read back. Review both the write path and the read path together. A mismatch means data is stored but inaccessible.

### 3. HTTP Header Construction in Express.js
Express provides multiple ways to access host information. Know the differences:
- `req.hostname` strips the port and returns only the domain
- `req.get('host')` returns the full Host header including port
- Neither includes the protocol scheme, always prepend `http://` or `https://` manually
- For production, consider using `req.protocol` to dynamically determine the scheme

### 4. State Machine Design
When implementing async processing with status polling:
- Gate download/access on the terminal success state (COMPLETE), not an intermediate one
- All other states (PENDING, PROCESSING, ERROR) should return a retry or error response
- Write the condition as `status !== SUCCESS` to be inclusive, any non-success state is blocked

### 5. DynamoDB Key Consistency
DynamoDB keys are binary-compared and case-sensitive. A single character case difference creates an entirely different item. Best practices:
- Define key values as constants and reference them everywhere
- Never use raw string literals for keys in multiple functions
- Use ConditionExpression to prevent silent upserts on non-existent items
- Write integration tests that exercise the full create-update-read cycle

### 6. Silent Failures Are the Most Dangerous Bugs
The bonus bug produced zero error logs. DynamoDB silently created a phantom item. The resize endpoint hardcoded its response. Everything appeared to work, but the data was split across two items. Bugs that fail silently are harder to detect than ones that throw errors. Design systems to fail loudly when invariants are violated.

---

**Report prepared:** March 7, 2026
**All code changes deployed and verified against live production API**
**ALB endpoint:** hummingbird-production-alb-*.us-west-2.elb.amazonaws.com
