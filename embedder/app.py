from fastapi import FastAPI
from pydantic import BaseModel
from sentence_transformers import SentenceTransformer

app = FastAPI()

# รุ่นเริ่มต้นที่ลองก่อนได้
model = SentenceTransformer("paraphrase-multilingual-MiniLM-L12-v2")

class EmbedRequest(BaseModel):
    text: str

@app.get("/health")
def health():
    return {"ok": True}

@app.post("/embed")
def embed(req: EmbedRequest):
    vec = model.encode(req.text, normalize_embeddings=True).tolist()
    return {"vector": vec}