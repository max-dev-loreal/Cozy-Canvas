# ADR 004: Stateless File Storage and Client-Direct Uploads in Cozy Canvas

## Status
Accepted (Approved)

## Context
Cozy Canvas allows users to attach files (images, documents) to notes. Processing file uploads on a typical server requires handling:
- High network bandwidth.
- Substantial memory buffer allocations (buffering files in RAM).
- Local storage configurations (handling disk fill-ups, backup synchronization, and lack of horizontal scalability).

To maintain high API performance and keep the Go backend lightweight, we needed a scalable storage architecture.

## Decision
We decided to decouple file storage from the application server by using **MinIO (an S3-compatible object storage service)** and implementing a **Client-Direct Upload/Download architecture using Presigned URLs**:
1. **Private S3 Bucket**: All files are stored in a private bucket configured with strict access policies. No public/anonymous access is permitted.
2. **Presigned Upload (PUT) URLs**: When a user uploads a file, the frontend requests an upload URL from the Go API. The Go backend uses the MinIO SDK (`github.com/minio/minio-go/v7`) to generate a temporary, signed upload URL (HTTP PUT method) valid for a short duration (e.g., 15 minutes). The frontend then uploads the file directly to the S3-compatible storage using this URL.
3. **Presigned Download (GET) URLs**: When displaying file attachments, the backend generates temporary, signed download URLs (HTTP GET method) so the frontend can securely stream the files directly from the storage server.
4. **Stateless Backend**: The Go backend never receives, processes, or buffers the raw file bytes in its own memory space. It only handles lightweight metadata coordinates (S3 object keys, file names) stored in PostgreSQL.

## Justification
1. **Elimination of Backend Memory Pressure**:
   By bypassing the Go backend for the actual data transfer, we prevent the API server from exhausting memory (Out of Memory/OOM errors) during large or concurrent file uploads.
2. **Horizontal Scalability**:
   The Go API remains stateless and lightweight. It does not require persistent disk storage, allowing us to spin up multiple instances of the backend container seamlessly behind a load balancer.
3. **Security (Access Delegation)**:
   Because the bucket is private, files are not exposed to the public internet. Access is only delegated temporarily to authenticated users through signed cryptographic hashes in the URL query string.

## Consequences
- **Pros**:
  - Improved backend throughput and drastically reduced RAM usage.
  - Scalable storage decoupled from the compute nodes.
  - Secure data storage with time-limited access tokens.
- **Cons**:
  - Dual-phase Upload: The frontend has to make two calls (one to get the presigned URL, and one to upload the file). This increases frontend codebase complexity slightly but is mitigated by robust API error handling.
  - Clock Synchronization: Presigned URL verification relies on correct system times on both the Go backend and the S3 storage node.
