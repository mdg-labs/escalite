#!/usr/bin/env bash
# Configure repo-local SSH commit signing for Escalite.
# Run from repo root after registering your SSH public key on GitHub as a Signing key.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

PUBKEY="${HOME}/.ssh/id_ed25519.pub"

if [[ ! -f "$PUBKEY" ]] && [[ -f "${HOME}/.ssh/id_ed25519" ]]; then
  echo "Deriving ${PUBKEY} from private key..."
  ssh-keygen -y -f "${HOME}/.ssh/id_ed25519" >"$PUBKEY"
  chmod 644 "$PUBKEY"
fi

if [[ ! -f "$PUBKEY" ]]; then
  echo "error: no SSH public key at ${PUBKEY}" >&2
  echo "Set user.signingkey to your signing public key path, or create id_ed25519." >&2
  exit 1
fi

git config gpg.format ssh
git config user.signingkey "$PUBKEY"
git config commit.gpgsign true

echo "Configured repo-local SSH commit signing:"
git config --local --list | grep -E '^(gpg\.|commit\.gpgsign|user\.signingkey)'
