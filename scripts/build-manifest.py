#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
# SPDX-License-Identifier: Apache-2.0
"""Bind CE artifact bytes to their compiled identity and exact source qualification.

The workflow supplies a qualification only after its checks succeed. A snapshot
without one remains explicitly unqualified. This manifest is not a signature.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import shlex
import subprocess

PKG = 'github.com/choreoatlas2025/cli/internal/cli.'
CHECKS = ['lint', 'tests', 'race', 'vet', 'native-cli', 'browser', 'example', 'manifest-tests']


def git_value(*args):
    return subprocess.check_output(['git', *args], text=True).strip()


def digest(path):
    h = hashlib.sha256()
    with open(path, 'rb') as f:
        for block in iter(lambda: f.read(1 << 20), b''):
            h.update(block)
    return 'sha256:' + h.hexdigest()


def full_commit(value):
    if not re.fullmatch(r'[0-9a-f]{40}', value):
        raise ValueError('full 40-character commit is required')
    return value


def check_qualification(record, commit, tree):
    if (record.get('status') != 'passed' or record.get('gitCommit') != commit
            or record.get('gitTree') != tree or record.get('edition') != 'ce'
            or record.get('checks') != CHECKS):
        raise ValueError('qualification does not match exact source revision or required checks')


def compiled_identity(binary):
    text = subprocess.check_output(['go', 'version', '-m', str(binary)], text=True)
    build = {}
    for line in text.splitlines():
        if line.startswith('\tbuild\t'):
            key, value = line[len('\tbuild\t'):].split('=', 1)
            build[key] = value
    flags = shlex.split(build.get('-ldflags', ''))
    if len(flags) == 1:  # go version quotes the entire link flags value
        flags = shlex.split(flags[0])
    identity = {}
    for i, flag in enumerate(flags):
        if flag == '-X' and i + 1 < len(flags):
            key, value = flags[i + 1].split('=', 1)
            if key.startswith(PKG):
                name = key[len(PKG):]
                if name in identity:
                    raise ValueError('duplicate compiled metadata: ' + name)
                identity[name] = value
    return identity, build


def make_manifest(binaries, commit, channel, version, qualification=None):
    full_commit(commit)
    tree = git_value('rev-parse', 'HEAD^{tree}')
    if commit != git_value('rev-parse', 'HEAD'):
        raise ValueError('artifact commit differs from checked-out source')
    record = None
    if qualification:
        record = json.loads(Path(qualification).read_text())
        check_qualification(record, commit, tree)
        if git_value('status', '--porcelain', '--untracked-files=normal'):
            raise ValueError('qualified source must be clean')
    artifacts = []
    for binary in binaries:
        binary = Path(binary)
        identity, build = compiled_identity(binary)
        expected = {'GitCommit': commit, 'BuildChannel': channel, 'BuildEdition': 'ce', 'Version': version}
        if any(identity.get(key) != value for key, value in expected.items()):
            raise ValueError('compiled artifact metadata mismatch: ' + str(binary))
        if record and (build.get('vcs.revision') != commit or build.get('vcs.modified') != 'false'):
            raise ValueError('binary was not built from the qualified clean source')
        artifacts.append({'name': binary.name, 'sha256': digest(binary), 'size': binary.stat().st_size,
                          'os': build.get('GOOS'), 'arch': build.get('GOARCH'), 'sourceDirty': build.get('vcs.modified') != 'false', 'compiled': identity})
    return {'formatVersion': 1, 'edition': 'ce', 'gitCommit': commit, 'gitTree': tree,
            'version': version, 'buildChannel': channel, 'qualified': record is not None,
            'qualification': record, 'qualificationHash': digest(qualification) if record else None,
            'artifacts': artifacts}


def verify_manifest(record, directory, commit, snapshot=False):
    full_commit(commit)
    if record.get('formatVersion') != 1 or record.get('edition') != 'ce' or record.get('gitCommit') != commit:
        raise ValueError('manifest identity mismatch')
    qualified = record.get('qualified') is True
    if not snapshot and not qualified:
        raise ValueError('unqualified snapshot cannot be consumed as a release')
    if qualified:
        qualification = record.get('qualification')
        check_qualification(qualification or {}, commit, record.get('gitTree'))
        encoded = (json.dumps(qualification, indent=2) + '\n').encode()
        if record.get('qualificationHash') != 'sha256:' + hashlib.sha256(encoded).hexdigest():
            raise ValueError('qualification hash mismatch')
    if not record.get('artifacts'):
        raise ValueError('manifest contains no artifacts')
    seen = set()
    for artifact in record['artifacts']:
        name = artifact['name']
        if Path(name).name != name or name in seen:
            raise ValueError('invalid artifact name')
        seen.add(name)
        path = Path(directory) / name
        if artifact['sha256'] != digest(path) or artifact['size'] != path.stat().st_size:
            raise ValueError('artifact checksum mismatch: ' + name)
        identity = artifact['compiled']
        expected = {'GitCommit': commit, 'BuildChannel': record['buildChannel'], 'BuildEdition': 'ce', 'Version': record['version']}
        if any(identity.get(key) != value for key, value in expected.items()):
            raise ValueError('artifact identity mismatch: ' + name)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--qualify', action='store_true')
    parser.add_argument('--binary', action='append', default=[])
    parser.add_argument('--commit', required=True)
    parser.add_argument('--channel')
    parser.add_argument('--version')
    parser.add_argument('--out')
    parser.add_argument('--verify', action='store_true')
    parser.add_argument('--manifest')
    parser.add_argument('--artifact-dir', default='.')
    parser.add_argument('--qualification', default='')
    parser.add_argument('--snapshot', choices=['true', 'false'], default='false')
    args = parser.parse_args()
    if args.verify:
        if not args.manifest:
            parser.error('manifest is required for verification')
        verify_manifest(json.loads(Path(args.manifest).read_text()), args.artifact_dir, args.commit, args.snapshot == 'true')
        print('Artifact identity and checksums verified')
        return
    if not args.out:
        parser.error('out is required')
    if args.qualify:
        full_commit(args.commit)
        if git_value('rev-parse', 'HEAD') != args.commit or git_value('status', '--porcelain'):
            raise ValueError('qualification requires exact clean source')
        record = {'edition': 'ce', 'gitCommit': args.commit, 'gitTree': git_value('rev-parse', 'HEAD^{tree}'),
                  'status': 'passed', 'checks': CHECKS, 'repository': os.environ.get('GITHUB_REPOSITORY'),
                  'runId': os.environ.get('GITHUB_RUN_ID'), 'runAttempt': os.environ.get('GITHUB_RUN_ATTEMPT')}
    else:
        if not args.binary or not args.channel or not args.version:
            parser.error('binary, channel and version are required')
        if args.snapshot == 'false' and not args.qualification:
            parser.error('non-snapshot builds require source qualification')
        record = make_manifest(args.binary, args.commit, args.channel, args.version, args.qualification or None)
    Path(args.out).write_text(json.dumps(record, indent=2) + '\n')


if __name__ == '__main__':
    main()
