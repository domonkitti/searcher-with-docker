package internal

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func getenvStr(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func getenvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func getenvFloat(key string, def float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return f
}

type runtimeState struct {
	mu      sync.RWMutex
	docs    []Doc
	kits    []KitDetail
	ruleCfg RuleConfig
	engine  *Engine
	syn     *Synonyms
	cfg     EngineConfig
	db      *sql.DB

	titleBoost int
	embedder   Embedder
}

func (s *runtimeState) snapshot() ([]Doc, []KitDetail, RuleConfig, *Engine) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.docs, s.kits, s.ruleCfg, s.engine
}

func (s *runtimeState) reloadEngineFromDB(ctx context.Context) error {
	docs, err := LoadDocsFromDB(ctx, s.db, s.titleBoost)
	if err != nil {
		return err
	}
	engine := NewEngine(docs, s.cfg, s.syn, s.embedder)

	s.mu.Lock()
	s.docs = docs
	s.engine = engine
	s.mu.Unlock()
	return nil
}

func (s *runtimeState) reloadKitsFromDB(ctx context.Context) error {
	kits, err := LoadKitDetailsFromDB(ctx, s.db)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.kits = kits
	s.mu.Unlock()
	return nil
}

// RunServer loads data from PostgreSQL on startup and serves API.
func RunServer() {
	titleBoost := getenvInt("TITLE_BOOST", 3)
	minScore := getenvFloat("MIN_SCORE", 0.0)
	nMin := getenvInt("NGRAM_MIN", 3)
	nMax := getenvInt("NGRAM_MAX", 6)
	addr := getenvStr("ADDR", ":8080")
	rulesPath := getenvStr("RULES_JSON", "data/rules.json")

	db, err := OpenDB()
	if err != nil {
		panic(err)
	}
	if err := EnsureSchema(db); err != nil {
		panic(err)
	}

	cfg := DefaultEngineConfig()
	cfg.MinScore = minScore
	cfg.NMin = nMin
	cfg.NMax = nMax

	synPath := getenvStr("SYNONYMS_JSON", "data/synonyms.json")
	var syn *Synonyms
	if s, err := LoadSynonyms(synPath); err == nil {
		syn = s
	}

	embedder := NewHTTPEmbedderFromEnv()

	docs := MustBuildDocsOrEmpty(context.Background(), db, titleBoost)
	kits, err := LoadKitDetailsFromDB(context.Background(), db)
	if err != nil {
		fmt.Println("load kits from db error:", err)
		kits = []KitDetail{}
	}
	ruleCfg := LoadRuleConfig(rulesPath)

	state := &runtimeState{
		docs:       docs,
		kits:       kits,
		ruleCfg:    ruleCfg,
		engine:     NewEngine(docs, cfg, syn, embedder),
		syn:        syn,
		cfg:        cfg,
		db:         db,
		titleBoost: titleBoost,
		embedder:   embedder,
	}

	r := gin.Default()

	allowOrigins := strings.Split(getenvStr("ALLOW_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000"), ",")
	for i := range allowOrigins {
		allowOrigins[i] = strings.TrimSpace(allowOrigins[i])
	}
	r.Use(cors.New(cors.Config{
		AllowOrigins: allowOrigins,
		AllowMethods: []string{"GET", "POST", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept"},
	}))

	r.GET("/api/live", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})
	r.GET("/api/ready", func(c *gin.Context) {
		if err := db.PingContext(c.Request.Context()); err != nil {
			c.JSON(503, gin.H{"ok": false, "error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"ok": true})
	})
	r.GET("/api/health", func(c *gin.Context) {
		docs, kits, _, engine := state.snapshot()
		c.JSON(200, gin.H{
			"ok":       true,
			"docs":     len(docs),
			"kits":     len(kits),
			"minScore": engine.Cfg.MinScore,
		})
	})

	r.GET("/api/search", func(c *gin.Context) {
		_, _, _, engine := state.snapshot()
		q := c.Query("q")
		k := 20
		if kk := c.Query("k"); kk != "" {
			if n, err := strconv.Atoi(kk); err == nil && n >= 1 && n <= 50 {
				k = n
			}
		}
		res := engine.Search(q, k)
		c.JSON(200, gin.H{"query": q, "results": res})
	})

	r.GET("/api/doc/:id", func(c *gin.Context) {
		_, _, _, engine := state.snapshot()
		id := c.Param("id")
		if d, ok := engine.GetByID(id); ok {
			c.JSON(200, d)
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"detail": "Not Found"})
	})

	r.GET("/api/subcategory", func(c *gin.Context) {
		docs, _, _, _ := state.snapshot()
		main := strings.TrimSpace(c.Query("main"))
		sub := strings.TrimSpace(c.Query("sub"))

		items := make([]gin.H, 0, 200)
		for _, d := range docs {
			m := d.Meta
			cm, _ := m["categoryMain"].(string)
			cs, _ := m["categorySub"].(string)
			if strings.TrimSpace(cm) == main && strings.TrimSpace(cs) == sub {
				items = append(items, gin.H{
					"id":    d.ID,
					"title": d.Title,
					"page":  m["page"],
					"row":   m["row"],
				})
			}
		}

		sort.Slice(items, func(i, j int) bool {
			ri, _ := strconv.Atoi(strings.TrimSpace(toStr(items[i]["row"])))
			rj, _ := strconv.Atoi(strings.TrimSpace(toStr(items[j]["row"])))
			if ri == 0 || rj == 0 {
				return toStr(items[i]["title"]) < toStr(items[j]["title"])
			}
			return ri < rj
		})

		c.JSON(200, gin.H{"main": main, "sub": sub, "count": len(items), "items": items})
	})

	r.GET("/api/group", func(c *gin.Context) {
		docs, _, _, _ := state.snapshot()
		main := strings.TrimSpace(c.Query("main"))
		sub := strings.TrimSpace(c.Query("sub"))
		group := strings.TrimSpace(c.Query("group"))

		items := make([]gin.H, 0, 300)
		for _, d := range docs {
			m := d.Meta
			cm, _ := m["categoryMain"].(string)
			cs, _ := m["categorySub"].(string)
			gp, _ := m["group"].(string)
			if strings.TrimSpace(cm) == main && strings.TrimSpace(cs) == sub && strings.TrimSpace(gp) == group {
				items = append(items, gin.H{
					"id":    d.ID,
					"title": d.Title,
					"page":  m["page"],
					"row":   m["row"],
				})
			}
		}

		sort.Slice(items, func(i, j int) bool {
			ri, _ := strconv.Atoi(strings.TrimSpace(toStr(items[i]["row"])))
			rj, _ := strconv.Atoi(strings.TrimSpace(toStr(items[j]["row"])))
			if ri == 0 || rj == 0 {
				return toStr(items[i]["title"]) < toStr(items[j]["title"])
			}
			return ri < rj
		})

		c.JSON(200, gin.H{"main": main, "sub": sub, "group": group, "count": len(items), "items": items})
	})

	r.GET("/api/kits", func(c *gin.Context) {
		_, kits, _, _ := state.snapshot()
		c.JSON(200, gin.H{"kits": kits})
	})

	r.GET("/api/kits/:kitId", func(c *gin.Context) {
		_, kits, _, _ := state.snapshot()
		kitId := c.Param("kitId")
		var found *KitDetail
		for i := range kits {
			if kits[i].KitID == kitId {
				found = &kits[i]
				break
			}
		}
		if found == nil {
			c.JSON(404, gin.H{"detail": "Not Found"})
			return
		}
		c.JSON(200, gin.H{"kit": found})
	})

	r.GET("/api/rules/config", func(c *gin.Context) {
		_, _, ruleCfg, _ := state.snapshot()
		c.JSON(200, ruleCfg)
	})

	r.POST("/api/rules/eval", func(c *gin.Context) {
		_, _, ruleCfg, _ := state.snapshot()
		var payload map[string]any
		if err := c.BindJSON(&payload); err != nil {
			c.JSON(400, gin.H{"error": "bad json"})
			return
		}

		inputs := map[string]float64{}
		for _, inp := range ruleCfg.Inputs {
			v, ok := payload[inp.Key]
			if !ok {
				continue
			}
			switch t := v.(type) {
			case float64:
				inputs[inp.Key] = t
			case string:
				if f, err := strconv.ParseFloat(t, 64); err == nil {
					inputs[inp.Key] = f
				}
			}
		}

		budget, allTrue, conditions := EvalRules(ruleCfg, inputs)
		c.JSON(200, gin.H{
			"budgetType": budget,
			"allTrue":    allTrue,
			"conditions": conditions,
			"logicNote":  ruleCfg.LogicNote,
		})
	})

	r.POST("/api/admin/import/items", func(c *gin.Context) {
		fh, err := c.FormFile("file")
		if err != nil {
			c.JSON(400, gin.H{"error": "file is required"})
			return
		}

		tempFilePath, err := saveUploadedFileTemp(fh)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		defer os.Remove(tempFilePath)

		inserted, err := SaveUploadedExcelAndImport(c.Request.Context(), db, tempFilePath)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		if err := state.reloadEngineFromDB(c.Request.Context()); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"ok": true, "inserted": inserted})
	})

	r.POST("/api/admin/import/item-links", func(c *gin.Context) {
		fh, err := c.FormFile("file")
		if err != nil {
			c.JSON(400, gin.H{"error": "file is required"})
			return
		}

		tempFilePath, err := saveUploadedFileTemp(fh)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		defer os.Remove(tempFilePath)

		inserted, err := SaveUploadedItemLinksExcelAndImport(c.Request.Context(), db, tempFilePath)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		if err := state.reloadEngineFromDB(c.Request.Context()); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"ok": true, "inserted": inserted})
	})

	r.POST("/api/admin/import/kits", func(c *gin.Context) {
		fh, err := c.FormFile("file")
		if err != nil {
			c.JSON(400, gin.H{"error": "file is required"})
			return
		}

		tempFilePath, err := saveUploadedFileTemp(fh)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		defer os.Remove(tempFilePath)

		inserted, err := SaveUploadedKitsExcelAndImport(c.Request.Context(), db, tempFilePath)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		if err := state.reloadKitsFromDB(c.Request.Context()); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"ok": true, "inserted": inserted})
	})

	nextProxyURL := strings.TrimSpace(getenvStr("NEXT_PROXY_URL", ""))
	if nextProxyURL != "" {
		attachNextProxy(r, nextProxyURL)
	} else {
		r.NoRoute(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.Path, "/api") {
				c.JSON(404, gin.H{"detail": "Not Found"})
				return
			}
			c.JSON(404, gin.H{"detail": "Not Found"})
		})
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		slog.Info("server listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	_ = db.Close()
}

func saveUploadedFileTemp(fh *multipart.FileHeader) (string, error) {
	ext := filepath.Ext(fh.Filename)
	tempFile, err := os.CreateTemp("", "items-import-*."+strings.TrimPrefix(ext, "."))
	if err != nil {
		return "", err
	}
	tempFile.Close()

	src, err := fh.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	dst, err := os.OpenFile(tempFile.Name(), os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := dst.ReadFrom(src); err != nil {
		return "", err
	}
	return tempFile.Name(), nil
}

func toStr(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}
