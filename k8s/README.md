# Kubernetes + Argo CD layout

ตอนนี้โปรเจกต์ถูกแยกเป็น **base + overlays** เรียบร้อยแล้ว

## โครงสร้าง

- `k8s/base/backend` → manifest กลางของ API
- `k8s/base/frontend` → manifest กลางของ web + ingress
- `k8s/base/postgres` → manifest กลางของ PostgreSQL
- `k8s/overlays/dev/*` → patch สำหรับ dev
- `k8s/overlays/prod/*` → patch สำหรับ prod
- `argocd/dev/*` → Argo CD Application ของ dev
- `argocd/prod/*` → Argo CD Application ของ prod

## ทำไมต้องแยกแบบนี้

- manifest กลางอยู่ที่เดียว แก้ครั้งเดียวแล้ว reuse ได้
- dev กับ prod ต่างกันเฉพาะค่าที่ควรต่าง เช่น namespace, host, replica, image tag, Vault path
- backend ใช้ Vault Agent เพราะมี secret จริง
- frontend ใช้ public config เท่านั้น จึงไม่จำเป็นต้องใช้ Vault สำหรับ `NEXT_PUBLIC_*`

## ตัวอย่าง apply manual

### dev

```bash
kubectl apply -k k8s/overlays/dev/postgres
kubectl apply -k k8s/overlays/dev/backend
kubectl apply -k k8s/overlays/dev/frontend
```

### prod

```bash
kubectl apply -k k8s/overlays/prod/postgres
kubectl apply -k k8s/overlays/prod/backend
kubectl apply -k k8s/overlays/prod/frontend
```

## ตัวอย่าง apply ผ่าน Argo CD

### dev

```bash
kubectl apply -f argocd/dev/postgres-app.yaml
kubectl apply -f argocd/dev/backend-app.yaml
kubectl apply -f argocd/dev/frontend-app.yaml
```

### prod

```bash
kubectl apply -f argocd/prod/postgres-app.yaml
kubectl apply -f argocd/prod/backend-app.yaml
kubectl apply -f argocd/prod/frontend-app.yaml
```

## ค่าที่คุณต้องแก้เอง

- `YOUR_REGISTRY/search-api:*`
- `YOUR_REGISTRY/search-web:*`
- `YOUR_DOMAIN`
- `https://github.com/YOUR_ORG/YOUR_REPO.git`
- รหัสผ่าน postgres ใน overlay
- Vault role และ Vault path ให้ตรงกับ infra จริง

## Vault path ที่แนะนำ

- dev backend → `kv/data/searchable-from-book/dev/backend`
- prod backend → `kv/data/searchable-from-book/prod/backend`

โดยใน pod จะ inject เป็นไฟล์:

- `/vault/secrets/backend-env`

และ entrypoint ของ backend จะ source ไฟล์นี้ก่อนรัน binary

## หมายเหตุสำคัญ

frontend ไม่ได้ซ่อน secret ได้จริง เพราะค่าที่ browser ใช้จะมองเห็นได้อยู่แล้ว
ดังนั้นแยกหน้าที่ให้ชัด:

- **backend** → secret จริงใน Vault
- **frontend** → public config เช่น `/api`
