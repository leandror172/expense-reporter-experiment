#!/usr/bin/env python3
"""T-22 benchmark prep: derive two description-annotated taxonomy variants from the
real config/taxonomy.json (English and Portuguese type-level descriptions), without
touching the original file.

Usage: python3 make_desc_taxonomies.py
Writes: taxonomy-descen.json, taxonomy-descpt.json (this script's directory).
"""
import json
import sys
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
SOURCE_TAXONOMY = SCRIPT_DIR.parents[2] / "expense-reporter" / "config" / "taxonomy.json"

DESCRIPTIONS_EN = {
    "Fixas": (
        "Recurring fixed monthly commitments with a predictable, roughly constant "
        "amount: rent, financing installments, insurance, subscriptions, "
        "phone/internet, recurring taxes."
    ),
    "Variáveis": (
        "Recurring essential spending whose amount changes month to month: "
        "groceries, fuel and daily transport, pharmacy and routine health, home "
        "utilities (power/water/gas), personal care."
    ),
    "Extras": (
        "Irregular, one-off or unexpected necessary expenses, usually larger: "
        "medical and hospital, car or home maintenance/repairs, moving, school "
        "supplies, legal."
    ),
    "Adicionais": (
        "Discretionary optional spending (wants, not needs): leisure and "
        "entertainment, travel, dining out and delivery, clothing, gifts, hobbies."
    ),
}

DESCRIPTIONS_PT = {
    "Fixas": (
        "Compromissos mensais fixos e recorrentes, de valor previsível e "
        "aproximadamente constante: aluguel, prestações de financiamento, seguros, "
        "assinaturas, telefone/internet, impostos recorrentes."
    ),
    "Variáveis": (
        "Gastos essenciais recorrentes cujo valor muda de mês para mês: mercado, "
        "combustível e transporte do dia a dia, farmácia e saúde de rotina, contas "
        "de casa (luz/água/gás), cuidados pessoais."
    ),
    "Extras": (
        "Despesas necessárias irregulares, pontuais ou inesperadas, geralmente "
        "maiores: médico e hospital, manutenção/reparos de carro ou casa, mudança, "
        "material escolar, jurídico."
    ),
    "Adicionais": (
        "Gastos opcionais e discricionários (desejos, não necessidades): lazer e "
        "entretenimento, viagens, restaurantes e delivery, vestuário, presentes, "
        "hobbies."
    ),
}


def load_source_taxonomy(path: Path) -> dict:
    if not path.exists():
        sys.exit(f"ERROR: source taxonomy not found: {path}")
    with path.open("r", encoding="utf-8") as f:
        return json.load(f)


def inject_descriptions(taxonomy: dict, descriptions: dict) -> dict:
    """Return a new taxonomy dict identical to `taxonomy` except each element of
    types[] whose name matches a key in `descriptions` gets a "description" key
    inserted right after "name" (preserving remaining key order). Fails loudly if
    any expected name in `descriptions` is never matched against types[]."""
    types = taxonomy.get("types")
    if not isinstance(types, list):
        sys.exit('ERROR: source taxonomy has no top-level "types" array')

    matched = set()
    new_types = []
    for t in types:
        name = t.get("name")
        if name in descriptions:
            matched.add(name)
            new_t = {}
            for k, v in t.items():
                new_t[k] = v
                if k == "name":
                    new_t["description"] = descriptions[name]
            new_types.append(new_t)
        else:
            new_types.append(dict(t))

    missing = set(descriptions) - matched
    if missing:
        sys.exit(
            "ERROR: the following expected type names were not found in "
            f"taxonomy.json types[]: {sorted(missing)}"
        )

    new_taxonomy = dict(taxonomy)
    new_taxonomy["types"] = new_types
    return new_taxonomy


def write_variant(taxonomy: dict, out_path: Path) -> None:
    with out_path.open("w", encoding="utf-8") as f:
        json.dump(taxonomy, f, ensure_ascii=False, indent=2)
        f.write("\n")


def main() -> None:
    source = load_source_taxonomy(SOURCE_TAXONOMY)

    en_variant = inject_descriptions(source, DESCRIPTIONS_EN)
    write_variant(en_variant, SCRIPT_DIR / "taxonomy-descen.json")

    pt_variant = inject_descriptions(source, DESCRIPTIONS_PT)
    write_variant(pt_variant, SCRIPT_DIR / "taxonomy-descpt.json")

    print(f"Wrote {SCRIPT_DIR / 'taxonomy-descen.json'}")
    print(f"Wrote {SCRIPT_DIR / 'taxonomy-descpt.json'}")


if __name__ == "__main__":
    main()
