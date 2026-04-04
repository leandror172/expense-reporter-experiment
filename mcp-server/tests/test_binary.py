"""Tests for the expense-reporter binary wrapper.

These are integration tests that build and call the actual Go binary.
They validate the subprocess interface that the MCP tools depend on.
"""

import json
import pytest

from expense_mcp.binary import find_binary, run_binary, BinaryNotFoundError, BinaryError, _DATA_DIR


class TestFindBinary:
    """Test binary discovery logic."""

    def test_finds_binary_via_env_var(self, monkeypatch, tmp_path):
        """EXPENSE_REPORTER_BIN env var takes priority."""
        fake_bin = tmp_path / "expense-reporter"
        fake_bin.touch()
        fake_bin.chmod(0o755)
        monkeypatch.setenv("EXPENSE_REPORTER_BIN", str(fake_bin))
        assert find_binary() == str(fake_bin)

    def test_env_var_missing_file_raises(self, monkeypatch):
        """Env var pointing to non-existent file raises BinaryNotFoundError."""
        monkeypatch.setenv("EXPENSE_REPORTER_BIN", "/nonexistent/binary")
        with pytest.raises(BinaryNotFoundError):
            find_binary()

    def test_falls_back_to_go_build(self, monkeypatch):
        """Without env var, find_binary should locate or build from Go module."""
        monkeypatch.delenv("EXPENSE_REPORTER_BIN", raising=False)
        # This will either find a pre-built binary or build one
        # Just verify it returns a path (may fail if Go isn't installed)
        path = find_binary()
        assert path is not None


class TestRunBinary:
    """Integration tests that call the real Go binary with --json."""

    @pytest.fixture(autouse=True)
    def ensure_binary(self):
        """Build the binary once for all tests in this class."""
        self.binary = find_binary()

    def test_classify_returns_valid_json(self):
        """classify (via auto --json) returns parseable JSON with expected keys."""
        result = run_binary(
            self.binary,
            ["auto", "--json", "--data-dir", str(_DATA_DIR), "Uber Centro", "35,50", "15/04"],
        )
        data = json.loads(result.stdout)
        assert "item" in data
        assert "candidates" in data
        assert "action" in data

    def test_add_dry_run_returns_valid_json(self):
        """add --dry-run --json returns parseable JSON with expected keys."""
        result = run_binary(
            self.binary,
            ["add", "--dry-run", "--json", "Uber Centro;15/04;35,50;Uber/Taxi"],
        )
        data = json.loads(result.stdout)
        assert data["item"] == "Uber Centro"
        assert data["action"] == "would_insert"
        assert data["subcategory"] == "Uber/Taxi"

    def test_add_dry_run_invalid_input_returns_error(self):
        """add --dry-run with malformed input returns non-zero exit."""
        with pytest.raises(BinaryError) as exc_info:
            run_binary(
                self.binary,
                ["add", "--dry-run", "--json", "incomplete input"],
            )
        assert exc_info.value.exit_code != 0

    def test_nonexistent_command_raises(self):
        """Unknown subcommand raises BinaryError."""
        with pytest.raises(BinaryError):
            run_binary(self.binary, ["nonexistent"])
