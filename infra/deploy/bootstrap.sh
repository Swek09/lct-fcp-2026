#!/bin/bash
# One-time administrator setup. Run from this directory with the public key path.
set -euo pipefail
test "$(id -u)" = 0
SOURCE=$(cd -- "$(dirname -- "$0")" && pwd)
PUBKEY=${1:?Supply the dedicated deploy public key file}
grep -Eq '^ssh-ed25519 [A-Za-z0-9+/=]+( .*)?$' "$PUBKEY"

install -d -m 700 /opt/fsp-career
if [[ ! -f /opt/fsp-career/neighbors-before.json ]]; then
    python3 "$SOURCE/verify-neighbors.py" capture /opt/fsp-career/neighbors-before.json
fi
install -d -m 700 /opt/fsp-career/incoming /opt/fsp-career/backups /etc/fsp-career
install -d -m 755 /var/lib/fsp-career/acme
install -m 644 "$SOURCE/compose.deploy.yaml" /opt/fsp-career/compose.yaml
install -m 755 "$SOURCE/fsp-deploy.py" /usr/local/sbin/fsp-deploy
install -m 755 "$SOURCE/verify-neighbors.py" /opt/fsp-career/verify-neighbors.py

if [[ ! -f /etc/fsp-career/service.env ]]; then
    python3 - <<'PY'
import os, secrets
os.umask(0o077)
with open('/etc/fsp-career/service.env', 'x') as output:
    output.write('POSTGRES_PASSWORD=' + secrets.token_hex(32) + '\nAPP_ENV=demo\nDEMO_SEED=true\nSMTP_HOST=mailpit\nSMTP_PORT=1025\nSMTP_FROM=platform@fsp.swek.tech\n')
PY
fi
if ! id fsp-deploy >/dev/null 2>&1; then
    useradd --create-home --shell /bin/bash fsp-deploy
fi
# Root ownership prevents replacement of the forced command by the deploy user.
chown root:root /home/fsp-deploy
chmod 755 /home/fsp-deploy
install -d -o root -g root -m 755 /home/fsp-deploy/.ssh
printf 'restrict,command="sudo -n /usr/local/sbin/fsp-deploy" %s\n' "$(cat "$PUBKEY")" > /home/fsp-deploy/.ssh/authorized_keys
chown root:root /home/fsp-deploy/.ssh/authorized_keys
chmod 644 /home/fsp-deploy/.ssh/authorized_keys
printf 'fsp-deploy ALL=(root) NOPASSWD: /usr/local/sbin/fsp-deploy\n' > /etc/sudoers.d/fsp-career-deploy
chmod 440 /etc/sudoers.d/fsp-career-deploy
visudo -cf /etc/sudoers.d/fsp-career-deploy

if [[ ! -f /etc/letsencrypt/live/fsp.swek.tech/fullchain.pem ]]; then
    # Only this domain's file is added; Certbot never edits neighboring vhosts.
    cat > /etc/nginx/sites-available/fsp.swek.tech <<'NGINX'
server {
    listen 80;
    listen [::]:80;
    server_name fsp.swek.tech;
    location ^~ /.well-known/acme-challenge/ {
        root /var/lib/fsp-career/acme;
        default_type text/plain;
    }
    location / { return 503; }
}
NGINX
    ln -sfn /etc/nginx/sites-available/fsp.swek.tech /etc/nginx/sites-enabled/fsp.swek.tech
    nginx -t
    systemctl reload nginx
    certbot certonly --webroot -w /var/lib/fsp-career/acme \
        -d fsp.swek.tech --cert-name fsp.swek.tech \
        --non-interactive --agree-tos --register-unsafely-without-email
fi
install -m 644 "$SOURCE/fsp.swek.tech.conf" /etc/nginx/sites-available/fsp.swek.tech
ln -sfn /etc/nginx/sites-available/fsp.swek.tech /etc/nginx/sites-enabled/fsp.swek.tech
nginx -t
systemctl reload nginx

# This hook reloads only Nginx after a successful renewal of the FSP certificate.
install -d -m 755 /etc/letsencrypt/renewal-hooks/deploy
cat > /etc/letsencrypt/renewal-hooks/deploy/fsp-career <<'HOOK'
#!/bin/sh
if [ "$RENEWED_LINEAGE" = /etc/letsencrypt/live/fsp.swek.tech ]; then
    /usr/sbin/nginx -t && /usr/bin/systemctl reload nginx
fi
HOOK
chmod 755 /etc/letsencrypt/renewal-hooks/deploy/fsp-career

# Pull only our dependencies; running containers from other projects are untouched.
FSP_REVISION=0000000000000000000000000000000000000000 \
    docker compose --project-name fsp-career --env-file /etc/fsp-career/service.env \
        -f /opt/fsp-career/compose.yaml pull postgres redis mailpit
python3 /opt/fsp-career/verify-neighbors.py verify /opt/fsp-career/neighbors-before.json
echo 'FSP bootstrap complete; ready to receive tested image releases'
