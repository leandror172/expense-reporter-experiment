"""Entry point for `python -m expense_mcp`."""

from expense_mcp.server import mcp

mcp.run(transport="stdio")
