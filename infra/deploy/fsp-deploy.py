#!/usr/bin/python3
"""Root-owned, forced-command deploy endpoint; receives SHA + docker save archive.

Install as /usr/local/sbin/fsp-deploy, owner root, mode 0755. The deploy user
can invoke this file through sudo, but cannot edit it, Compose, or the env file.
No shell command or arbitrary Compose configuration is accepted from CI.
"""

import gzip
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import tarfile
import time
import urllib.request

ROOT = Path('/opt/fsp-career')
COMPOSE = ROOT / 'compose.yaml'
ENV = Path('/etc/fsp-career/service.env')
CURRENT = ROOT / 'current-revision'
MAX_ARCHIVE_BYTES = 512 * 1024 * 1024
SHA = re.compile(r'[0-9a-f]{40}')


def run(args, *, env=None, timeout=300, capture=False):
    return subprocess.run(args, check=True, env=env, timeout=timeout,
                          text=True, capture_output=capture)


def compose(revision, *args, timeout=300, capture=False):
    env = dict(os.environ, FSP_REVISION=revision)
    return run(['docker', 'compose', '--project-name', 'fsp-career',
                '--env-file', str(ENV), '-f', str(COMPOSE), *args],
               env=env, timeout=timeout, capture=capture)


def validate_archive(archive, revision):
    with tarfile.open(archive, 'r:gz') as bundle:
        manifest_info = bundle.getmember('manifest.json')
        if not manifest_info.isfile() or manifest_info.size > 65536:
            raise ValueError('Invalid Docker image manifest')
        manifest = json.load(bundle.extractfile(manifest_info))
        expected = {f'fsp-career-api:{revision}', f'fsp-career-web:{revision}'}
        actual = set()
        if len(manifest) != 2:
            raise ValueError('Exactly two application images are required')
        for image in manifest:
            tags = image.get('RepoTags') or []
            if len(tags) != 1 or tags[0] not in expected:
                raise ValueError('Archive may only tag the two FSP images')
            actual.update(tags)
            config_info = bundle.getmember(image['Config'])
            if not config_info.isfile() or config_info.size > 1024 * 1024:
                raise ValueError('Invalid image configuration')
            config = json.load(bundle.extractfile(config_info))
            if config.get('config', {}).get('Labels', {}).get('org.opencontainers.image.revision') != revision:
                raise ValueError('Image revision label does not match the tested commit')
        if actual != expected:
            raise ValueError('Both application images must be present')
        # Docker's containerd store can prefer OCI names to manifest.json tags.
        # Check both formats so a bundle cannot retag a neighboring service.
        if 'index.json' in bundle.getnames():
            info = bundle.getmember('index.json')
            if not info.isfile() or info.size > 65536:
                raise ValueError('Invalid OCI index')
            index = json.load(bundle.extractfile(info))
            descriptors = index.get('manifests', [])
            if len(descriptors) != 2:
                raise ValueError('Exactly two OCI application references are required')
            names = set()
            for descriptor in descriptors:
                annotations = descriptor.get('annotations', {})
                name = annotations.get('io.containerd.image.name', '').removeprefix('docker.io/library/')
                if name not in expected or annotations.get('org.opencontainers.image.ref.name') != revision:
                    raise ValueError('OCI references may only name this FSP release')
                names.add(name)
            if names != expected:
                raise ValueError('Both OCI application references must be present')


def health(revision):
    for _ in range(30):
        try:
            with urllib.request.urlopen('http://127.0.0.1:18080/api/health', timeout=4) as response:
                if response.status != 200 or json.load(response).get('status') != 'ok':
                    raise ValueError('API dependencies are not healthy')
            with urllib.request.urlopen('http://127.0.0.1:18080/release.json', timeout=4) as response:
                if json.load(response).get('revision') != revision:
                    raise ValueError('The served frontend revision is incorrect')
            return
        except Exception:
            time.sleep(2)
    raise RuntimeError('Post-deployment health or revision check failed')


