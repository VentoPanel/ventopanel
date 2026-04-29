// Package template provides a static catalog of pre-built Dockerfile
// templates for common web frameworks. Templates are baked into the binary —
// no database table required. The only thing stored in the DB is the
// chosen template slug per site (sites.template_id).
package template

// Template describes one framework template.
type Template struct {
	ID          string   `json:"id"`           // e.g. "nextjs"
	Name        string   `json:"name"`         // e.g. "Next.js"
	Description string   `json:"description"`
	Runtime     string   `json:"runtime"`      // "node" | "php" | "python" | "go" | "static"
	Tags        []string `json:"tags"`
	// Dockerfile is written to the app directory on the server before
	// `docker build` runs. It intentionally overrides any Dockerfile from
	// the repo when the user has chosen a template.
	Dockerfile      string `json:"dockerfile"`
	HealthcheckPath string `json:"healthcheck_path"` // suggested default
}

// Catalog is the full list of bundled templates.
var Catalog = []Template{
	{
		ID:          "nextjs",
		Name:        "Next.js",
		Description: "Production-ready Next.js app. Runs `npm run build` then starts the server. Works with any Next.js 13+ project (App Router or Pages Router).",
		Runtime:     "node",
		Tags:        []string{"node", "react", "ssr", "next"},
		HealthcheckPath: "/",
		Dockerfile: `FROM node:20-alpine AS deps
WORKDIR /app
COPY package*.json ./
RUN npm ci

FROM node:20-alpine AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

FROM node:20-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
COPY --from=builder /app/public ./public
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
EXPOSE 3000
ENV PORT=3000
ENV HOSTNAME=0.0.0.0
CMD ["node", "server.js"]
`,
	},
	{
		ID:          "node-express",
		Name:        "Node.js / Express",
		Description: "Generic Node.js application (Express, Fastify, Koa, etc.). Installs production dependencies and runs `npm start`.",
		Runtime:     "node",
		Tags:        []string{"node", "express", "fastify", "api"},
		HealthcheckPath: "/",
		Dockerfile: `FROM node:20-alpine
WORKDIR /app
COPY package*.json ./
RUN npm ci --omit=dev
COPY . .
EXPOSE 3000
ENV NODE_ENV=production
CMD ["npm", "start"]
`,
	},
	{
		ID:          "react-vite",
		Name:        "React / Vite (SPA)",
		Description: "Builds a Vite/CRA single-page app with `npm run build` and serves the static output via Nginx.",
		Runtime:     "node",
		Tags:        []string{"node", "react", "vite", "spa", "static"},
		HealthcheckPath: "/",
		Dockerfile: `FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY --from=builder /app/dist /usr/share/nginx/html
RUN echo 'server { listen 3000; root /usr/share/nginx/html; index index.html; location / { try_files $uri $uri/ /index.html; } }' > /etc/nginx/conf.d/default.conf
EXPOSE 3000
CMD ["nginx", "-g", "daemon off;"]
`,
	},
	{
		ID:          "laravel",
		Name:        "Laravel",
		Description: "Laravel PHP application using Composer for dependencies and the built-in Artisan server for development, or FrankenPHP for production.",
		Runtime:     "php",
		Tags:        []string{"php", "laravel", "composer"},
		HealthcheckPath: "/",
		Dockerfile: `FROM php:8.3-fpm-alpine AS base

RUN apk add --no-cache \
    nginx curl unzip git \
    libzip-dev libpng-dev libjpeg-turbo-dev freetype-dev \
    oniguruma-dev && \
    docker-php-ext-configure gd --with-freetype --with-jpeg && \
    docker-php-ext-install pdo_mysql mbstring zip gd opcache

COPY --from=composer:2 /usr/bin/composer /usr/bin/composer

WORKDIR /var/www/html

COPY . .

RUN composer install --no-dev --optimize-autoloader --no-interaction

RUN cp .env.example .env 2>/dev/null || true && \
    php artisan key:generate --force 2>/dev/null || true && \
    php artisan config:cache 2>/dev/null || true && \
    php artisan route:cache 2>/dev/null || true

RUN chown -R www-data:www-data storage bootstrap/cache

RUN printf 'server {\n  listen 3000;\n  root /var/www/html/public;\n  index index.php;\n  location / { try_files $uri $uri/ /index.php?$query_string; }\n  location ~ \\.php$ { fastcgi_pass 127.0.0.1:9000; fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name; include fastcgi_params; }\n}\n' > /etc/nginx/http.d/default.conf

EXPOSE 3000

CMD sh -c "php-fpm -D && nginx -g 'daemon off;'"
`,
	},
	{
		ID:          "python-fastapi",
		Name:        "Python / FastAPI",
		Description: "FastAPI application served with Uvicorn. Reads dependencies from requirements.txt. Entry point is `main:app`.",
		Runtime:     "python",
		Tags:        []string{"python", "fastapi", "uvicorn", "api"},
		HealthcheckPath: "/health",
		Dockerfile: `FROM python:3.12-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
EXPOSE 3000
ENV PORT=3000
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "3000"]
`,
	},
	{
		ID:          "python-django",
		Name:        "Python / Django",
		Description: "Django application served with Gunicorn. Reads dependencies from requirements.txt. Entry point is `wsgi:application` (adjust to your project name).",
		Runtime:     "python",
		Tags:        []string{"python", "django", "gunicorn"},
		HealthcheckPath: "/",
		Dockerfile: `FROM python:3.12-slim
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends gcc libpq-dev && rm -rf /var/lib/apt/lists/*
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt gunicorn psycopg2-binary
COPY . .
RUN python manage.py collectstatic --noinput 2>/dev/null || true
EXPOSE 3000
CMD ["gunicorn", "wsgi:application", "--bind", "0.0.0.0:3000", "--workers", "2"]
`,
	},
	{
		ID:          "go-generic",
		Name:        "Go",
		Description: "Multi-stage Go build. Compiles a minimal static binary and runs it in a scratch container.",
		Runtime:     "go",
		Tags:        []string{"go", "golang", "api"},
		HealthcheckPath: "/health",
		Dockerfile: `FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server ./...

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/server ./server
EXPOSE 3000
ENV PORT=3000
CMD ["./server"]
`,
	},
	{
		ID:          "static-nginx",
		Name:        "Static HTML (Nginx)",
		Description: "Serves a folder of static files (HTML, CSS, JS) via Nginx. No build step required. Ideal for landing pages and documentation.",
		Runtime:     "static",
		Tags:        []string{"static", "nginx", "html"},
		HealthcheckPath: "/",
		Dockerfile: `FROM nginx:1.27-alpine
COPY . /usr/share/nginx/html
RUN echo 'server { listen 3000; root /usr/share/nginx/html; index index.html index.htm; location / { try_files $uri $uri/ =404; } gzip on; gzip_types text/plain text/css application/javascript; }' > /etc/nginx/conf.d/default.conf
EXPOSE 3000
CMD ["nginx", "-g", "daemon off;"]
`,
	},
}

// ByID looks up a template by its slug. Returns nil if not found.
func ByID(id string) *Template {
	for i := range Catalog {
		if Catalog[i].ID == id {
			return &Catalog[i]
		}
	}
	return nil
}
