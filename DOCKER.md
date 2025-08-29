# Docker Deployment - PodCache

## Dockerfile Fix Applied

### Problem
The original Dockerfile was trying to copy both `go.mod` and `go.sum`, but `go.sum` doesn't exist because the project has no external dependencies.

### Solution
Modified the COPY command to handle missing `go.sum` gracefully:

```dockerfile
# Copy go module files (go.sum optional)
COPY go.mod* go.sum* ./
```

## Docker Build & Run

### Build Image
```bash
docker build -t podcache:latest .
```

### Run Container
```bash
# Basic run
docker run -p 6379:6379 podcache:latest

# With persistent cache data
docker run -p 6379:6379 -v podcache-data:/home/podcache/.cas podcache:latest

# With environment variables
docker run -p 6379:6379 \
  -e PODCACHE_PORT=6379 \
  -e PODCACHE_PARTITIONS=3 \
  -e PODCACHE_CAPACITY_MB=100 \
  podcache:latest
```

## Security Improvements

- ✅ Runs as non-root user (`podcache:1001`)
- ✅ Minimal Alpine base image
- ✅ No unnecessary packages
- ✅ Optimized binary with stripped symbols (`-ldflags="-w -s"`)

## Configuration

The container respects these environment variables:
- `PODCACHE_PORT` - Server port (default: 6379)
- `PODCACHE_PARTITIONS` - Number of cache partitions (default: 3)
- `PODCACHE_CAPACITY_MB` - Total cache capacity in MB (default: 100)

## Cache Persistence

Cache data is stored in `/home/podcache/.cas` and can be persisted using Docker volumes.

## Health Check

Test if the container is running:
```bash
# Connect with Redis CLI
redis-cli -h localhost -p 6379 ping

# Or use netcat
echo "PING" | nc localhost 6379
```
