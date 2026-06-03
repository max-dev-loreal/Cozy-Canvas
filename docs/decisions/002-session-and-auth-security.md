# ADR 002: Session and Authentication Security in Cozy Canvas

## Status
Accepted (Approved)

## Context
The Cozy Canvas system allows users to create personal boards/canvases, add interactive notes, and connect notes. However, because notes can be shared and canvases represent personal or collaborated workspaces, we required:
1. Secure credential storage.
2. High-performance, stateless user session verification for REST API endpoints.
3. Secure, granular authorization mechanisms to control read-only or read-write access to foreign notes when notes are connected across different users.

## Decision
We implemented a multi-layered security system consisting of:
1. **Adaptive Password Hashing**: Passwords stored in the database are hashed using **bcrypt** (cost parameter 10). Plaintext passwords are never logged, stored, or processed beyond the authentication endpoint.
2. **Stateless JWT Tokens**: Upon successful authentication, the backend generates a JSON Web Token (JWT) signed using a secure symmetric HS256 key (`JWT_SECRET`). The token payload contains the user's ID (`user_id`) and name (`username`). 
3. **HTTP Context Propagation**: A middleware inspects the `Authorization: Bearer <token>` header, validates the signature, extracts the claims, and injects them directly into the Go `http.Request` context for downstream handlers.
4. **Temporary Access Grants**: For accessing connected foreign notes, we implemented a dedicated DB table tracking explicit read-only access grants, verified via middleware before accessing notes owned by other users.

## Justification
1. **Defense Against Brute-force & Rainbow Table Attacks**:
   Bcrypt uses a configurable work factor and a unique salt per password, protecting stored credentials even if the underlying database is compromised.
2. **Stateless Scalability**:
   Using stateless JWTs allows the REST API server to verify requests without querying a database or session cache (like Redis) on every API request. This reduces API latency and database load.
3. **Granular Access Control (Least Privilege)**:
   Instead of giving full canvas access, the access grant mechanism allows temporary, read-only permissions for specifically linked canvas notes.

## Consequences
- **Pros**:
  - Secure credential storage resilient to database leakage.
  - Reduced database load for authenticated requests due to stateless token verification.
  - Clean separation of authentication (JWT middleware) and authorization (access grant queries).
- **Cons**:
  - Token Revocation: Since tokens are stateless, they cannot be easily invalidated before their expiration time without introducing a revocation registry. We mitigated this by setting appropriate token lifetimes.
  - Secret Key Management: The security of the JWT relies entirely on `JWT_SECRET`. It must be stored securely (e.g., in Vercel environment variables or Docker secrets) and rotated periodically.
