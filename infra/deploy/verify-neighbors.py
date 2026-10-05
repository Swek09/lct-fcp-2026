#!/usr/bin/python3
"""Capture and compare existing services without reading their secrets."""

import hashlib
import json
from pathlib import Path
import subprocess
import sys


def snapshot():
    containers = {}
    ids = subprocess.check_output(['docker', 'ps', '-q'], text=True).split()
    if ids:
        data = json.loads(subprocess.check_output(['docker', 'inspect', *ids], text=True))
        for container in data:
            if container['Config'].get('Labels', {}).get('com.docker.compose.project') == 'fsp-career':
                continue
            containers[container['Name']] = {'id': container['Id'], 'started': container['State']['StartedAt']}
    configs = {}
    for directory in ['/etc/nginx/sites-available', '/etc/nginx/conf.d']:
        for path in Path(directory).iterdir():
            if path.is_file() and path.name != 'fsp.swek.tech':
                configs[str(path)] = hashlib.sha256(path.read_bytes()).hexdigest()
    urls = {}
    for host in ['ai.swek.tech', 'ctfd.swek.tech', 'subtree.ru', 'api.subtree.ru', 'api.synthora.ru', 'swek.tech', 'synthora.ru']:
        urls[host] = subprocess.check_output(['curl', '-ksS', '--max-time', '10', '-o', '/dev/null', '-w', '%{http_code}',
                                             '--resolve', f'{host}:443:127.0.0.1', f'https://{host}/'], text=True)
    return {'containers': containers, 'nginx_configs': configs, 'http_status': urls}


if __name__ == '__main__':
    action, filename = sys.argv[1:]
    current = snapshot()
    if action == 'capture':
        Path(filename).write_text(json.dumps(current, indent=2) + '\n')
        print(f"Captured {len(current['containers'])} neighboring containers and {len(current['nginx_configs'])} configs")
    elif action == 'verify':
        previous = json.loads(Path(filename).read_text())
        for category, values in previous.items():
            for name, value in values.items():
                if current[category].get(name) != value:
                    raise SystemExit(f'Neighbor changed: {category}: {name}')
        print(f"Unchanged: {len(previous['containers'])} neighboring containers, all existing Nginx files and HTTP statuses")
    else:
        raise SystemExit('Use capture or verify')
