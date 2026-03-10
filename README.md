# searchable-from-book (Next.js + Go/Gin)

โปรเจกต์นี้ถูกจัดโครงสร้างให้รองรับ 2 แบบชัดเจนแล้ว:

- **local dev** → ใช้ `docker-compose.dev.yml` + `.env.dev`
- **cluster / Argo CD** → ใช้ `k8s/base` + `k8s/overlays/dev|prod` + Vault Agent ที่ backend

## โครงสร้างหลัก

- `backend/` → Go/Gin API
- `frontend/` → Next.js app
- `docker-compose.dev.yml` → local dev stack (postgres + api + web)
- `k8s/base/` → manifest กลางที่ใช้ร่วมกัน
- `k8s/overlays/dev/` → ค่าเฉพาะ dev namespace/domain/image tag
- `k8s/overlays/prod/` → ค่าเฉพาะ prod namespace/domain/image tag
- `argocd/dev/` → Argo CD Application สำหรับ dev
- `argocd/prod/` → Argo CD Application สำหรับ prod
- `docs/vault/` → ตัวอย่างการผูก HashiCorp Vault Agent

## Data files (Excel / JSON)

Put your files here:
- `backend/data/items.xlsx`
- `backend/data/kits.xlsx`
- `backend/data/rules.json`
- `backend/data/synonyms.json` (optional)

## Run local dev ด้วย Docker Compose

คัดลอก env ตัวอย่างก่อน:

```bash
cp backend/.env.dev.example backend/.env.dev
cp frontend/.env.dev.example frontend/.env.dev
```

จากนั้นรัน:

```bash
docker compose -f docker-compose.dev.yml up --build
```

เปิดใช้งาน:
- frontend: http://localhost:3000
- backend: http://localhost:8080

## Run local dev แบบไม่ใช้ Docker

Backend:

```bash
cd backend
cp .env.example .env
go mod tidy
go run ./cmd/server
```

Frontend:

```bash
cd frontend
cp .env.dev.example .env.local
npm install
npm run dev
```

## Deploy ด้วย Argo CD

dev:

```bash
kubectl apply -f argocd/dev/postgres-app.yaml
kubectl apply -f argocd/dev/backend-app.yaml
kubectl apply -f argocd/dev/frontend-app.yaml
```

prod:

```bash
kubectl apply -f argocd/prod/postgres-app.yaml
kubectl apply -f argocd/prod/backend-app.yaml
kubectl apply -f argocd/prod/frontend-app.yaml
```

## แนวคิดที่ใช้

- backend ใช้ Vault Agent สำหรับ secret จริง เช่น `DATABASE_URL`
- frontend ไม่เก็บ secret ใน Vault เพราะ `NEXT_PUBLIC_*` เป็น public config
- local dev ไม่บังคับให้มี Vault เพื่อไม่ให้การพัฒนายุ่งเกินจำเป็น
- prod/dev บน cluster ใช้ overlays แยกกันเพื่อเปลี่ยน namespace, host, image tag, secret path ได้ง่าย

รายละเอียดเพิ่มดูที่ `k8s/README.md`
