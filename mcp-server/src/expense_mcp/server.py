"""FastMCP server exposing expense-reporter Go binary as MCP tools.

Two tools: classify_expense (candidates + recommendation) and add_expense (insert).
Each tool shells out to the Go binary with --json and returns the output directly.
"""

import json
from contextlib import asynccontextmanager
from typing import AsyncIterator

from mcp.server.fastmcp import FastMCP

from expense_mcp.binary import (
    find_binary,
    run_binary,
    BinaryNotFoundError,
    BinaryError,
    _DATA_DIR,
)


@asynccontextmanager
async def lifespan(server: FastMCP) -> AsyncIterator[dict]:
    """Locate the expense-reporter binary once at startup."""
    try:
        binary_path = find_binary()
    except BinaryNotFoundError as e:
        raise RuntimeError(f"Binary not found: {e}") from e
    yield {"binary_path": binary_path}


mcp = FastMCP("expense-reporter", lifespan=lifespan)


@mcp.tool()
async def classify_expense(
    item: str,
    value: str,
    date: str,
    model: str | None = None,
    top: int | None = None,
) -> str:
    """Classify an expense and get a recommendation.

    Returns candidates with confidence scores and a recommendation
    (would_insert / review / excluded) based on confidence threshold.

    Args:
        item: Expense description (e.g., "Uber Centro")
        value: Amount in Brazilian format (e.g., "35,50")
        date: Date as DD/MM (e.g., "15/04")
        model: Ollama model override (default: my-classifier-q3)
        top: Number of top candidates to return
    """
    ctx = mcp.get_context()
    binary_path = ctx.request_context.lifespan_context["binary_path"]

    args = ["auto", "--json", "--data-dir", str(_DATA_DIR)]
    if model is not None:
        args.extend(["--model", model])
    if top is not None:
        args.extend(["--top", str(top)])
    args.extend([item, value, date])

    try:
        result = run_binary(binary_path, args)
        return result.stdout
    except BinaryError as e:
        return json.dumps({"error": e.stderr, "exit_code": e.exit_code})


@mcp.tool()
async def add_expense(
    item: str,
    date: str,
    value: str,
    subcategory: str,
    workbook: str | None = None,
    dry_run: bool = False,
) -> str:
    """Add an expense to the workbook.

    Assembles the expense string and calls the Go binary to insert.
    Use dry_run=True to validate without inserting.

    Args:
        item: Expense description (e.g., "Uber Centro")
        date: Date as DD/MM (e.g., "15/04")
        value: Amount in Brazilian format (e.g., "35,50")
        subcategory: Target subcategory (e.g., "Uber/Taxi")
        workbook: Path to Excel workbook (optional, uses env default)
        dry_run: Validate and parse without inserting
    """
    ctx = mcp.get_context()
    binary_path = ctx.request_context.lifespan_context["binary_path"]

    expense_string = f"{item};{date};{value};{subcategory}"
    args = ["add", "--json", "--data-dir", str(_DATA_DIR)]
    if dry_run:
        args.append("--dry-run")
    if workbook is not None:
        args.extend(["--workbook", workbook])
    args.append(expense_string)

    try:
        result = run_binary(binary_path, args)
        return result.stdout
    except BinaryError as e:
        return json.dumps({"error": e.stderr, "exit_code": e.exit_code})
