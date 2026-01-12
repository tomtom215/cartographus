# Secrets Management Guide

**Last Verified**: 2026-01-11

This guide covers best practices for managing sensitive configuration values in Cartographus deployments.

## Overview

Cartographus requires several sensitive configuration values:

| Secret | Purpose | Requirements |
|--------|---------|--------------|
| `JWT_SECRET` | Signs JWT authentication tokens | 32+ characters, cryptographically random |
| `ADMIN_PASSWORD` | Admin account password | 12+ characters, NIST SP 800-63B compliant |
| `PLEX_TOKEN` | Plex API authentication | 20+ characters (from Plex) |
| `TAUTULLI_API_KEY` | Tautulli API authentication | From Tautulli settings |
| `OIDC_CLIENT_SECRET` | OIDC provider authentication | From your identity provider |
| Database credentials | If using external database | Strong, unique credentials |

## Generating Secure Secrets

### JWT_SECRET

Generate a cryptographically secure random string:

```bash
# Option 1: OpenSSL (recommended)
openssl rand -base64 32

# Option 2: /dev/urandom
head -c 32 /dev/urandom | base64

# Option 3: Python
python3 -c "import secrets; print(secrets.token_urlsafe(32))"
```

**Never use**:
- Placeholder values (`REPLACE_WITH...`, `CHANGEME`, etc.)
- Predictable patterns
- Short strings (< 32 characters)
- Values from examples or documentation

### ADMIN_PASSWORD

Use a password manager to generate a strong password:
- Minimum 12 characters recommended
- Mix of uppercase, lowercase, numbers, symbols
- Unique to this deployment

---

## Environment-Specific Approaches

### Development (Local)

For local development, use a `.env` file:

```bash
# Copy the example and customize
cp .env.example .env

# Generate and set secrets
JWT_SECRET=$(openssl rand -base64 32)
sed -i "s/REPLACE_WITH_RANDOM_STRING_MIN_32_CHARS/$JWT_SECRET/" .env
```

**Important**: Never commit `.env` files to version control.

### Docker Compose

Use Docker secrets or environment files:

```yaml
# docker-compose.yml
services:
  cartographus:
    image: ghcr.io/tomtom215/cartographus:latest
    env_file:
      - .env.production  # Not committed to git
    # Or use Docker secrets:
    secrets:
      - jwt_secret
      - admin_password

secrets:
  jwt_secret:
    file: ./secrets/jwt_secret.txt
  admin_password:
    file: ./secrets/admin_password.txt
```

### Kubernetes

Use Kubernetes Secrets (base64 encoded):

```yaml
# secrets.yaml
apiVersion: v1
kind: Secret
metadata:
  name: cartographus-secrets
  namespace: cartographus
type: Opaque
stringData:
  JWT_SECRET: "your-generated-secret-here"
  ADMIN_PASSWORD: "your-admin-password"
  PLEX_TOKEN: "your-plex-token"
```

Apply with:
```bash
kubectl apply -f secrets.yaml
```

**Better: Use external secret managers**:
- [External Secrets Operator](https://external-secrets.io/)
- [Sealed Secrets](https://sealed-secrets.netlify.app/)
- [Vault Secrets Operator](https://developer.hashicorp.com/vault/docs/platform/k8s/vso)

### HashiCorp Vault

For enterprise deployments, use Vault:

```bash
# Store secrets in Vault
vault kv put secret/cartographus \
  jwt_secret="$(openssl rand -base64 32)" \
  admin_password="your-secure-password" \
  plex_token="your-plex-token"

# Retrieve at runtime
export JWT_SECRET=$(vault kv get -field=jwt_secret secret/cartographus)
```

### AWS Secrets Manager

```bash
# Create secret
aws secretsmanager create-secret \
  --name cartographus/production \
  --secret-string '{"JWT_SECRET":"...", "ADMIN_PASSWORD":"..."}'

# Retrieve in application
aws secretsmanager get-secret-value --secret-id cartographus/production
```

---

## Secret Rotation

### JWT_SECRET Rotation

Rotating `JWT_SECRET` will invalidate all existing sessions:

1. Generate new secret
2. Update configuration
3. Restart application
4. All users must re-authenticate

**Graceful rotation** (for zero-downtime):
1. Configure application to accept both old and new secrets temporarily
2. Deploy with new secret
3. Wait for old tokens to expire (default: 24 hours)
4. Remove old secret from configuration

### Password Rotation

1. Log in as admin
2. Change password via UI or API
3. Update stored password in secret manager
4. Test new password

---

## Security Best Practices

### DO:

- Use a secret manager (Vault, AWS Secrets Manager, etc.)
- Rotate secrets regularly (quarterly minimum)
- Use different secrets per environment
- Audit secret access logs
- Enable secret versioning
- Use short-lived tokens where possible

### DON'T:

- Commit secrets to version control
- Share secrets via email, Slack, etc.
- Use the same secret across environments
- Log secrets or include in error messages
- Store secrets in plaintext files on servers
- Use placeholder or example values

---

## Validation

Cartographus validates secrets at startup:

```
# Application refuses to start with placeholder values:
Error: JWT_SECRET contains a placeholder value - generate a secure secret with: openssl rand -base64 32

# Application refuses to start with weak secrets:
Error: JWT_SECRET must be at least 32 characters for security
Error: ADMIN_PASSWORD must be at least 12 characters (NIST SP 800-63B)
```

### Testing Your Configuration

```bash
# Verify secrets are properly set (without revealing values)
./cartographus --validate-config

# Check for placeholder patterns
grep -E "(REPLACE|CHANGEME|TODO|FIXME)" .env && echo "WARNING: Placeholder values detected!"
```

---

## Incident Response

If secrets are compromised:

1. **Immediately rotate** the compromised secret
2. **Revoke** any tokens signed with the old secret
3. **Audit** access logs for unauthorized access
4. **Investigate** how the secret was exposed
5. **Document** the incident and remediation steps

### JWT_SECRET Compromise

```bash
# 1. Generate new secret immediately
NEW_SECRET=$(openssl rand -base64 32)

# 2. Update configuration and restart
# This invalidates ALL existing sessions

# 3. Force all users to re-authenticate
# (automatic when JWT_SECRET changes)

# 4. Review audit logs for suspicious activity
```

---

## Checklist

Before deploying to production:

- [ ] All secrets generated with cryptographic randomness
- [ ] No placeholder values in configuration
- [ ] Secrets stored in appropriate secret manager
- [ ] `.env` files not committed to git
- [ ] Secret rotation schedule documented
- [ ] Incident response plan in place
- [ ] Access to secrets restricted to necessary personnel
- [ ] Secret access logging enabled

---

## Related Documentation

- [SECURITY.md](../SECURITY.md) - Security policy and configuration
- [Configuration Reference](./API-REFERENCE.md#configuration) - All configuration options
- [Docker Deployment](../README.md#docker) - Docker deployment guide
