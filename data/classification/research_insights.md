# Research Insights: Expense Categorization

This document provides insights for machine learning and AI researchers studying automatic expense categorization, text classification, and financial data analysis.

**Relevance to Layer 5:** Sections 1–3 directly inform prompt engineering for the `classify` command. Section 5 (Supervised Learning) and Section 7 (Novel Research Directions) inform the long-term improvement roadmap.

## Table of Contents

1. [What Makes Expense Categorization Challenging?](#1-what-makes-expense-categorization-challenging)
2. [Most Discriminative Features Identified](#2-most-discriminative-features-identified)
3. [Patterns That Humans Might Miss](#3-patterns-that-humans-might-miss)
4. [Potential for Transfer Learning](#4-potential-for-transfer-learning)
5. [Supervised Learning Approaches](#5-supervised-learning-approaches)
6. [Data Augmentation Possibilities](#6-data-augmentation-possibilities)
7. [Novel Research Directions](#7-novel-research-directions)
8. [Benchmark Dataset Creation](#8-benchmark-dataset-creation)
- [Conclusion: Open Research Questions](#conclusion-open-research-questions)

---

## 1. What Makes Expense Categorization Challenging?

### Unique Characteristics of Expense Data

**1.1 High Context Dependency**

Unlike news articles or product reviews, expense descriptions are extremely terse and context-dependent:
- Input: `"Maeda"` (single word)
- Required inference: Japanese bakery chain → Food category → Padaria subcategory
- Challenge: No explicit category indicators in the text

**Contrast with traditional text classification**:
- Email spam: Rich linguistic features (headers, content, links)
- Sentiment analysis: Explicit sentiment words ("love", "hate", "excellent")
- Topic classification: Multi-sentence documents with topic signals

**Expense classification**:
- Often 1-3 words
- Vendor names are opaque (e.g., "San Michel")
- Heavy reliance on external knowledge (knowing what vendors exist)

**1.2 Multi-Modal Classification**

Successful classification requires combining:
- **Text features**: Vendor names, item descriptions
- **Numerical features**: Transaction amounts
- **Temporal features**: Date, day of week, time of day (if available)
- **External knowledge**: Vendor database, geographic location, cultural context

This is fundamentally a multi-modal problem disguised as text classification.

**1.3 Class Imbalance and Long Tail**

Real-world expense data exhibits:
- **Dominant classes**: Supermercado (groceries), Restaurantes (dining) appear frequently
- **Rare classes**: Jardinagem (gardening), Papelaria (stationery) appear rarely
- **Power law distribution**: A few categories capture 80% of transactions

```
Category Distribution (Expected):
Supermercado: 25%
Delivery/Restaurantes: 20%
Transporte: 12%
Saúde: 10%
...
Jardinagem: 0.5%
Ferramentas: 0.3%
```

Traditional classifiers struggle with rare classes, often defaulting to majority class.

**1.4 Ambiguity Without Explicit Disambiguation**

Examples:
- `"Consulta"` → Medical? Dental? Veterinary?
- `"Compras"` → Groceries? Clothing? Electronics?
- `"Pagamento"` → Which service? Which vendor?

Resolution requires:
- Value-based inference (R$ 200 more likely dental than veterinary)
- Historical pattern matching (user's typical expenses)
- Sequential context (previous/next transactions)

**1.5 Evolving Vendor Landscape**

New vendors appear constantly:
- Online marketplaces (Shopee, Mercado Livre)
- Delivery apps (iFood, Rappi)
- Subscription services (Netflix, Spotify)

Models trained on historical data become stale quickly.

**1.6 Cultural and Linguistic Specificity**

Brazilian Portuguese has unique characteristics:
- Slang for sensitive purchases (`"chá"` = cannabis, `"seda"` = rolling papers)
- Local vendor names vary by region
- Abbreviations (`"DME"` = municipal water/electricity)
- Mixed language (English brand names in Portuguese context)

Transfer learning from English datasets has limited effectiveness.

### Research Question
**How can we build classification systems that handle extreme brevity, multi-modality, class imbalance, and cultural specificity simultaneously?**

---

## 2. Most Discriminative Features Identified

Based on this analysis, the following features showed highest discriminative power:

### 2.1 Vendor Name (Specificity: 0.95-1.0)

**Finding**: Exact vendor names are the single most reliable feature.

**Examples**:
- `"Maeda"` → 100% Padaria (n=8, specificity=1.0)
- `"Drogasil"` → 100% Farmácia (n=6, specificity=1.0)
- `"San Michel"` → 100% Supermercado (n=3, specificity=1.0)

**Implications**:
- Building a **vendor lookup table** is critical
- Consider **entity linking** to canonical vendor names
- Fuzzy matching needed for typos (`"Maeda"` vs. `"Maedaa"`)
- Geographic databases could help (`"San Michel"` is regional)

**Research opportunity**: Can we automatically mine vendor names from transaction history and build a knowledge graph?

### 2.2 Service Type + Personal Name (Specificity: 0.98)

**Finding**: Combining service type with person name gives near-perfect accuracy.

**Example**: `"Diarista Letícia"` → 98% confidence for Diarista category

**Pattern**: `<service_type> <person_name>`
- Diarista Maria
- Dentista João
- Cabeleireiro Ana

**Implication**: Named entity recognition (NER) for person names can boost accuracy.

**Research opportunity**: Can we learn these patterns automatically without manual rule creation?

### 2.3 Value Ranges (AUC-ROC: ~0.75)

**Finding**: Value distribution is moderately discriminative but not definitive.

**Examples**:
- Uber/Taxi: R$ 15-120, median R$ 38.50
- Supermercado: R$ 50-800, median R$ 220
- Cigarro: R$ 10-30, median R$ 18

**Challenge**: Overlapping ranges reduce discriminative power.

```
Value Overlap Example:
R$ 100-150 range could be:
- Small grocery run
- Restaurant meal for two
- Taxi to airport
- Pharmacy purchase
```

**Implication**: Value alone insufficient; needs combination with text features.

**Research opportunity**: Can Gaussian Mixture Models (GMM) capture multi-modal value distributions within categories?

### 2.4 Temporal Patterns (Unexplored in Current System)

**Hypothesis**: Time-based features could improve accuracy:
- **Day of month**: Utility bills on specific dates (e.g., 2nd-5th of month)
- **Day of week**: Grocery shopping on weekends
- **Frequency**: Monthly (rent, insurance) vs. sporadic (medical)

**Example Pattern**:
```
Unimed vencimento + 2nd of month + R$ 700-800 = Health insurance (99% confidence)
```

**Research opportunity**: Can we learn temporal patterns automatically using time series methods?

### 2.5 Semantic Clusters (Precision: ~0.85)

**Finding**: Manually defined semantic clusters effectively group related categories.

**Example cluster (Transportation)**:
```
Keywords: ["uber", "taxi", "99", "combustível", "gasolina", "posto", "pedágio"]
Categories: [Uber/Taxi, Combustível, Pedágio]
```

**Effectiveness**: When any cluster keyword matches, accuracy for cluster categories increases ~30%.

**Research opportunity**: Can we learn semantic clusters automatically using:
- Word embeddings (Word2Vec, GloVe)
- Topic modeling (LDA)
- Graph-based community detection

---

## 3. Patterns That Humans Might Miss

### 3.1 Bimodal Value Distributions

**Discovery**: Some categories have two distinct value modes.

**Example: "Anita" Business Expenses**
```
Mode 1: R$ 150-200 (ingredients, small supplies)
Mode 2: R$ 300-450 (loan repayments, bulk orders)
```

**Implication**: Single Gaussian assumption fails. Use Gaussian Mixture Models.

**Human oversight**: Humans might average these without noticing the bimodality.

**ML advantage**: Clustering algorithms naturally detect multi-modal distributions.

### 3.2 Contextual Meaning Shift

**Discovery**: Same keyword has different meanings based on value.

**Example: "chá" (tea)**
```
Value < R$ 50: Beverage tea (Café category)
Value > R$ 200: Cannabis slang (Diamba category)
```

**Pattern**: `meaning = f(keyword, value)`

**Human tendency**: Humans might consistently interpret "chá" one way without value context.

**ML advantage**: Decision trees naturally learn threshold-based rules.

### 3.3 Installment Payment Patterns

**Discovery**: Notation `"X/Y"` indicates installment (X out of Y payments).

**Examples**:
- `"Shopee Xiaomi 14t pro;24/06;3890,00/6"` → 6 installments
- `"Ração Orion Petz;31/05;485/3"` → 3 installments

**Implications**:
- Need to parse and extract installment information
- Track multi-month purchases
- Calculate total cost vs. monthly cost

**Human oversight**: Might not systematically track installment status.

**ML opportunity**: Regular expression parsing + tracking state across months.

### 3.4 Vendor Name Variations

**Discovery**: Same vendor appears with multiple spellings.

**Examples**:
- `"San michel"`, `"San michel 22/12"`, `"San Michel"`
- `"Maeda"`, `"MAEDA"`

**Solution**: Canonicalization + fuzzy matching.

**Research opportunity**: 
- Use edit distance (Levenshtein)
- Learn vendor name embeddings
- Entity resolution techniques

### 3.5 Implicit Location Information

**Discovery**: Some vendors are location-specific.

**Example**: `"Pizza na roça"` → "roça" is a specific restaurant name in Poços de Caldas.

**Implication**: Geographic knowledge bases could improve accuracy.

**Research opportunity**: Build location-aware vendor databases, similar to Yelp/Foursquare integration.

---

## 4. Potential for Transfer Learning

### 4.1 Cross-User Transfer

**Question**: Can a model trained on User A's expenses classify User B's expenses?

**Hypothesis**: Partially yes, with caveats.

**Transferable patterns**:
- Major vendors (McDonalds, Uber, Amazon)
- Common services (Supermercado, Delivery, Combustível)
- Value ranges (groceries typically R$ 100-300)

**Non-transferable patterns**:
- Personal services (User A's "Diarista Letícia" ≠ User B's "Diarista Maria")
- Regional vendors (São Paulo's "Pão de Açúcar" ≠ Poços de Caldas's "Almeida")
- Personal business expenses ("Anita" specific to this user)

**Proposed approach**:
1. Pre-train on multi-user dataset for general patterns
2. Fine-tune on user-specific data for personalization
3. Use few-shot learning for rare categories

**Research opportunity**: Meta-learning for expense categorization across users.

### 4.2 Cross-Language Transfer

**Question**: Can models trained on English expenses transfer to Portuguese?

**Challenge**: Cultural and linguistic differences.

**Example differences**:
- Brazilian slang (`"chá"`, `"diamba"`) has no English equivalent
- Vendor names are country-specific
- Value ranges differ (currency, cost of living)

**Possible transfer**:
- High-level semantic categories (Food, Transportation, Healthcare)
- Value distribution patterns (relative, not absolute)

**Proposed approach**:
- Multilingual embeddings (mBERT, XLM-R)
- Cross-lingual category mapping
- Value normalization by purchasing power parity (PPP)

**Research opportunity**: Cross-lingual expense classification with cultural adaptation.

### 4.3 Cross-Domain Transfer

**Question**: Can techniques from other text classification domains help?

**Potentially useful techniques**:

**From Named Entity Recognition (NER)**:
- Vendor name extraction
- Service type identification
- Person name detection (for personal services)

**From Information Retrieval**:
- TF-IDF for keyword importance
- BM25 for document similarity
- Query expansion for fuzzy matching

**From E-Commerce Product Categorization**:
- Hierarchical classification (Category → Subcategory)
- Multi-label classification (one expense, multiple categories)
- Product title parsing

**From Medical Coding (ICD-10)**:
- Handling ambiguous short texts
- Hierarchical code assignment
- Confidence-based recommendations

**Research opportunity**: Systematically evaluate transfer learning from these domains.

---

## 5. Supervised Learning Approaches

### 5.1 Recommended Model Architectures

**5.1.1 Baseline: Logistic Regression with TF-IDF**
```
Input: TF-IDF vectors (text) + normalized value
Output: Subcategory (multi-class)
Advantages: Interpretable, fast, low data requirements
Disadvantages: No sequence modeling, no context
```

**5.1.2 Intermediate: Random Forest / Gradient Boosting**
```
Features:
- TF-IDF vectors
- Bag-of-words
- Value
- Day of month
- Month of year
- Value percentile within history

Advantages: Handles non-linear relationships, feature importance
Disadvantages: Less effective for text than deep learning
```

**5.1.3 Advanced: Hierarchical Attention Network**
```
Architecture:
1. Word-level attention (which words matter?)
2. Vendor-level embedding (known vendors)
3. Value embedding (discretized into bins)
4. Temporal embedding (month, day)
5. Hierarchical classification (Sheet → Category → Subcategory)

Advantages: Captures hierarchy, multi-modal, attention weights interpretable
```

**5.1.4 State-of-the-Art: Fine-tuned BERT**
```
Model: BERTimbau (Portuguese BERT)
Fine-tuning: Classification head on [CLS] token
Additional inputs: Value, date as special tokens
Output: Subcategory probabilities

Advantages: Transfer learning, context understanding
Disadvantages: Overkill for short texts, resource-intensive
```

### 5.2 Feature Engineering Recommendations

**Text Features**:
1. **Character n-grams** (handles typos better than word n-grams)
2. **Word embeddings** (Word2Vec trained on transaction history)
3. **Vendor embedding** (learned representation of each vendor)
4. **Edit distance to known vendors** (fuzzy matching)

**Numerical Features**:
1. **Raw value**
2. **Log(value)** (reduces outlier impact)
3. **Value percentile** (relative to user's history)
4. **Z-score** (standardized value)
5. **Value bin** (discretized: <50, 50-100, 100-200, 200-500, >500)

**Temporal Features**:
1. **Day of month** (1-31)
2. **Day of week** (0-6)
3. **Month** (1-12)
4. **Days since last transaction** (frequency indicator)
5. **Seasonality** (holiday season, summer, etc.)

**Categorical Features**:
1. **Known vendor** (binary: yes/no)
2. **Has number in text** (binary)
3. **Contains person name** (binary, NER-based)
4. **Text length** (number of words)

**Interaction Features**:
1. **Vendor × Value** (specific vendor-value combinations)
2. **Day of month × Category** (bill payment patterns)
3. **Month × Category** (seasonal expenses)

### 5.3 Loss Functions

**Standard Cross-Entropy**:
```
L = -Σ y_true * log(y_pred)
```
Good for balanced datasets.

**Weighted Cross-Entropy** (for class imbalance):
```
L = -Σ w_i * y_true * log(y_pred)
where w_i = 1 / frequency(class_i)
```
Penalizes misclassification of rare classes more heavily.

**Focal Loss** (for hard examples):
```
L = -Σ (1 - y_pred)^γ * y_true * log(y_pred)
where γ = 2 (focusing parameter)
```
Focuses learning on hard-to-classify examples.

**Hierarchical Loss** (for category tree):
```
L = L_sheet + α * L_category + β * L_subcategory
```
Penalizes higher-level errors less than lower-level errors.

### 5.4 Evaluation Metrics

**Accuracy**: Overall correctness (biased toward frequent classes)

**Macro F1**: Average F1 across all classes (treats all classes equally)

**Weighted F1**: F1 weighted by class frequency (reflects real-world performance)

**Hierarchical Accuracy**:
```
Score = 1.0 if correct subcategory
        0.5 if correct category but wrong subcategory
        0.0 if wrong category
```

**Confidence-Calibrated Accuracy**:
```
For predictions with confidence > 0.85: measure accuracy
Expect > 95% accuracy
```

**Manual Review Rate**:
```
Percentage of expenses requiring human review (confidence < threshold)
Goal: < 20%
```

---

## 6. Data Augmentation Possibilities

### 6.1 Synthetic Data Generation

**6.1.1 Paraphrasing**
```
Original: "Uber Centro"
Augmented:
- "Uber centro SP"
- "Uber para centro"
- "Uber - centro"
- "centro uber"
```

**Method**: Use backtranslation (PT → EN → PT) or GPT-based paraphrasing.

**6.1.2 Vendor Name Variations**
```
Original: "Supermercado San Michel"
Augmented:
- "San Michel supermercado"
- "San michel"
- "Compras San Michel"
- "San Michel mercado"
```

**6.1.3 Value Perturbation**
```
Original: R$ 35.50
Augmented:
- R$ 33.20 (-7%)
- R$ 38.90 (+10%)
- R$ 35.00 (rounded)
```

**Method**: Sample from learned value distribution (Gaussian) for each category.

### 6.2 Cross-User Data Sharing

**Concept**: Pool data from multiple users (with privacy considerations).

**Approach**:
1. Users opt-in to share anonymized data
2. Vendor names canonicalized (remove personal info)
3. Train global model on pooled data
4. Fine-tune per user

**Privacy**: Differential privacy to prevent data leakage.

### 6.3 Active Learning

**Concept**: Strategically select which expenses to label.

**Strategy**:
1. Start with small labeled dataset (e.g., 50 expenses)
2. Train initial model
3. Classify unlabeled expenses
4. Request labels for:
   - Lowest confidence predictions
   - Expenses from underrepresented categories
   - Expenses with high prediction disagreement (ensemble)
5. Retrain with new labels
6. Repeat until performance plateaus

**Expected result**: Achieve 90% accuracy with 30-40% less labeled data.

---

## 7. Novel Research Directions

### 7.1 Few-Shot Learning for Rare Categories

**Problem**: Categories with < 5 examples in training data.

**Proposed approach**:
- Prototypical Networks
- Siamese Networks (learn similarity metric)
- Matching Networks

**Evaluation**: Can we correctly classify categories with only 1-3 examples?

### 7.2 Continual Learning (Non-Stationary Distribution)

**Problem**: New vendors appear, old vendors close, spending patterns change.

**Challenge**: Catastrophic forgetting (model forgets old patterns when learning new ones).

**Proposed solutions**:
- Elastic Weight Consolidation (EWC)
- Progressive Neural Networks
- Memory-based approaches (store old examples)

### 7.3 Explainable AI for Financial Categorization

**User need**: "Why did you categorize this as X?"

**Proposed explanations**:
- Attention visualization (which words mattered?)
- Counterfactual explanations ("If value was R$ 100, would be Y")
- Rule extraction (IF uber THEN Uber/Taxi)

**Evaluation**: User study on explanation quality.

### 7.4 Multi-Task Learning

**Tasks**:
1. Categorization (primary task)
2. Vendor extraction (auxiliary task)
3. Value prediction (predict typical value for category)
4. Anomaly detection (fraudulent transactions)

**Hypothesis**: Learning multiple related tasks improves performance on all tasks.

### 7.5 Graph-Based Classification

**Idea**: Model expenses as nodes in a graph.

**Edges**:
- Temporal (previous/next expense)
- Similarity (similar text/value)
- Category (same subcategory)

**Method**: Graph Neural Networks (GNN)
- Message passing between related expenses
- Learn category from neighborhood

**Advantage**: Captures sequential patterns, handles context.

### 7.6 Reinforcement Learning for Category Suggestion

**Formulation**:
- State: Current expense + transaction history
- Action: Suggest category
- Reward: +1 if user accepts, -1 if user corrects
- Policy: Learn optimal suggestion strategy

**Advantage**: Directly optimizes user satisfaction, not just accuracy.

---

## 8. Benchmark Dataset Creation

**Proposal**: Create public benchmark for expense categorization research.

**Dataset characteristics**:
- Multi-user (10+ users, anonymized)
- Multi-language (Portuguese, English, Spanish)
- Diverse spending patterns (students, families, retirees)
- Temporal span (12+ months)
- Hierarchical labels (Sheet → Category → Subcategory)

**Annotation**:
- Expert annotations (financial advisors)
- Inter-annotator agreement (Kappa > 0.80)
- Ambiguous cases flagged

**Evaluation protocol**:
- Train/validation/test split (60/20/20)
- Per-user evaluation (user-specific test set)
- Cross-user evaluation (train on users A-I, test on user J)
- Cold-start evaluation (classify with < 10 labeled examples)

**Baseline models**:
- TF-IDF + Logistic Regression
- Random Forest with engineered features
- Fine-tuned BERT

**Leaderboard metrics**:
- Macro F1 (primary)
- Weighted F1
- Hierarchical accuracy
- Manual review rate (confidence < 0.85)

---

## Conclusion: Open Research Questions

1. **How can we build expense categorization systems that generalize across users while preserving personalization?**

2. **What is the optimal balance between rule-based patterns (interpretable, precise) and machine learning (flexible, scalable)?**

3. **Can we achieve > 95% accuracy with < 100 labeled examples per user (few-shot learning)?**

4. **How can we handle concept drift as spending patterns and vendor landscapes evolve over time?**

5. **What role can large language models (LLMs) play in expense categorization beyond traditional classifiers?**

6. **How can we design human-AI collaboration workflows where AI handles high-confidence cases and intelligently routes uncertain cases to humans?**

---

**Document Purpose**: Inform future research in automatic expense categorization, text classification with multi-modal data, and personalized AI systems.

**Target Audience**: ML/AI researchers, NLP practitioners, FinTech developers, data scientists.

**Last Updated**: 2025-12-26
