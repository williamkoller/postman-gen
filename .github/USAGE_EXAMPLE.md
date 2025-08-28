# Using postman-gen in CI/CD

## GitHub Actions Integration

Here's how you can integrate `postman-gen` into your CI/CD pipeline to automatically generate and update Postman collections:

### Basic Integration

```yaml
name: Update Postman Collection

on:
  push:
    branches: [main]
    paths:
      - '**/*.go' # Only run when Go files change

jobs:
  update-postman:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Download postman-gen
        run: |
          curl -L -o postman-gen https://github.com/williamkoller/postman-gen/releases/latest/download/postman-gen-linux-amd64
          chmod +x postman-gen

      - name: Generate Postman Collection
        run: |
          ./postman-gen \
            -dir . \
            -name "My API v${{ github.sha }}" \
            -base-url "https://api.myapp.com" \
            -group-depth 2 \
            -group-by-method \
            -tag-folders \
            -out postman-collection.json \
            -env-out postman-environment.json \
            -env-name "Production"

      - name: Upload artifacts
        uses: actions/upload-artifact@v3
        with:
          name: postman-collections
          path: |
            postman-collection.json
            postman-environment.json
```

### Advanced Integration with Postman API

```yaml
name: Deploy to Postman

on:
  release:
    types: [published]

jobs:
  deploy-to-postman:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Download postman-gen
        run: |
          curl -L -o postman-gen https://github.com/williamkoller/postman-gen/releases/latest/download/postman-gen-linux-amd64
          chmod +x postman-gen

      - name: Generate Postman Collection
        run: |
          ./postman-gen \
            -dir . \
            -name "My API ${{ github.event.release.tag_name }}" \
            -base-url "https://api.myapp.com" \
            -group-depth 2 \
            -group-by-method \
            -tag-folders \
            -out collection.json \
            -env-out environment.json \
            -env-name "Production"

      - name: Update Postman Collection
        env:
          POSTMAN_API_KEY: ${{ secrets.POSTMAN_API_KEY }}
          COLLECTION_UID: ${{ secrets.POSTMAN_COLLECTION_UID }}
        run: |
          # Update collection via Postman API
          curl -X PUT \
            "https://api.getpostman.com/collections/$COLLECTION_UID" \
            -H "X-Api-Key: $POSTMAN_API_KEY" \
            -H "Content-Type: application/json" \
            -d @collection.json

      - name: Update Postman Environment
        env:
          POSTMAN_API_KEY: ${{ secrets.POSTMAN_API_KEY }}
          ENVIRONMENT_UID: ${{ secrets.POSTMAN_ENVIRONMENT_UID }}
        run: |
          # Update environment via Postman API
          curl -X PUT \
            "https://api.getpostman.com/environments/$ENVIRONMENT_UID" \
            -H "X-Api-Key: $POSTMAN_API_KEY" \
            -H "Content-Type: application/json" \
            -d @environment.json
```

### Multi-Environment Setup

```yaml
name: Multi-Environment Postman Collections

on:
  push:
    branches: [main, develop, staging]

jobs:
  generate-collections:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        environment:
          - name: development
            base_url: https://dev-api.myapp.com
            branch: develop
          - name: staging
            base_url: https://staging-api.myapp.com
            branch: staging
          - name: production
            base_url: https://api.myapp.com
            branch: main

    if: github.ref == format('refs/heads/{0}', matrix.environment.branch)

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Download postman-gen
        run: |
          curl -L -o postman-gen https://github.com/williamkoller/postman-gen/releases/latest/download/postman-gen-linux-amd64
          chmod +x postman-gen

      - name: Generate Postman Collection for ${{ matrix.environment.name }}
        run: |
          ./postman-gen \
            -dir . \
            -name "My API (${{ matrix.environment.name }})" \
            -base-url "${{ matrix.environment.base_url }}" \
            -group-depth 2 \
            -group-by-method \
            -tag-folders \
            -out "collection-${{ matrix.environment.name }}.json" \
            -env-out "environment-${{ matrix.environment.name }}.json" \
            -env-name "${{ matrix.environment.name }}"

      - name: Upload ${{ matrix.environment.name }} artifacts
        uses: actions/upload-artifact@v3
        with:
          name: postman-${{ matrix.environment.name }}
          path: |
            collection-${{ matrix.environment.name }}.json
            environment-${{ matrix.environment.name }}.json
```

## Secrets Configuration

To use the Postman API integration, add these secrets to your GitHub repository:

1. `POSTMAN_API_KEY`: Your Postman API key
2. `POSTMAN_COLLECTION_UID`: The UID of your Postman collection
3. `POSTMAN_ENVIRONMENT_UID`: The UID of your Postman environment

## Local Development

For local development, you can use the same commands:

```bash
# Generate collection for local development
./postman-gen \
  -dir . \
  -name "My API (Local)" \
  -base-url "http://localhost:8080" \
  -group-depth 2 \
  -group-by-method \
  -tag-folders \
  -out local-collection.json \
  -env-out local-environment.json \
  -env-name "Local Development"
```

## Tips

1. **Annotation Best Practices**: Use comprehensive annotations in your Go code for better documentation
2. **Branch-specific Collections**: Generate different collections for different environments
3. **Version Tagging**: Include version information in collection names
4. **Automation**: Set up webhooks to trigger collection updates when API changes
5. **Team Collaboration**: Share generated collections with your team through Postman workspaces
