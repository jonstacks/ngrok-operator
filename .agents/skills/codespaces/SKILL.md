---
name: codespaces
description: >
  Creates, configures, and runs chainsaw e2e tests in a GitHub Codespace for
  ngrok-operator. Covers codespace lifecycle (create, wait, delete), secrets
  access via login shell, nix devshell, correct deploy target, and chainsaw
  test execution. Use when the user mentions codespaces, e2e tests in a
  codespace, verifying a branch in a codespace, or spinning up a test
  environment.
---

# Codespaces Skill — ngrok-operator

This skill covers the full lifecycle of a GitHub Codespace for the
ngrok-operator: creating one, verifying the environment, deploying the
operator, running chainsaw e2e tests, and tearing it down.

## Creating a Codespace

```bash
gh codespace create --repo ngrok/ngrok-operator --branch <branch> --machine premiumLinux
```

Then wait for it to become `Available`:

```bash
while true; do
  state=$(gh codespace list --json name,state | jq -r '.[] | select(.name == "<name>") | .state')
  echo "state: $state"
  [ "$state" = "Available" ] && break
  sleep 10
done
```

**Machine size**: always use `premiumLinux` (8 cores, 32 GB RAM, 64 GB storage).
This is also enforced via `hostRequirements` in `.devcontainer/devcontainer.json`.

### What happens on first boot

The devcontainer (`/.devcontainer/devcontainer.json`) runs `post-create.sh` which:
1. Starts the nix-daemon
2. Runs `direnv allow`
3. Runs `make kind-create` — the `ngrok-operator` kind cluster is created automatically

You do **not** need to create the kind cluster manually.

## Required Codespace Secrets

Three user-level secrets must be configured in GitHub (Settings → Codespaces → Secrets)
and scoped to `ngrok/ngrok-operator`:

| Secret | Purpose |
|--------|---------|
| `NGROK_API_KEY` | ngrok API key for operator credentials |
| `NGROK_AUTHTOKEN` | ngrok authtoken for operator credentials |
| `NGROK_EMAIL` | present but not used by the deploy flow |

## Critical: Always Use a Login Shell

Codespace secrets (`NGROK_API_KEY`, `NGROK_AUTHTOKEN`) and `CODESPACE_NAME` are
only available in a **login shell**. A plain `gh codespace ssh -- 'command'` does
**not** start a login shell, so those variables will be empty.

**Always use `bash -l`:**

```bash
gh codespace ssh --codespace <name> -- 'bash -l -c "<your command>"'
```

Verify secrets are present before proceeding:

```bash
gh codespace ssh --codespace <name> -- 'bash -l -c "
  echo NGROK_API_KEY present: $([ -n \"$NGROK_API_KEY\" ] && echo YES || echo NO)
  echo NGROK_AUTHTOKEN present: $([ -n \"$NGROK_AUTHTOKEN\" ] && echo YES || echo NO)
  echo CODESPACE_NAME: $CODESPACE_NAME
"'
```

## Nix Devshell

All dev tools (`kind`, `kubectl`, `helm`, `chainsaw`) live in the Nix devshell
and are **not** on `$PATH` in SSH sessions. Always run make targets through:

```bash
/nix/var/nix/profiles/default/bin/nix develop --command make <target>
```

The nix-daemon may not be running in fresh SSH sessions. Check and start it before
running any nix commands:

```bash
gh codespace ssh --codespace <name> -- 'bash -l -c "
  if ! pidof nix-daemon > /dev/null 2>&1; then
    sudo -n sh -c \". /nix/var/nix/profiles/default/etc/profile.d/nix-daemon.sh; /nix/var/nix/profiles/default/bin/nix-daemon > /tmp/nix-daemon.log 2>&1\" &
    for i in \$(seq 1 30); do
      /nix/var/nix/profiles/default/bin/nix --version > /dev/null 2>&1 && echo nix-daemon ready && break || sleep 1
    done
  else
    echo nix-daemon already running
  fi
"'
```

## Deploying the Operator

**Always use `make deploy_for_e2e`**, not `make deploy`, before running chainsaw
tests. Using `make deploy` will cause the `operator-registration` test to fail
because it does not enable bindings.

Key differences in `deploy_for_e2e`:
- `bindings.enabled=true` — required for the `operator-registration` chainsaw test
- `image.pullPolicy=Never` — uses the locally loaded kind image
- `oneClickDemoMode=false`
- e2e pod annotations (`k8s.ngrok.com/test: {env: e2e}`)
- `log.level=debug`

