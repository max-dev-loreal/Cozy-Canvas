# Cozy Canvas Notes 🌸

Cozy Canvas is a beautiful, interactive, visual note-taking application designed for engineers. It combines a free-form digital canvas with a dynamic force-directed physics graph to organize notes, track compositor environment variables, and link nodes together.

## 🏗️ Project Architecture

This project is structured as a monorepo consisting of:
*   **Frontend (`/website`):** A modern, high-performance web interface built with HTML5, vanilla HSL CSS, and JavaScript, utilizing `d3.js` for realtime graph physics simulations.
*   **Backend (`/backend`):** A lightweight, secure REST API backend built in Go.
*   **Database (`/migrations`):** PostgreSQL database schema with strict foreign keys and cascading delete constraints.
*   **Object Storage (`/infrastructure`):** S3-compatible MinIO storage service used to store note attachments directly from the client via presigned URLs.

```
├── backend/          # Go REST API backend
├── website/          # Vanilla JS/CSS Vite frontend
├── migrations/       # SQL database migrations
├── infrastructure/   # Docker-compose files & initialization scripts
└── Makefile          # Unified development command panel
```

---

## 🔒 Security & Core Features

1.  **Plaintext-Free Passwords:** User passwords are encrypted using `bcrypt` before storage. Plaintext passwords never hit the database.
2.  **JWT Authentication:** All API requests to protected endpoints (like notes, connections, and files) are authorized via JSON Web Tokens (`Authorization: Bearer <token>`).
3.  **Role-Based Access Control (RBAC):** Users can share read-only access to their notes with colleagues using owner email, password, and codewords. Write requests are strictly restricted to the resource owner.
4.  **Client-Direct S3 Uploads (Presigned URLs):** Uploaded attachments are sent directly from the browser to MinIO using presigned PUT URLs, meaning large binaries never pass through or consume backend server memory.

---

## 🚀 Getting Started

### Prerequisites

Make sure you have the following installed on your machine:
*   [Docker Desktop](https://www.docker.com/) (including Docker Compose)
*   [Go](https://go.dev/) (1.22+ or 1.25+)
*   [Node.js & npm](https://nodejs.org/)
*   [Make](https://www.gnu.org/software/make/) (e.g. `winget install GnuWin32.Make` on Windows)

### Running Locally

1.  **Configure Environment Variables:**
    Copy the template env file:
    ```bash
    cp infrastructure/docker/.env.example infrastructure/docker/.env
    ```

2.  **Start Databases (PostgreSQL + MinIO):**
    ```bash
    make dev
    ```

3.  **Apply Database Migrations:**
    ```bash
    make migrate
    ```

4.  **Run Go Backend:**
    ```bash
    make backend
    ```

5.  **Run Web Frontend:**
    ```bash
    make website
    ```
    Open [http://localhost:5173](http://localhost:5173) in your browser.

---

## 🧪 Testing

The API includes a suite of integration testing scripts to verify authorization, note creation, and access grants.

To run the automated test suite:
```bash
make test
```
