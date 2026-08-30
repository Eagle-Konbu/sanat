#!/bin/bash
set -e

echo "Setting up 1Password SSH/GPG agent integration..."

# Ensure .ssh directory exists
mkdir -p ~/.ssh
chmod 700 ~/.ssh

# Configure SSH to use 1Password agent
cat > ~/.ssh/config <<EOF
Host *
    IdentityAgent /tmp/1password-agent.sock
EOF

chmod 600 ~/.ssh/config

# Verify SSH agent is accessible
if [ -S "$SSH_AUTH_SOCK" ]; then
    echo "✓ SSH agent socket is accessible at: $SSH_AUTH_SOCK"
else
    echo "⚠ Warning: SSH agent socket not found at: $SSH_AUTH_SOCK"
    echo "  Make sure 1Password SSH agent is enabled on the host machine"
fi

# Test SSH agent connection
if ssh-add -l >/dev/null 2>&1; then
    echo "✓ SSH agent is working"
    echo "Available SSH keys:"
    ssh-add -l
else
    echo "⚠ Warning: Cannot connect to SSH agent"
    echo "  Ensure 1Password SSH agent is running on the host"
fi

echo "1Password SSH/GPG setup complete!"
