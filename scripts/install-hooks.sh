#!/bin/sh

# Install git hooks for codeplugs project
# Run this script after cloning the repository

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
HOOKS_DIR="$REPO_ROOT/.git/hooks"

echo "Installing git hooks..."

# Create pre-commit hook
cat > "$HOOKS_DIR/pre-commit" << 'EOF'
#!/bin/sh

# Pre-commit hook to run golangci-lint
# This ensures code quality before commits are made

set -e

echo "Running golangci-lint..."

# Check if golangci-lint is installed
if ! command -v golangci-lint >/dev/null 2>&1; then
    echo "golangci-lint not found. Installing..."
    # Try to install via go install
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest || {
        echo "Failed to install golangci-lint. Please install it manually:"
        echo "https://golangci-lint.run/usage/install/"
        exit 1
    }
fi

# Run golangci-lint
if ! golangci-lint run ./...; then
    echo ""
    echo "golangci-lint found issues. Please fix them before committing."
    echo "Run 'golangci-lint run ./...' to see details."
    exit 1
fi

echo "golangci-lint passed!"

# Optional: Run tests
# Uncomment the following lines if you want tests to run on every commit
# echo "Running tests..."
# if ! go test ./...; then
#     echo "Tests failed. Please fix them before committing."
#     exit 1
# fi
# echo "Tests passed!"

exit 0
EOF

chmod +x "$HOOKS_DIR/pre-commit"

echo "Pre-commit hook installed successfully!"
echo ""
echo "The hook will run golangci-lint before each commit."
echo "To skip the hook in an emergency, use: git commit --no-verify"
