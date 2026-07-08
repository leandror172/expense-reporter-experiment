import json
import os
import sys
import math
import argparse
import urllib.request
from dataclasses import dataclass
from typing import List, Dict, Tuple, Optional, Set, Any
from collections import Counter, defaultdict
from pathlib import Path


@dataclass
class PoolEntry:
    item: str
    subcategory: str
    source: str


def _load_training_data(path: Path) -> List[PoolEntry]:
    """Load training data from JSON file."""
    with open(path, "r", encoding="utf-8") as f:
        data = json.load(f)
    return [
        PoolEntry(
            item=item["item"],
            subcategory=item["subcategory"],
            source="training"
        )
        for item in data["expenses"]
    ]


def _load_feedback_data(path: Path) -> List[PoolEntry]:
    """Load feedback data from JSONL file."""
    entries = []
    with open(path, "r", encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            data = json.loads(line)
            status = data["status"]
            if status == "manual":
                continue
            item = data["item"]
            if status == "confirmed":
                subcategory = data["predicted_subcategory"]
            elif status == "corrected":
                subcategory = data["actual_subcategory"]
            else:
                raise ValueError(f"Unknown status: {status}")
            entries.append(
                PoolEntry(
                    item=item,
                    subcategory=subcategory,
                    source="feedback"
                )
            )
    return entries


def _merge_pools(feedback_entries: List[PoolEntry], training_entries: List[PoolEntry]) -> List[PoolEntry]:
    """Merge pools like the Go MergeExamplePools function."""
    seen = set()
    result = []
    
    for entry in feedback_entries:
        key = entry.item.lower().strip()
        if key not in seen:
            seen.add(key)
            result.append(entry)
    
    # Faithful to Go MergeExamplePools (loader.go:186-191): training keys are
    # NOT added to seen — duplicate training items are all kept.
    for entry in training_entries:
        key = entry.item.lower().strip()
        if key not in seen:
            result.append(entry)
    
    return result


def _get_unique_texts(pool: List[PoolEntry], queries: List[Dict]) -> Set[str]:
    """Get unique texts to embed from pool and queries."""
    texts = set()
    for entry in pool:
        texts.add(entry.item)
    for query in queries:
        texts.add(query["item"])
    return texts


def _get_cache_path(outdir: Path, model_name: str) -> Path:
    """Generate cache file path for a model."""
    model_id = model_name.replace(":", "-").replace("/", "-")
    return outdir / f"emb-cache-{model_id}.jsonl"


def _embed_text(text: str, model_name: str, ollama_url: str) -> List[float]:
    """Embed a single text using Ollama API. Retries once on failure."""
    url = f"{ollama_url}/api/embeddings"
    payload = json.dumps({"model": model_name, "prompt": text}).encode("utf-8")
    request = urllib.request.Request(url, data=payload, headers={"Content-Type": "application/json"})

    last_err = None
    for attempt in range(2):
        try:
            with urllib.request.urlopen(request) as response:
                embedding_data = json.loads(response.read().decode("utf-8"))
                return embedding_data["embedding"]
        except (urllib.error.URLError, OSError) as e:
            last_err = e
    print(f"Embedding failed twice for '{text}' with model '{model_name}': {last_err}")
    sys.exit(1)


def _get_embeddings(
    texts: Set[str],
    model_name: str,
    ollama_url: str,
    cache_path: Path
) -> Dict[str, List[float]]:
    """Get embeddings for all texts using Ollama API."""
    os.makedirs(cache_path.parent, exist_ok=True)

    cache_entries = []
    if os.path.exists(cache_path):
        with open(cache_path, "r", encoding="utf-8") as f_cache:
            cache_entries = [json.loads(line) for line in f_cache if line.strip()]
    
    embeddings = {}
    for entry in cache_entries:
        text = entry["text"]
        if text in texts:
            embeddings[text] = entry["embedding"]
    
    missing_texts = sorted(texts - set(embeddings.keys()))
    if missing_texts:
        with open(cache_path, "a", encoding="utf-8") as f_cache:
            for i, text in enumerate(missing_texts):
                if i % 200 == 0 and i > 0:
                    print(f"Embedded {i} texts")
                try:
                    embedding = _embed_text(text, model_name, ollama_url)
                    embeddings[text] = embedding
                    f_cache.write(json.dumps({"text": text, "embedding": embedding}) + "\n")
                except Exception as e:
                    print(f"Error embedding '{text}' with model '{model_name}': {str(e)}")
                    sys.exit(1)
    
    return embeddings


def _process_queries(
    queries: List[Dict],
    pool: List[PoolEntry],
    embeddings: Dict[str, List[float]],
    topk: int
) -> List[Dict]:
    """Process each query to find nearest neighbors."""
    results = []
    for query in queries:
        item = query["item"]
        actual_subcategory = query["actual_subcategory"]
        
        # Get dedup key
        key = item.lower().strip()
        
        # Filter pool entries by different keys
        candidates = [
            entry for entry in pool
            if entry.item.lower().strip() != key
        ]
        
        # Get embeddings for candidates and query
        candidate_embeddings = {
            entry.item: embeddings[entry.item] for entry in candidates
        }
        query_embedding = embeddings[item]
        
        # Compute cosine similarity
        similarities = []
        for entry in candidates:
            text = entry.item
            if text not in candidate_embeddings:
                continue  # Shouldn't happen since we checked earlier
            sim = _cosine_similarity(query_embedding, candidate_embeddings[text])
            similarities.append((sim, entry))
        
        # Sort by similarity and take topk
        similarities.sort(key=lambda pair: pair[0], reverse=True)
        top_neighbors = similarities[:topk]
        
        # Check for hits
        hit1 = any(entry.subcategory == actual_subcategory for sim, entry in top_neighbors[:1])
        hit3 = any(entry.subcategory == actual_subcategory for sim, entry in top_neighbors[:3])
        hit5 = any(entry.subcategory == actual_subcategory for sim, entry in top_neighbors[:5])
        
        # Prepare results
        neighbors = []
        for sim, entry in top_neighbors:
            neighbors.append({
                "item": entry.item,
                "subcategory": entry.subcategory,
                "source": entry.source,
                "similarity": round(sim, 4)
            })
        
        results.append({
            "id": query["id"],
            "item": item,
            "actual_subcategory": actual_subcategory,
            "neighbors": neighbors,
            "hit1": hit1,
            "hit3": hit3,
            "hit5": hit5
        })
    
    return results


def _cosine_similarity(a: List[float], b: List[float]) -> float:
    """Compute cosine similarity between two vectors."""
    dot_product = sum(x * y for x, y in zip(a, b))
    norm_a = math.sqrt(sum(x ** 2 for x in a))
    norm_b = math.sqrt(sum(y ** 2 for y in b))
    return dot_product / (norm_a * norm_b) if norm_a * norm_b != 0 else 0.0


def _write_results(results: List[Dict], outdir: Path, model_name: str) -> None:
    """Write results to JSONL file."""
    sanitized_model = model_name.replace(":", "-").replace("/", "-")
    output_path = outdir / f"nn-{sanitized_model}.jsonl"
    
    with open(output_path, "w", encoding="utf-8") as f_out:
        for result in results:
            f_out.write(json.dumps(result) + "\n")


def _calculate_hit_rates(results: List[Dict]) -> Tuple[float, float, float]:
    """Calculate hit@1, hit@3, and hit@5 rates."""
    total = len(results)
    if total == 0:
        return (0.0, 0.0, 0.0)
    
    hit1 = sum(1 for result in results if result["hit1"])
    hit3 = sum(1 for result in results if result["hit3"])
    hit5 = sum(1 for result in results if result["hit5"])
    
    return (hit1 / total * 100, hit3 / total * 100, hit5 / total * 100)


def _get_source_counts(results: List[Dict]) -> Dict[str, int]:
    """Count how many queries had a correct neighbor from feedback vs training."""
    source_counts = defaultdict(int)
    
    for result in results:
        if not result["hit5"]:
            continue
        
        # Find the first correct neighbor
        for neighbor in result["neighbors"]:
            if neighbor["subcategory"] == result["actual_subcategory"]:
                source_counts[neighbor["source"]] += 1
                break
    
    return source_counts


def _get_subcategory_hit_counts(results: List[Dict]) -> Dict[str, Dict[str, int]]:
    """Per actual_subcategory: total queries and hit@5 count."""
    counts: Dict[str, Dict[str, int]] = {}

    for result in results:
        subcat = result["actual_subcategory"]
        entry = counts.setdefault(subcat, {"hits": 0, "total": 0})
        entry["total"] += 1
        if result["hit5"]:
            entry["hits"] += 1

    return counts


def _print_model_summary(model_name: str, hit_rates: Tuple[float, float, float]) -> None:
    """Print model summary with hit rates."""
    print(f"{model_name}: hit@5 {hit_rates[2]:5.1f}%  (hit@1 {hit_rates[0]:5.1f}%, hit@3 {hit_rates[1]:5.1f}%)")


def _write_summary(
    outdir: Path,
    models: List[str],
    results_by_model: Dict[str, Dict]
) -> None:
    """Write summary JSON file."""
    summary = {
        "models": [
            {
                "name": model_name,
                "hit1": result["hit1"],
                "hit3": result["hit3"],
                "hit5": result["hit5"],
                "n_queries": len(result["results"]),
                "pool_size": result["pool_size"],
                "subcategory_hits": result["subcategory_hits"]
            }
            for model_name, result in results_by_model.items()
        ]
    }
    
    summary_path = outdir / "nn-summary.json"
    with open(summary_path, "w", encoding="utf-8") as f_out:
        json.dump(summary, f_out, indent=2)


def main() -> None:
    """Main function to run the nearest neighbor retrieval analysis."""
    parser = argparse.ArgumentParser(description="Measure embedding-based retrieval performance.")
    parser.add_argument("--training", default="../../../data/classification/training_data_complete.json")
    parser.add_argument("--feedback", default="../../../expense-reporter/classifications.jsonl")
    parser.add_argument("--queries", default="./retrieval.jsonl")
    parser.add_argument("--outdir", default=".")
    parser.add_argument("--models", default="nomic-embed-text,bge-m3,qwen3-embedding:8b,embeddinggemma,snowflake-arctic-embed2")
    parser.add_argument("--ollama", default="http://localhost:11434")
    parser.add_argument("--topk", type=int, default=5)
    
    args = parser.parse_args()
    
    script_dir = Path(os.path.dirname(os.path.abspath(__file__)))

    def _resolve(p: str) -> Path:
        path = Path(p)
        return path if path.is_absolute() else (script_dir / path).resolve()

    training_path = _resolve(args.training)
    feedback_path = _resolve(args.feedback)
    queries_path = _resolve(args.queries)
    outdir = _resolve(args.outdir)
    
    # Load data
    try:
        training_entries = _load_training_data(training_path)
        feedback_entries = _load_feedback_data(feedback_path)
        merged_pool = _merge_pools(feedback_entries, training_entries)
        print(f"Merged pool size: {len(merged_pool)}")
        
        with open(queries_path, "r", encoding="utf-8") as f:
            queries = [json.loads(line) for line in f if line.strip()]
        
        query_count = len([q for q in queries if q["n_examples"] == 0])
        print(f"Query count: {query_count}")
    except Exception as e:
        print(f"Error loading data: {str(e)}")
        sys.exit(1)
    
    # Process each model
    models = args.models.split(",")
    results_by_model = {}

    for model_name in models:
        try:
            cache_path = _get_cache_path(outdir, model_name)
            
            texts = _get_unique_texts(merged_pool, queries)
            embeddings = _get_embeddings(texts, model_name, args.ollama, cache_path)
            
            results = _process_queries(
                [q for q in queries if q["n_examples"] == 0],
                merged_pool,
                embeddings,
                args.topk
            )
            
            hit_rates = _calculate_hit_rates(results)
            source_counts = _get_source_counts(results)
            subcategory_hits = _get_subcategory_hit_counts(results)
            
            results_by_model[model_name] = {
                "results": results,
                "hit1": hit_rates[0],
                "hit3": hit_rates[1],
                "hit5": hit_rates[2],
                "pool_size": len(merged_pool),
                "subcategory_hits": subcategory_hits
            }

            _write_results(results, outdir, model_name)
            _print_model_summary(model_name, hit_rates)
            print(f"  first-correct-neighbor source: {dict(source_counts)}")
        except Exception as e:
            print(f"Error processing model '{model_name}': {str(e)}")

    # Final ranked table
    print("\n=== Summary (sorted by hit@5) ===")
    ranked = sorted(results_by_model.items(), key=lambda kv: kv[1]["hit5"], reverse=True)
    for model_name, r in ranked:
        print(f"{model_name:30s} hit@1 {r['hit1']:5.1f}%  hit@3 {r['hit3']:5.1f}%  hit@5 {r['hit5']:5.1f}%")

    _write_summary(outdir, models, results_by_model)


if __name__ == "__main__":
    main()
