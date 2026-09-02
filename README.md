# AI Candidate Screening & ATS Platform

A full-stack Applicant Tracking System (ATS) with AI-powered candidate screening, semantic matching, resume management, recruiter workflows, and analytics.

---

## Features

- AI-powered candidate screening
- Semantic resume matching
- Resume upload & management
- Job and candidate management
- Recruiter dashboard
- Analytics & notifications
- JWT Authentication
- Docker Compose support

---

## Tech Stack

| Layer | Technology |
|--------|------------|
| Backend | Go, Gin |
| Frontend | Next.js, React, TypeScript |
| Database | PostgreSQL + pgvector |
| AI | Gemini / OpenAI |
| Containerization | Docker Compose |

---

## Project Structure

```text
.
├── backend/
├── frontend/
├── docker-compose.yml
├── .env.example
└── README.md
```

---

## Prerequisites

- Docker & Docker Compose
- Git

---

## Quick Start

Clone the repository

```bash
git clone <repository-url>
cd ai-ats-platform
```

Create environment file

```bash
cp .env.example .env
```

Start the complete application

```bash
docker compose up --build
```

Stop the application

```bash
docker compose down
```

---

## Application URLs

| Service | URL |
|----------|-----|
| Frontend | http://localhost:3000 |
| Backend API | http://localhost:8000 |
| Health Check | http://localhost:8000/health |
| PostgreSQL | localhost:5435 |
| Adminer | http://localhost:8081 |

---

## Running Without Docker

### Backend

```bash
cd backend
go mod download
go run ./cmd/api
```

### Frontend

```bash
cd frontend
npm install
npm run dev
```

---

## Environment Variables

Create a `.env` file from `.env.example`.

Important variables:

- `DATABASE_URL`
- `JWT_SECRET`
- `API_BACKEND_URL`
- `GEMINI_API_KEY`
- `OPENAI_API_KEY`

---

## Docker

Build and start all services

```bash
docker compose up --build
```

Rebuild after changes

```bash
docker compose build --no-cache
docker compose up
```

Stop containers

```bash
docker compose down
```

---

## Tech Highlights

- AI Candidate Screening
- Semantic Search using pgvector
- Resume Parsing
- Recruiter Dashboard
- REST API
- JWT Authentication
- Dockerized Full Stack Application

---

## License

This project was developed for educational and assessment purposes.