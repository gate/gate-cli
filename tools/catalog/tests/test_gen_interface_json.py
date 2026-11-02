"""生成器失败必须阻断构建，不能默默丢掉权限条目。"""
import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

SCRIPT = Path(__file__).parents[1] / "gen_interface_json.py"

class CatalogGenerationTest(unittest.TestCase):
    def run_generator(self, content):
        directory = tempfile.TemporaryDirectory()
        self.addCleanup(directory.cleanup)
        root = Path(directory.name)
        source = root / "openapi.yaml"
        source.write_text(content)
        output = root / "interface.json"
        result = subprocess.run([sys.executable,str(SCRIPT),"--src",str(source),"--out",str(output)],capture_output=True,text=True,timeout=5)
        return result, output

    def test_single_sdk_file_and_explicit_public(self):
        result, output = self.run_generator("""openapi: 3.0.0
paths:
  /stock/exchanges:
    get:
      operationId: queryStockExchanges
      security: []
  /stock/users/assets:
    get:
      operationId: queryStockUserAssets
""")
        self.assertEqual(result.returncode,0,result.stderr)
        operations=json.loads(output.read_text())["operations"]
        self.assertEqual([(x["path"],x["auth_required"]) for x in operations],[("/api/v4/stock/exchanges",False),("/api/v4/stock/users/assets",True)])

    def test_invalid_and_duplicate_contracts_fail_closed(self):
        for content in ["paths: [", "paths: []", "paths: {}", "paths: { /stock/exchanges: { get: {security: []} } }", """paths:
  /stock/exchanges:
    get: {operationId: first, security: []}
  /stock/exchanges:
    get: {operationId: second, security: []}
"""]:
            with self.subTest(content=content):
                result, output=self.run_generator(content)
                self.assertNotEqual(result.returncode,0)
                self.assertFalse(output.exists(),"invalid contract must not produce a partial catalog")

if __name__ == "__main__":
    unittest.main()
