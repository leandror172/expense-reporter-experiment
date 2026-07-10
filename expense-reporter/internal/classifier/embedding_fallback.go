package classifier

import (
	"expense-reporter/internal/logger"
	"net/http"
	"sync"
)

const defaultEmbedModel = "snowflake-arctic-embed2"

type embedState struct {
	sync.Once
	store map[string][]float64
	err   error
}

var packageEmbedState embedState

func fallbackExamples(item string, pool []Example, emb Embedder, cachePath string, topK int, st *embedState) []Example {
	st.Do(func() {
		if len(pool) > 0 {
			logger.Info("embedding pool: reconciling cache", "poolSize", len(pool))
		}
		texts := make([]string, len(pool))
		for i, e := range pool {
			texts[i] = e.Item
		}
		store, err := ReconcileEmbeddings(cachePath, texts, emb)
		st.store = store
		st.err = err
	})

	if st.err != nil {
		logger.Debug("embedding fallback: reconcile failed", "err", st.err)
		return nil
	}

	queryVec, err := emb.Embed(item)
	if err != nil {
		logger.Debug("embedding fallback: query embed failed", "err", err)
		return nil
	}

	return TopKEmbeddingExamples(item, queryVec, pool, st.store, topK)
}

func embeddingFallback(item string, pool []Example, cfg Config, st *embedState) []Example {
	if cfg.NoEmbedRetrieval || cfg.DataDir == "" {
		return nil
	}

	model := cfg.EmbedModel
	if model == "" {
		model = defaultEmbedModel
	}

	emb := &OllamaEmbedder{
		URL:    cfg.OllamaURL,
		Model:  model,
		Client: http.DefaultClient,
	}

	cachePath := EmbeddingCachePath(cfg.DataDir, model)

	return fallbackExamples(item, pool, emb, cachePath, 5, st)
}
