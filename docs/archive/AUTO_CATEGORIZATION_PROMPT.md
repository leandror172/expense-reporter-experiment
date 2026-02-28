# Expense Auto-Categorization Analysis and Artifact Generation

## Context and Objective

You are tasked with analyzing an Excel expense workbook and automatically categorizing uncategorized expenses from a CSV file. Your primary goals are:

1. **Analyze** the existing expense data in the Excel workbook to understand categorization patterns
2. **Auto-assign** categories to new expenses in the provided CSV (which lacks category information)
3. **Generate comprehensive artifacts** documenting every aspect of your classification logic, reasoning, and decision-making process

This is a research and development task. The artifacts you generate will be used by other AI systems and developers to:
- Understand the classification algorithm
- Reproduce the categorization logic
- Research auto-classification approaches
- Develop automated categorization systems
- Study pattern recognition in expense data

## Input Files You Will Receive

### 1. Excel Workbook Structure
The Excel file contains multiple sheets with historical expense data:
- **Sheet Names**: "Fixas", "Variáveis", "Extras", "Investimentos"
- **Structure**: Each sheet has:
  - Column A: Empty/formatting
  - Column B: Category names (merged cells for subcategories)
  - Column C: Subcategory names (merged cells spanning multiple rows)
  - Columns D onwards: Month columns (January, February, etc.)
  - Each month has 3 columns: Item, Date, Value
- **Reference Sheet**: Contains mapping of subcategories to sheets and categories

### 2. CSV File Format (Input - Uncategorized)
```csv
# Format: <item_description>;<DD/MM>;<value_##,##>
# Note: NO SUBCATEGORY - you must predict it

Uber Centro;15/04;35,50
Compras Carrefour;03/01;150,00
Pão francês;22/12;8,50
Consulta médica;05/02;200,00
```

**Your task**: Add the fourth field (subcategory) based on analysis of the Excel data.

## Analysis Requirements

### Phase 1: Data Extraction and Understanding

1. **Load and parse the Excel workbook**
2. **Extract all historical expenses** from all sheets
3. **Build a comprehensive dataset** with fields:
   - Item description
   - Date
   - Value
   - Subcategory
   - Category
   - Sheet name
4. **Create data summary statistics**:
   - Total number of expenses per category
   - Total number of expenses per subcategory
   - Value distributions
   - Temporal patterns
   - Text pattern frequencies

### Phase 2: Pattern Analysis and Feature Extraction

Analyze the historical data to identify patterns. For each expense, extract:

1. **Lexical Features**:
   - Keywords and n-grams (unigrams, bigrams, trigrams)
   - Word frequencies
   - Character patterns
   - Text length statistics

2. **Semantic Features**:
   - Semantic similarity between item descriptions
   - Conceptual clustering (e.g., "Uber", "táxi", "99" → Transportation)
   - Brand/merchant identification
   - Location indicators

3. **Numerical Features**:
   - Value ranges per category/subcategory
   - Value percentiles
   - Outlier detection

4. **Temporal Features**:
   - Time-of-year patterns
   - Frequency patterns (recurring vs. one-time)

### Phase 3: Classification Algorithm Development

Develop and document your classification approach:

1. **Algorithm Selection**:
   - Document which approach(es) you're using (rule-based, similarity-based, probabilistic, etc.)
   - Explain why you chose this approach

2. **Feature Engineering**:
   - Document how you transform raw data into classification features
   - Explain feature weighting and importance

3. **Decision Logic**:
   - Document the step-by-step decision tree/logic
   - Explain how conflicts are resolved
   - Document confidence scoring

### Phase 4: Classification Execution

For each uncategorized expense in the CSV:

1. **Extract features** from the item description and value
2. **Apply classification algorithm** step-by-step
3. **Generate confidence scores** for each potential category
4. **Select final category** with reasoning
5. **Document any ambiguities or edge cases**

## Required Artifacts (CRITICAL - Generate ALL of These)

### Artifact 1: Training Data Export
**File**: `training_data.json`
**Format**: JSON
**Content**: Complete historical expense dataset
```json
{
  "metadata": {
    "total_expenses": 1234,
    "date_range": "2025-01-01 to 2025-12-31",
    "sheets": ["Fixas", "Variáveis", "Extras", "Investimentos"],
    "extraction_timestamp": "ISO-8601"
  },
  "expenses": [
    {
      "id": 1,
      "item": "Uber Centro",
      "date": "2025-04-15",
      "value": 35.50,
      "subcategory": "Uber/Taxi",
      "category": "Transporte",
      "sheet": "Variáveis",
      "month": 4,
      "year": 2025
    }
  ]
}
```

