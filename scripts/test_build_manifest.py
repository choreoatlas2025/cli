# SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
# SPDX-License-Identifier: Apache-2.0
import copy
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

loader = importlib.util.spec_from_file_location('build_manifest', Path(__file__).with_name('build-manifest.py'))
m = importlib.util.module_from_spec(loader)
loader.loader.exec_module(m)


class ManifestTests(unittest.TestCase):
    def test_exact_revision_and_complete_checks(self):
        good = {'status': 'passed', 'gitCommit': 'a'*40, 'gitTree': 'b'*40, 'edition': 'ce', 'checks': m.CHECKS}
        m.check_qualification(good, 'a'*40, 'b'*40)
        for key, value in [('gitCommit', 'c'*40), ('gitTree', 'c'*40), ('checks', ['tests']), ('status', 'failed')]:
            bad = dict(good, **{key: value})
            with self.assertRaises(ValueError):
                m.check_qualification(bad, 'a'*40, 'b'*40)
        with self.assertRaises(ValueError):
            m.full_commit('a'*7)

    def test_snapshot_and_tampered_artifact(self):
        with tempfile.TemporaryDirectory() as task_tmp:
            path = Path(task_tmp) / 'choreoatlas'
            path.write_bytes(b'compiled artifact')
            identity = {'GitCommit': 'a'*40, 'BuildChannel': 'make', 'BuildEdition': 'ce', 'Version': 'fixture-ce'}
            def git(*args):
                return 'b'*40 if args[-1] == 'HEAD^{tree}' else 'a'*40
            with patch.object(m, 'git_value', side_effect=git), patch.object(m, 'compiled_identity', return_value=(identity, {'vcs.modified': 'true'})):
                record = m.make_manifest([path], 'a'*40, 'make', 'fixture-ce')
            m.verify_manifest(record, task_tmp, 'a'*40, True)
            with self.assertRaises(ValueError):
                m.verify_manifest(record, task_tmp, 'a'*40)
            with self.assertRaises(ValueError):
                m.verify_manifest(record, task_tmp, 'c'*40, True)
            bad = copy.deepcopy(record)
            bad['artifacts'][0]['compiled']['GitCommit'] = 'c'*40
            with self.assertRaises(ValueError):
                m.verify_manifest(bad, task_tmp, 'a'*40, True)
            path.write_bytes(b'tampered artifact')
            with self.assertRaises(ValueError):
                m.verify_manifest(record, task_tmp, 'a'*40, True)

    def test_qualified_hash_and_dirty_source(self):
        with tempfile.TemporaryDirectory() as task_tmp:
            path = Path(task_tmp) / 'choreoatlas'; path.write_bytes(b'compiled')
            q = {'status': 'passed', 'gitCommit': 'a'*40, 'gitTree': 'b'*40, 'edition': 'ce', 'checks': m.CHECKS}
            qp = Path(task_tmp) / 'qualification.json'; qp.write_text(json.dumps(q, indent=2)+'\n')
            identity = {'GitCommit': 'a'*40, 'BuildChannel': 'goreleaser', 'BuildEdition': 'ce', 'Version': 'fixture-ce'}
            def git(*args):
                if args[0] == 'status': return ''
                return 'b'*40 if args[-1] == 'HEAD^{tree}' else 'a'*40
            with patch.object(m, 'git_value', side_effect=git), patch.object(m, 'compiled_identity', return_value=(identity, {'vcs.modified': 'false', 'vcs.revision': 'a'*40})):
                record = m.make_manifest([path], 'a'*40, 'goreleaser', 'fixture-ce', qp)
            m.verify_manifest(record, task_tmp, 'a'*40)
            record['qualification']['checks'] = ['tests']
            with self.assertRaises(ValueError): m.verify_manifest(record, task_tmp, 'a'*40)
            with patch.object(m, 'git_value', side_effect=lambda *args: ' M source.go' if args[0] == 'status' else git(*args)):
                with self.assertRaises(ValueError): m.make_manifest([path], 'a'*40, 'goreleaser', 'fixture-ce', qp)


if __name__ == '__main__':
    unittest.main()
