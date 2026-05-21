from __future__ import annotations

import importlib.util
import os
import struct
import sys
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from vpcc import __main__ as vpcc_main


REPO_ROOT = Path(__file__).resolve().parents[1]


def _load_upstream_check():
    path = REPO_ROOT / ".github" / "scripts" / "upstream_check.py"
    spec = importlib.util.spec_from_file_location("upstream_check", path)
    assert spec and spec.loader
    module = importlib.util.module_from_spec(spec)
    sys.modules["upstream_check"] = module
    spec.loader.exec_module(module)
    return module


def _minimal_pe_with_bun(vsize: int, rsize: int, payload: bytes) -> bytes:
    pe_off = 0x80
    coff = pe_off + 4
    opt_size = 0
    sect_table = coff + 20 + opt_size
    raw_off = 0x200
    data = bytearray(raw_off + rsize)

    data[:2] = b"MZ"
    struct.pack_into("<I", data, 0x3C, pe_off)
    data[pe_off:pe_off + 4] = b"PE\x00\x00"
    struct.pack_into("<H", data, coff + 2, 1)      # NumberOfSections
    struct.pack_into("<H", data, coff + 16, opt_size)

    s = sect_table
    data[s:s + 8] = b".bun\x00\x00\x00\x00"
    struct.pack_into("<I", data, s + 8, vsize)     # VirtualSize
    struct.pack_into("<I", data, s + 16, rsize)    # SizeOfRawData
    struct.pack_into("<I", data, s + 20, raw_off)  # PointerToRawData

    data[raw_off:raw_off + len(payload)] = payload
    return bytes(data)


class BunPERegressionTests(unittest.TestCase):
    def test_pe_bun_section_uses_virtual_size_not_raw_padding(self) -> None:
        vsize = 96
        rsize = 128
        payload = b"A" * (vsize - len(vpcc_main._BUN_TRAILER)) + vpcc_main._BUN_TRAILER
        data = _minimal_pe_with_bun(vsize, rsize, payload)

        off, size = vpcc_main._find_bun_section_pe(bytearray(data))

        self.assertEqual(off, 0x200)
        self.assertEqual(size, vsize)

    def test_bun_trailer_validation_ignores_pe_alignment_padding(self) -> None:
        vsize = 96
        rsize = 128
        payload = b"A" * (vsize - len(vpcc_main._BUN_TRAILER)) + vpcc_main._BUN_TRAILER
        data = _minimal_pe_with_bun(vsize, rsize, payload)
        raw_off = 0x200

        self.assertTrue(vpcc_main._bun_section_has_valid_trailer(data, raw_off, raw_off + rsize))


class McpGuardRegressionTests(unittest.TestCase):
    def test_mcp_guard_dry_run_succeeds_without_installed_preload(self) -> None:
        with tempfile.TemporaryDirectory() as home:
            with patch.object(Path, "home", return_value=Path(home)):
                ok, msg = vpcc_main._apply_mcp_guard({}, dry_run=True)

            self.assertTrue(ok)
            self.assertIn("would", msg)
            self.assertFalse((Path(home) / ".local/share/void-patcher/claude-preload.js").exists())


class WrapperRegressionTests(unittest.TestCase):
    def test_wrapper_install_creates_parent_directory(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            wrapper = Path(tmp) / ".local" / "bin" / "claude"

            ok, msg = vpcc_main._apply_wrapper({"wrapper_path": str(wrapper)}, "bun_sea", None)

            self.assertTrue(ok, msg)
            self.assertTrue(wrapper.exists())
            self.assertTrue(os.access(wrapper, os.X_OK))


class UpstreamCheckRegressionTests(unittest.TestCase):
    def test_meta_failures_do_not_count_as_broken_signatures(self) -> None:
        upstream_check = _load_upstream_check()
        rows = [
            {"status": "ok", "id": "js-good", "msg": "would apply 1 in-place"},
            {"status": "ok", "id": "js-zero", "msg": "would apply 0 in-place"},
            {"status": "fail", "id": "mcp-guard", "msg": "preload format unexpected"},
        ]
        patch_types = {"js-good": "js_replace", "js-zero": "js_replace", "mcp-guard": "mcp_guard"}

        classified = upstream_check.classify_rows(rows, patch_types)

        self.assertEqual([r["id"] for r in classified["broken_signatures"]], ["js-zero"])
        self.assertEqual([r["id"] for r in classified["meta_failures"]], ["mcp-guard"])
        self.assertEqual(classified["needs_issue"], 2)


class VerifyRegressionTests(unittest.TestCase):
    def test_verify_reports_failure_when_no_patches_applied(self) -> None:
        """cmd_verify must return non-zero when optional patches are all unapplied (batch failure)."""
        from unittest.mock import MagicMock

        vsize = 256
        rsize = 256
        trailer = vpcc_main._BUN_TRAILER
        payload = b"\x00" * (vsize - len(trailer)) + trailer
        pe_data = _minimal_pe_with_bun(vsize, rsize, payload)

        with tempfile.TemporaryDirectory() as tmp:
            binary = Path(tmp) / "claude.exe"
            binary.write_bytes(pe_data)

            patch_obj = {
                "id": "test-optional-patch",
                "type": "js_replace",
                "patches": [
                    {
                        "search_regex": "nonexistent_pattern",
                        "replace": "replacement",
                        "applied_marker": "nonexistent_marker_xyz_unique",
                        "required": False,
                        "count": 0,
                    }
                ],
                "anchor_strings": ["nonexistent_pattern"],
            }

            with (
                patch.object(vpcc_main, "find_target", return_value=(binary, "bun_sea")),
                patch.object(vpcc_main, "load_patches", return_value=[patch_obj]),
            ):
                rc = vpcc_main.cmd_verify(MagicMock())

        # optional_missing (1) > applied (0) → systemic failure → exit 1
        self.assertEqual(rc, 1)

    def test_verify_succeeds_when_marker_present(self) -> None:
        """cmd_verify must return 0 when the applied_marker is found in the binary."""
        from unittest.mock import MagicMock

        marker = b"nonexistent_marker_xyz_unique"
        vsize = 256
        rsize = 256
        trailer = vpcc_main._BUN_TRAILER
        filler = b"\x00" * (vsize - len(marker) - len(trailer))
        payload = marker + filler + trailer
        pe_data = _minimal_pe_with_bun(vsize, rsize, payload)

        with tempfile.TemporaryDirectory() as tmp:
            binary = Path(tmp) / "claude.exe"
            binary.write_bytes(pe_data)

            patch_obj = {
                "id": "test-optional-patch",
                "type": "js_replace",
                "patches": [
                    {
                        "search_regex": "nonexistent_pattern",
                        "replace": "replacement",
                        "applied_marker": "nonexistent_marker_xyz_unique",
                        "required": False,
                        "count": 0,
                    }
                ],
                "anchor_strings": ["nonexistent_pattern"],
            }

            with (
                patch.object(vpcc_main, "find_target", return_value=(binary, "bun_sea")),
                patch.object(vpcc_main, "load_patches", return_value=[patch_obj]),
            ):
                rc = vpcc_main.cmd_verify(MagicMock())

        # marker present → applied (1) > optional_missing (0) → exit 0
        self.assertEqual(rc, 0)


if __name__ == "__main__":
    unittest.main()