```bash
gh codespace ssh --codespace <name> -- 'bash -l -c "
  cd /workspaces/ngrok-operator
  /nix/var/nix/profiles/default/bin/nix develop --command make deploy_for_e2e
"'
```

After deploying, wait for all 3 pods to be ready (`manager`, `agent`, `bindings-forwarder`):

```bash
gh codespace ssh --codespace <name> -- 'bash -l -c "
  cd /workspaces/ngrok-operator
  /nix/var/nix/profiles/default/bin/nix develop --command \
    kubectl wait --for=condition=ready pod \
      -l app.kubernetes.io/name=ngrok-operator \
      -n ngrok-operator \
      --timeout=120s
"'
```

## CODESPACE_NAME and Domain Suffix

When `CODESPACE_NAME` is set (which it will be in a login shell), the chainsaw
tests automatically derive a unique domain suffix. Defined in `tools/make/test.mk`:

```makefile
ifneq ($(CODESPACE_NAME),)
CHAINSAW_NGROK_DOMAIN_SUFFIX ?= -$(CODESPACE_NAME).ngrok.app
CHAINSAW_NGROK_DOMAIN_SUFFIX_DASHES ?= -$(CODESPACE_NAME)-ngrok-app
```

This namespaces domain resources per codespace (e.g.
`dom-labels-<codespace-name>-ngrok-app`), preventing collisions
across concurrent runs. No extra configuration needed — just ensure you're using
a login shell.

## Running Chainsaw Tests

```bash
gh codespace ssh --codespace <name> -- 'bash -l -c "
  cd /workspaces/ngrok-operator
  /nix/var/nix/profiles/default/bin/nix develop --command make e2e-tests
"'
```

Expected output when all tests pass:

```
Tests Summary...
- Passed  tests 6
- Failed  tests 0
- Skipped tests 0
```

The 6 tests: `sanity-checks`, `domain-controller-labels`, `finalizers`,
`operator-registration`, `loadbalancer-services`, `internal-domains`.

## Deleting the Codespace

```bash
gh codespace delete --codespace <name>
```

Verify it's gone:
```bash
gh codespace list --json name | jq '.[] | select(.name == "<name>")'
# should return nothing
```

## Complete End-to-End Script

```bash
CODESPACE=<name>

# 1. Verify secrets are present (login shell required)
gh codespace ssh --codespace $CODESPACE -- 'bash -l -c "
  echo NGROK_API_KEY: $([ -n \"$NGROK_API_KEY\" ] && echo YES || echo NO)
  echo NGROK_AUTHTOKEN: $([ -n \"$NGROK_AUTHTOKEN\" ] && echo YES || echo NO)
  echo CODESPACE_NAME: $CODESPACE_NAME
"'

# 2. Ensure nix-daemon is running
gh codespace ssh --codespace $CODESPACE -- 'bash -l -c "
  pidof nix-daemon > /dev/null && echo already running || (
    sudo -n sh -c \". /nix/var/nix/profiles/default/etc/profile.d/nix-daemon.sh; /nix/var/nix/profiles/default/bin/nix-daemon > /tmp/nix-daemon.log 2>&1\" &
    for i in \$(seq 1 30); do
      /nix/var/nix/profiles/default/bin/nix --version > /dev/null 2>&1 && echo ready && break || sleep 1
    done
  )
"'

# 3. Verify kind cluster exists
gh codespace ssh --codespace $CODESPACE -- 'bash -l -c "
  cd /workspaces/ngrok-operator
  /nix/var/nix/profiles/default/bin/nix develop --command kind get clusters
"'

# 4. Deploy for e2e
gh codespace ssh --codespace $CODESPACE -- 'bash -l -c "
  cd /workspaces/ngrok-operator
  /nix/var/nix/profiles/default/bin/nix develop --command make deploy_for_e2e
"'

# 5. Wait for pods
gh codespace ssh --codespace $CODESPACE -- 'bash -l -c "
  cd /workspaces/ngrok-operator
  /nix/var/nix/profiles/default/bin/nix develop --command \
    kubectl wait --for=condition=ready pod \
      -l app.kubernetes.io/name=ngrok-operator \
      -n ngrok-operator \
      --timeout=120s
"'

# 6. Run chainsaw tests
gh codespace ssh --codespace $CODESPACE -- 'bash -l -c "
  cd /workspaces/ngrok-operator
  /nix/var/nix/profiles/default/bin/nix develop --command make e2e-tests
"'

# 7. Delete codespace when done
gh codespace delete --codespace $CODESPACE
```
