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

func fallbackExamples(item string, pool []Example, embedder Embedder, cachePath string, topK int, embedState *embedState) []Example {
	embedState.Do(func() {
		if len(pool) > 0 {
			logger.Info("embedding pool: reconciling cache", "poolSize", len(pool))
		}
		texts := make([]string, len(pool))
		for i, e := range pool {
			texts[i] = e.Item
		}
		store, err := ReconcileEmbeddings(cachePath, texts, embedder)
		embedState.store = store
		embedState.err = err
	})

	if embedState.err != nil {
		logger.Debug("embedding fallback: reconcile failed", "err", embedState.err)
		return nil
	}

	queryVec, err := embedder.Embed(item)
	if err != nil {
		logger.Debug("embedding fallback: query embed failed", "err", err)
		return nil
	}

	return TopKEmbeddingExamples(item, queryVec, pool, embedState.store, topK)
}

func embeddingFallback(item string, pool []Example, cfg Config, embedState *embedState) []Example {
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

	return fallbackExamples(item, pool, emb, cachePath, 5, embedState)
}
