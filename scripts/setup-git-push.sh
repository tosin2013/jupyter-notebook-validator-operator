#!/bin/bash
# Setup Git authentication for pushing to GitHub
# This script helps configure git credentials for automated pushes

set -e

echo "========================================="
echo "Git Push Authentication Setup"
echo "========================================="
echo ""

# Check current remote
REMOTE_URL=$(git remote get-url origin)
echo "📍 Current remote: $REMOTE_URL"
echo ""

# Check if using HTTPS or SSH
if [[ $REMOTE_URL == https://* ]]; then
    echo "🔐 Repository uses HTTPS authentication"
    echo ""
    echo "To push to GitHub via HTTPS, you need a Personal Access Token (PAT)."
    echo ""
    echo "📋 Steps to create a PAT:"
    echo "  1. Go to: https://github.com/settings/tokens"
    echo "  2. Click 'Generate new token' → 'Generate new token (classic)'"
    echo "  3. Give it a name (e.g., 'Operator Development')"
    echo "  4. Select scopes: 'repo' (full control of private repositories)"
    echo "  5. Click 'Generate token'"
    echo "  6. Copy the token (you won't see it again!)"
    echo ""
    echo "💡 Option 1: Use Git Credential Helper (Recommended)"
    echo "   Run: git config --global credential.helper store"
    echo "   Then: git push (you'll be prompted once for username and token)"
    echo ""
    echo "💡 Option 2: Use Environment Variable"
    echo "   Add to .env file:"
    echo "   export GITHUB_TOKEN=ghp_your_token_here"
    echo "   Then use: git push https://\$GITHUB_TOKEN@github.com/tosin2013/jupyter-notebook-validator-operator.git"
    echo ""
    echo "💡 Option 3: Switch to SSH (Recommended for long-term)"
    echo "   Run this script with: $0 --switch-to-ssh"
    echo ""
    
    if [ "$1" == "--switch-to-ssh" ]; then
        echo "🔄 Switching to SSH authentication..."
        SSH_URL=$(echo $REMOTE_URL | sed 's|https://github.com/|git@github.com:|')
        git remote set-url origin "$SSH_URL"
        echo "✅ Remote URL changed to: $SSH_URL"
        echo ""
        echo "📋 Next steps:"
        echo "  1. Generate SSH key if you don't have one:"
        echo "     ssh-keygen -t ed25519 -C 'your_email@example.com'"
        echo "  2. Add SSH key to GitHub:"
        echo "     cat ~/.ssh/id_ed25519.pub"
        echo "     Go to: https://github.com/settings/keys"
        echo "  3. Test connection:"
        echo "     ssh -T git@github.com"
        echo "  4. Try pushing:"
        echo "     git push origin \$(git branch --show-current)"
    fi
    
elif [[ $REMOTE_URL == git@* ]]; then
    echo "🔐 Repository uses SSH authentication"
    echo ""
    echo "📋 Checking SSH setup..."
    
    if [ -f ~/.ssh/id_ed25519 ] || [ -f ~/.ssh/id_rsa ]; then
        echo "✅ SSH key found"
        
        # Test SSH connection
        if ssh -T git@github.com 2>&1 | grep -q "successfully authenticated"; then
            echo "✅ SSH authentication working!"
            echo ""
            echo "You can now push with:"
            echo "  make git-push-rebuild MSG='your commit message'"
        else
            echo "⚠️  SSH key exists but authentication failed"
            echo ""
            echo "📋 Steps to fix:"
            echo "  1. Copy your public key:"
            echo "     cat ~/.ssh/id_ed25519.pub  # or ~/.ssh/id_rsa.pub"
            echo "  2. Add it to GitHub:"
            echo "     https://github.com/settings/keys"
            echo "  3. Test connection:"
            echo "     ssh -T git@github.com"
        fi
    else
        echo "❌ No SSH key found"
        echo ""
        echo "📋 Steps to create SSH key:"
        echo "  1. Generate key:"
        echo "     ssh-keygen -t ed25519 -C 'your_email@example.com'"
        echo "  2. Add to ssh-agent:"
        echo "     eval \"\$(ssh-agent -s)\""
        echo "     ssh-add ~/.ssh/id_ed25519"
        echo "  3. Copy public key:"
        echo "     cat ~/.ssh/id_ed25519.pub"
        echo "  4. Add to GitHub:"
        echo "     https://github.com/settings/keys"
        echo "  5. Test connection:"
        echo "     ssh -T git@github.com"
    fi
fi

echo ""
echo "========================================="
echo "Current Git Configuration"
echo "========================================="
git config --list | grep -E "user\.|credential\." || echo "No user or credential config found"
echo ""

echo "📝 Tip: After setting up authentication, you can push with:"
echo "  make git-push-rebuild MSG='your commit message'"