### Artifact 2: Feature Dictionary
**File**: `feature_dictionary.json`
**Content**: All extracted features from historical data
```json
{
  "lexical_features": {
    "keywords": {
      "uber": {"frequency": 45, "subcategories": ["Uber/Taxi"], "tf_idf": 0.89},
      "compras": {"frequency": 234, "subcategories": ["Supermercado", "Feira"], "tf_idf": 0.45}
    },
    "ngrams": {
      "bigrams": {
        "compras carrefour": {"frequency": 12, "subcategory": "Supermercado"}
      },
      "trigrams": {}
    }
  },
  "value_ranges": {
    "Uber/Taxi": {"min": 15.00, "max": 120.00, "mean": 42.30, "median": 38.50, "std_dev": 18.22},
    "Supermercado": {"min": 50.00, "max": 800.00, "mean": 245.50, "median": 220.00, "std_dev": 95.40}
  },
  "semantic_clusters": {
    "transportation": {
      "keywords": ["uber", "taxi", "99", "transporte", "viagem"],
      "subcategories": ["Uber/Taxi", "Combustível"]
    }
  }
}
```

### Artifact 3: Classification Algorithm Specification
**File**: `classification_algorithm.md`
**Content**: Detailed algorithm documentation in markdown

**Required sections**:
1. Algorithm Overview (high-level approach)
2. Step-by-Step Process (pseudocode or flowchart in mermaid)
3. Feature Extraction Pipeline
4. Similarity/Matching Functions (with formulas)
5. Confidence Scoring Method
6. Tie-Breaking Rules
7. Edge Case Handling
8. Performance Characteristics (time/space complexity)

### Artifact 4: Decision Matrix
**File**: `decision_matrix.csv`
**Content**: Matrix showing classification scores for each input expense vs. each possible subcategory
```csv
Expense_ID,Item,Uber/Taxi,Supermercado,Padaria,Dentista,...,Selected_Category,Confidence
1,"Uber Centro",0.95,0.02,0.01,0.00,...,Uber/Taxi,0.95
2,"Compras Carrefour",0.01,0.98,0.05,0.00,...,Supermercado,0.98
```

### Artifact 5: Similarity Matrix
**File**: `similarity_matrix.json`
**Content**: Pairwise similarity scores between input expenses and historical expenses
```json
{
  "input_expense_1": {
    "item": "Uber Centro",
    "top_matches": [
      {"historical_id": 456, "similarity": 0.95, "method": "cosine", "item": "Uber centro SP"},
      {"historical_id": 123, "similarity": 0.89, "method": "cosine", "item": "Uber"},
      {"historical_id": 789, "similarity": 0.85, "method": "cosine", "item": "Uber para centro"}
    ],
    "similarity_metrics": {
      "cosine_similarity": 0.95,
      "jaccard_similarity": 0.87,
      "levenshtein_distance": 3,
      "semantic_similarity": 0.92
    }
  }
}
```

### Artifact 6: Vector Representations
**File**: `vector_representations.json`
**Content**: Numerical vector representations of text (if using embeddings/vectorization)
```json
{
  "method": "TF-IDF" or "word2vec" or "custom",
  "dimensions": 100,
  "vocabulary_size": 500,
  "vectors": {
    "input_expense_1": [0.23, 0.45, 0.12, ...],
    "historical_expense_456": [0.25, 0.43, 0.15, ...]
  },
  "transformation_matrix": {
    "description": "If dimensionality reduction applied (PCA, etc.)",
    "original_dimensions": 500,
    "reduced_dimensions": 100,
    "explained_variance": 0.95,
    "components": [[0.12, 0.34, ...], ...]
  }
}
```

### Artifact 7: Classification Reasoning Log
**File**: `classification_reasoning.md`
**Content**: Detailed reasoning for EACH classified expense

**Format per expense**:
```markdown
## Expense #1: "Uber Centro;15/04;35,50"

### Feature Extraction
- Keywords detected: ["uber", "centro"]
- Value: 35.50 BRL
- Date: April 15

### Pattern Matching
1. Keyword "uber" found in historical data:
   - 45 occurrences
   - 100% in subcategory "Uber/Taxi"
   - Average value: 42.30 BRL (within 1 std dev)

2. Value analysis:
   - 35.50 is within expected range for "Uber/Taxi" [15.00, 120.00]
   - 48th percentile of historical Uber/Taxi expenses

3. Semantic similarity:
   - Highest match: "Uber centro SP" (similarity: 0.95)
   - Cluster: Transportation
   - Confidence: HIGH

### Candidate Scores
| Subcategory | Keyword_Score | Value_Score | Semantic_Score | Final_Score |
|-------------|---------------|-------------|----------------|-------------|
| Uber/Taxi   | 1.00          | 0.85        | 0.95           | 0.95        |
| Combustível | 0.00          | 0.30        | 0.15           | 0.08        |
| ...         | ...           | ...         | ...            | ...         |

### Decision
**Selected**: Uber/Taxi
**Confidence**: 0.95 (HIGH)
**Reasoning**: Strong keyword match (100% historical correlation), value within expected range, high semantic similarity to historical Uber expenses.
```

