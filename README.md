# Station

A small Nextcloud-style depot for an **existing AWS S3 bucket**.

The browser talks to S3 with **presigned URLs**. Uploaded files never land on this server. Folder listings are cached in **Postgres**. Sessions live in **Redis**.

## What it does

- Sign in with `AUTH_USERNAME` / `AUTH_PASSWORD`
- Browse, create folders, upload, preview images/audio/video
- **Trash** moves objects to `.station-trash/` in *your* bucket (nothing is removed from S3)
- **Settings → Delete forever / Empty trash** is the only path that actually deletes objects
- **Reload from bucket** drops that folder’s cache and lists S3 again
- Opening a folder that is not cached always reads the bucket

## Run

```bash
cp .env.example .env
# fill AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_REGION, S3_BUCKET
docker compose up --build
```

Open [http://localhost:8080](http://localhost:8080).

The app container runs [Air](https://github.com/air-verse/air) with the repo mounted, so edits to `.go`, `.templ`, `.css`, and `.js` regenerate templ and restart the server.

Compose only starts **Postgres**, **Redis**, and the app. It does not create a bucket.

Without Docker for the app (Task starts Postgres and Redis, then Air):

```bash
task dev
```

Other Task commands: `task generate`, `task server`, `task kill`, `task test:go`, `task docker:up`.

## S3 CORS

Browser uploads are `PUT`s to the presigned URL, so the bucket needs CORS. Example:

```json
[
  {
    "AllowedOrigins": ["http://localhost:8080"],
    "AllowedMethods": ["GET", "PUT", "HEAD"],
    "AllowedHeaders": ["*"],
    "ExposeHeaders": ["ETag", "Content-Length"],
    "MaxAgeSeconds": 3000
  }
]
```

## Local Go

```bash
task generate
task server
```

Needs Postgres and Redis on the URLs in `.env`, plus working AWS credentials for the bucket.
