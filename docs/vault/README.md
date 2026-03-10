# HashiCorp Vault Agent setup example

ตัวอย่างนี้ใช้ Vault KV v2 ที่ mount ชื่อ `kv`

## 1) เก็บ secret ของ backend

```bash
vault kv put kv/searchable-from-book/backend \
  DATABASE_URL='postgres://search_user:change-me@postgres:5432/searchdb?sslmode=disable'
```

## 2) สร้าง policy

ไฟล์ตัวอย่างอยู่ที่:
- `docs/vault/backend-policy.hcl`

Apply:

```bash
vault policy write search-backend docs/vault/backend-policy.hcl
```

## 3) ผูก Vault Kubernetes auth role

ตัวอย่าง:

```bash
vault write auth/kubernetes/role/search-backend \
  bound_service_account_names=search-api-sa \
  bound_service_account_namespaces=searchable-from-book \
  policies=search-backend \
  ttl=24h
```

## 4) สิ่งที่ deployment backend ทำ

ใน `k8s/backend/deployment.yaml` มี annotation เหล่านี้:

- `vault.hashicorp.com/agent-inject: "true"`
- `vault.hashicorp.com/role: "search-backend"`
- `vault.hashicorp.com/agent-inject-secret-backend-env: "kv/data/searchable-from-book/backend"`

Vault Agent จะ render file แบบนี้ออกมา:

```bash
export DATABASE_URL="..."
```

แล้ว entrypoint ของ container จะ load เข้า environment ก่อนสั่งรัน API

## 5) ทำไม frontend ไม่ใช้ Vault ในตัวอย่างนี้

เพราะค่า `NEXT_PUBLIC_*` ไม่ใช่ secret จริง และสุดท้ายจะถูกส่งไป browser อยู่ดี
ดังนั้นค่าพวกนี้ไม่ควรถูกปฏิบัติเป็น secret

ค่าที่เป็น secret จริงควรอยู่ backend เท่านั้น เช่น
- database password
- private API key
- signing secret