### Artifact 8: Confusion Analysis
**File**: `confusion_analysis.json`
**Content**: Analysis of ambiguous or low-confidence classifications
```json
{
  "ambiguous_cases": [
    {
      "expense": "Consulta;05/02;200,00",
      "candidates": [
        {"subcategory": "Dentista", "score": 0.65, "reason": "Value matches typical dental consultation"},
        {"subcategory": "Médico", "score": 0.60, "reason": "Generic 'consulta' term"},
        {"subcategory": "Orion - Consultas", "score": 0.58, "reason": "Veterinary consultation range"}
      ],
      "selected": "Médico",
      "confidence": "MEDIUM",
      "recommendation": "Manual review suggested - ambiguous term 'consulta'"
    }
  ],
  "edge_cases": [
    {
      "expense": "Item raro nunca visto;01/01;1000,00",
      "issue": "No similar historical items found",
      "approach": "Fell back to value-based classification",
      "confidence": "LOW"
    }
  ]
}
```

### Artifact 9: Algorithm Parameters
**File**: `algorithm_parameters.json`
**Content**: All tunable parameters and hyperparameters used
```json
{
  "feature_weights": {
    "keyword_match": 0.45,
    "semantic_similarity": 0.30,
    "value_proximity": 0.15,
    "temporal_pattern": 0.10
  },
  "thresholds": {
    "high_confidence": 0.85,
    "medium_confidence": 0.60,
    "low_confidence": 0.40,
    "minimum_similarity": 0.30
  },
  "similarity_methods": {
    "text": "cosine_similarity",
    "value": "gaussian_kernel",
    "hybrid": "weighted_average"
  },
  "preprocessing": {
    "lowercase": true,
    "remove_accents": false,
    "remove_stopwords": true,
    "stemming": false
  }
}
```

### Artifact 10: Statistical Summary
**File**: `statistical_summary.json`
**Content**: Statistical analysis of the classification results
```json
{
  "input_summary": {
    "total_expenses": 50,
    "date_range": "2025-01-01 to 2025-12-31",
    "total_value": 5432.10
  },
  "classification_summary": {
    "high_confidence": 35,
    "medium_confidence": 12,
    "low_confidence": 3,
    "average_confidence": 0.82
  },
  "category_distribution": {
    "Variáveis": 28,
    "Fixas": 10,
    "Extras": 8,
    "Investimentos": 4
  },
  "subcategory_distribution": {
    "Uber/Taxi": 8,
    "Supermercado": 12,
    "Padaria": 5
  }
}
```

### Artifact 11: Categorized Output CSV
**File**: `categorized_expenses.csv`
**Content**: Original CSV with added subcategories
```csv
# Auto-categorized by expense classification algorithm
# Confidence levels: HIGH (>0.85), MEDIUM (0.60-0.85), LOW (<0.60)
# Format: <item>;<DD/MM>;<value>;<subcategory>;<confidence>

Uber Centro;15/04;35,50;Uber/Taxi;0.95
Compras Carrefour;03/01;150,00;Supermercado;0.98
Pão francês;22/12;8,50;Padaria;0.92
Consulta médica;05/02;200,00;Médico;0.65
```

### Artifact 12: LLM Internal Reasoning (META-ANALYSIS)
**File**: `llm_reasoning_meta.md`
**Content**: Self-analysis of YOUR OWN classification process

**Required sections**:
1. **Cognitive Approach**: How did you (the LLM) approach this problem?
2. **Pattern Recognition**: What patterns did you recognize that humans might not?
3. **Uncertainty Handling**: How did you handle ambiguous cases?
4. **Bias Detection**: What biases might be present in your classification?
5. **Confidence Calibration**: How did you determine confidence levels?
6. **Failure Modes**: What types of expenses would likely fail with this approach?
7. **Improvement Suggestions**: How could this approach be improved?

### Artifact 13: Reproducibility Package
**File**: `reproducibility_guide.md`
**Content**: Complete guide to reproduce your classification