def backup(revision):
    container = compose(revision, 'ps', '-q', 'postgres', capture=True).stdout.strip()
    if not container:
        return
    target = ROOT / 'backups' / (time.strftime('%Y%m%dT%H%M%SZ', time.gmtime()) + '.sql.gz')
    with gzip.open(target, 'wb') as output:
        process = subprocess.Popen(['docker', 'exec', container, 'pg_dump', '-U', 'fsp', '-d', 'fsp'], stdout=subprocess.PIPE)
        try:
            shutil.copyfileobj(process.stdout, output)
            if process.wait(timeout=90) != 0:
                raise RuntimeError('Database backup failed; deployment aborted')
        finally:
            process.stdout.close()
            if process.poll() is None:
                process.kill()
    os.chmod(target, 0o600)
    print('Database backup created', flush=True)


def activate(revision):
    compose(revision, 'up', '-d', '--no-build', '--pull', 'never', '--wait', '--wait-timeout', '180')
    health(revision)


def retain_own_releases(revision, previous):
    keep = {revision, previous}
    history = (ROOT / 'deployments.jsonl').read_text().splitlines()
    recent = []
    for line in reversed(history):
        candidate = json.loads(line)['revision']
        if candidate not in recent:
            recent.append(candidate)
        if len(recent) == 5:
            break
    keep.update(recent)
    images = run(['docker', 'image', 'ls', '--format', '{{.Repository}}:{{.Tag}}'], capture=True).stdout.splitlines()
    for image in images:
        match = re.fullmatch(r'fsp-career-(api|web):([0-9a-f]{40})', image)
        if match and match.group(2) not in keep:
            # No force/prune: a still-used image is left in place.
            subprocess.run(['docker', 'image', 'rm', image], capture_output=True, timeout=30)
    backups = sorted((ROOT / 'backups').glob('*.sql.gz'), reverse=True)
    for path in backups[14:]:
        path.unlink()


def main():
    import fcntl

    if os.geteuid() != 0:
        raise PermissionError('Root-owned deployment endpoint must run through sudo')
    os.umask(0o077)
    header = sys.stdin.buffer.readline(128).decode('ascii').strip()
    match = re.fullmatch(r'(deploy|rollback) ([0-9a-f]{40})', header)
    if not match:
        raise ValueError('Expected deploy/rollback followed by a full commit SHA')
    action, revision = match.groups()
    with (ROOT / 'deploy.lock').open('a') as lock:
        fcntl.flock(lock, fcntl.LOCK_EX)
        previous = CURRENT.read_text().strip() if CURRENT.exists() else None
        if previous is not None and not SHA.fullmatch(previous):
            raise ValueError('Invalid previous revision')
        if action == 'deploy':
            archive = ROOT / 'incoming' / f'{revision}.tar.gz'
            try:
                total = 0
                with archive.open('wb') as target:
                    while chunk := sys.stdin.buffer.read(1024 * 1024):
                        total += len(chunk)
                        if total > MAX_ARCHIVE_BYTES:
                            raise ValueError('Image archive exceeds 512 MiB')
                        target.write(chunk)
                validate_archive(archive, revision)
                run(['docker', 'load', '--input', str(archive)])
            finally:
                archive.unlink(missing_ok=True)
        else:
            for name in ('api', 'web'):
                run(['docker', 'image', 'inspect', f'fsp-career-{name}:{revision}'], capture=True)
        if previous:
            backup(previous)
        try:
            activate(revision)
        except Exception:
            if previous:
                print(f'Deployment failed; restoring {previous}', flush=True)
                activate(previous)
                print('Previous application revision restored; database was preserved', flush=True)
            raise
        temp = ROOT / 'next-revision'
        temp.write_text(revision + '\n')
        temp.replace(CURRENT)
        with (ROOT / 'deployments.jsonl').open('a') as history:
            history.write(json.dumps({'time': time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime()),
                                      'action': action, 'revision': revision, 'previous': previous}) + '\n')
        try:
            retain_own_releases(revision, previous)
        except Exception as error:
            print(f'FSP retention deferred: {type(error).__name__}', flush=True)
        print(f'FSP_DEPLOYED {revision}', flush=True)


if __name__ == '__main__':
    try:
        main()
    except Exception as error:
        print(f'FSP deployment failed: {type(error).__name__}: {error}', file=sys.stderr)
        sys.exit(1)
