# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is **private-isu**, a practice environment for ISUCON (Japanese web performance competition). It's a social media-like web application with multiple language implementations designed for learning web performance optimization.

## Essential Setup Commands

Before working with any implementation, initialize the project:
```bash
make init  # Downloads initial data (dump.sql.bz2 and image files)
```

## Language Implementations

The webapp has 5 different language implementations of the same application:

- **Ruby** (default): Sinatra + Unicorn
- **Go**: chi router + sqlx + gomemcache
- **PHP**: Slim framework + PHP-FPM
- **Python**: Flask + gunicorn
- **Node.js/TypeScript**: Hono framework + @hono/node-server

## Common Development Commands

### Building Applications

**Go implementation:**
```bash
cd webapp/golang
make  # builds to ./app
```

**Node.js implementation:**
```bash
cd webapp/node
npm install
npm start  # or npm run dev for development
```

**Benchmarker:**
```bash
cd benchmarker
make  # builds to ./bin/benchmarker
```

### Running Applications

**Docker Compose (recommended for development):**
```bash
cd webapp
docker compose up
```

**Local Ruby development:**
```bash
cd webapp/ruby
bundle install --path=vendor/bundle
bundle exec unicorn -c unicorn_config.rb
```

**Running benchmarker:**
```bash
cd benchmarker
./bin/benchmarker -t "http://localhost:8080" -u ./userdata
```

**Expected output format:**
```json
{"pass":true,"score":1710,"success":1434,"fail":0,"messages":[]}
```

## Architecture Overview

### Application Structure
- **Database**: MySQL 8.4 with users, posts, comments tables
- **Cache**: Memcached for session storage
- **Web Server**: Nginx as reverse proxy
- **Images**: Stored as BLOB in database (performance optimization target)

### Key Performance Bottlenecks
The application is intentionally designed with performance issues:
- Images stored in database as BLOBs
- N+1 query problems in timeline generation
- No database indexing optimization
- Session data stored in database instead of cache

### Database Schema
Main tables:
- `users`: User accounts and authentication
- `posts`: Image posts with binary data
- `comments`: User comments on posts

### Core Application Features
- User registration and authentication
- Image upload and display
- Timeline feed with posts and comments
- User profile pages
- Admin functionality

## Development Environment Options

1. **Docker Compose**: Full stack with all services
2. **Vagrant**: VM-based development with Ansible provisioning
3. **Local development**: Direct installation of dependencies
4. **Cloud deployment**: AWS AMI or cloud-init scripts

## File Structure
```
├── benchmarker/          # Go-based load testing tool
├── webapp/              # Multi-language implementations
│   ├── golang/          # Go implementation
│   ├── ruby/            # Ruby implementation (default)
│   ├── php/             # PHP implementation
│   ├── python/          # Python implementation
│   ├── node/            # Node.js/TypeScript implementation
│   └── sql/             # Database initialization
├── provisioning/        # Ansible playbooks
└── manual.md           # Competition manual
```

## Performance Optimization Guidelines

Common optimization targets:
1. Move image storage from database to filesystem/object storage
2. Add database indexes for timeline queries
3. Implement proper caching strategies
4. Optimize N+1 queries with JOIN operations
5. Use CDN for static assets
6. Implement connection pooling

## Language-Specific Notes

### Go Implementation (webapp/golang/)
- Uses `chi` router, `sqlx` for database, `gomemcache` for caching
- Binary: `./app`
- Database connection via environment variables

### Ruby Implementation (webapp/ruby/)
- Uses `Sinatra` framework with `Unicorn` server
- Session management via `Rack::Session::Memcache`
- Start with: `bundle exec unicorn -c unicorn_config.rb`

### PHP Implementation (webapp/php/)
- Uses `Slim` framework with `PHP-FPM`
- Requires nginx configuration change for routing
- Database access via `PDO`

### Python Implementation (webapp/python/)
- Uses `Flask` framework with `gunicorn`
- Dependencies managed via `uv` (pyproject.toml)
- Memcached integration via `python-memcached`

### Node.js Implementation (webapp/node/)
- Uses `Hono` framework with `@hono/node-server`
- TypeScript-based implementation
- Dependencies: `mysql2`, `multer`, `ejs`
- Start with: `npm start` or `npm run dev`

## Environment Variables

Key environment variables for applications:
- `ISUCONP_DB_HOST`: Database hostname
- `ISUCONP_DB_PORT`: Database port (3306)
- `ISUCONP_DB_USER`: Database user
- `ISUCONP_DB_PASSWORD`: Database password
- `ISUCONP_DB_NAME`: Database name (isuconp)
- `ISUCONP_MEMCACHED_ADDRESS`: Memcached server address