**Must include**:
1. Exact steps to recreate the classification
2. Required data preprocessing steps
3. Feature extraction code (pseudocode acceptable)
4. Classification algorithm (pseudocode or actual code)
5. Example walkthrough of classifying one expense
6. Testing/validation approach

### Artifact 14: Research Insights
**File**: `research_insights.md`
**Content**: Insights for ML/AI researchers

**Topics to cover**:
1. What makes expense categorization challenging?
2. Most discriminative features identified
3. Patterns that humans might miss
4. Potential for transfer learning
5. Suggestions for supervised learning approaches
6. Feature engineering recommendations
7. Data augmentation possibilities

## Output Format Requirements

### Primary Deliverable
Return the categorized CSV with this exact format:
```csv
<item>;<DD/MM>;<value>;<subcategory>
```

### Artifact Organization
All artifacts should be generated as separate code blocks or file outputs with clear labels.

### Confidence Reporting
For each classified expense, report:
- **Confidence Level**: HIGH/MEDIUM/LOW
- **Confidence Score**: 0.00 to 1.00
- **Top 3 Alternative Categories**: With their scores

## Special Instructions

### 1. Mathematical Rigor
When describing any mathematical operations:
- Provide the actual formulas (LaTeX acceptable)
- Explain the intuition
- Show example calculations

**Example**:
```
Cosine Similarity between vectors A and B:

similarity(A, B) = (A · B) / (||A|| × ||B||)

Where:
- A · B = sum of element-wise products
- ||A|| = L2 norm of vector A = sqrt(sum(a_i²))

Example:
A = [1, 0, 1]  (representing "uber centro")
B = [1, 0, 1]  (representing "uber centro sp")
similarity = (1×1 + 0×0 + 1×1) / (√2 × √2) = 2/2 = 1.00
```

### 2. Transparency
Be explicit about:
- Assumptions you're making
- Limitations of your approach
- Cases where you're uncertain
- Arbitrary choices (and why you made them)

### 3. Low-Level Detail
Include granular details like:
- Exact tokenization approach
- Normalization steps (lowercase, accent removal, etc.)
- Tie-breaking logic
- Rounding/precision handling
- Edge case handling

### 4. Matrix Operations
If you use any matrix operations (TF-IDF, embeddings, etc.):
- Show the actual matrices (or representative samples)
- Explain transformations step-by-step
- Document dimensionality at each step

### 5. Algorithm Complexity
Document computational complexity:
- Time complexity: O(?)
- Space complexity: O(?)
- Bottlenecks identified

## Quality Criteria

Your artifacts will be evaluated on:

1. **Completeness**: All 14 required artifacts generated
2. **Reproducibility**: Another system could recreate your results
3. **Transparency**: Decision logic is clear and well-documented
4. **Depth**: Low-level details are included
5. **Rigor**: Mathematical operations are properly specified
6. **Practicality**: Artifacts are usable for development/research
7. **Insights**: Non-obvious patterns are identified and documented

## Example Interaction Flow

1. **User provides**: Excel workbook file
2. **You respond**: "I've received the workbook. I'll now analyze it and extract historical expense data..."
3. **You generate**: Artifact 1 (Training Data Export)
4. **You continue**: "Now analyzing patterns... I've identified 234 unique keywords across 1,234 historical expenses..."
5. **You generate**: Artifact 2 (Feature Dictionary)
6. **User provides**: Uncategorized CSV
7. **You respond**: "I'll now classify these expenses using the patterns I've learned..."
8. **You generate**: All remaining artifacts (3-14)
9. **You provide**: Final categorized CSV

## Critical Reminders

- **Generate ALL 14 artifacts** - this is non-negotiable
- **Show your work** - include intermediate calculations
- **Be specific** - avoid vague descriptions
- **Include examples** - concrete examples for abstract concepts
- **Document assumptions** - be explicit about what you're assuming
- **Think like a researcher** - these artifacts should enable future research

## Meta-Instruction for Claude Opus 4.5

This is a complex multi-faceted task that requires:
- Analytical thinking (pattern extraction)
- Creative reasoning (classification logic)
- Rigorous documentation (artifact generation)
- Self-reflection (meta-analysis of your own process)

Take your time with each phase. The artifacts are just as important as the classification results themselves. Think of yourself as both:
1. A classification system (doing the task)
2. A researcher documenting the system (explaining how it works)

Your goal is to create artifacts so comprehensive that:
- A human developer could implement your algorithm
- Another AI could reproduce your results
- Researchers could study your approach
- The classification logic could be validated and improved

Good luck! The quality of your artifacts will determine the value of this entire exercise.
