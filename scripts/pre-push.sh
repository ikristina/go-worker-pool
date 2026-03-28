#!/bin/bash
echo "Refining the backbone: Running fmt, vet, and race tests..."
make lint
make test
if [ $? -ne 0 ]; then
    echo "❌ Push aborted: Tests or Linting failed. Fix the issues before pushing!"
    exit 1
fi
echo "✅ All checks passed. Pushing to GitHub..."
