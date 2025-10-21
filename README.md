# Login System - Next.js + Go + PostgreSQL

A complete authentication system built with Next.js frontend, Go backend, and PostgreSQL database.

## Features

- User registration and login
- JWT-based authentication
- Password hashing with bcrypt
- Protected routes
- Responsive UI with Tailwind CSS
- TypeScript support

## Prerequisites

- Go 1.21 or later
- Node.js 18 or later
- PostgreSQL 12 or later

## Setup Instructions

### 1. Database Setup

First, create a PostgreSQL database:

```sql
CREATE DATABASE loginapp;
```

Then run the schema:

```bash
cd backend
psql -d loginapp -f schema.sql
```

### 2. Backend Setup

```bash
cd backend
go mod tidy
```

Edit `backend/.env` with your database credentials:

```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=loginapp
JWT_SECRET=your-super-secret-jwt-key-change-in-production
PORT=8080
```

Start the backend server:

```bash
go run app.go
```

The API will be available at `http://localhost:8080`

### 3. Frontend Setup

```bash
cd frontend
npm install
```

Edit `frontend/.env.local`:

```env
NEXT_PUBLIC_API_URL=http://localhost:8080
```

Start the frontend:

```bash
npm run dev
```

The frontend will be available at `http://localhost:3000`

## API Endpoints

- `POST /api/register` - User registration
- `POST /api/login` - User login
- `GET /api/profile` - Get user profile (requires authentication)

## Project Structure

```
.
├── backend/
│   ├── app.go          # Main Go application
│   ├── go.mod          # Go dependencies
│   ├── schema.sql      # Database schema
│   └── .env            # Environment variables
├── frontend/
│   ├── src/
│   │   ├── app/        # Next.js app router pages
│   │   ├── components/ # React components
│   │   ├── contexts/   # React contexts
│   │   └── lib/        # Utility functions
│   ├── package.json    # Node.js dependencies
│   └── .env.local      # Environment variables
└── README.md
```

## Usage

1. Visit `http://localhost:3000`
2. Register a new account or login with existing credentials
3. Access the protected dashboard after authentication
4. Logout to return to the login page

## Technologies Used

### Backend
- Go
- Gorilla Mux (HTTP router)
- PostgreSQL with lib/pq driver
- JWT for authentication
- bcrypt for password hashing
- CORS support

### Frontend
- Next.js 15 with App Router
- TypeScript
- Tailwind CSS
- Axios for API calls
- React Context for state management

## Security Features

- Password hashing with bcrypt
- JWT tokens with expiration
- Protected API routes
- CORS configuration
- Input validation
- Secure headers
