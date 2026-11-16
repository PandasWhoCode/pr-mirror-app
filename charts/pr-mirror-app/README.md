<!--
  ~ SPDX-License-Identifier: Apache-2.0
-->

# pr-mirror-app

Helm chart for the [pr-mirror-app](https://github.com/PandasWhoCode/pr-mirror-app) HTTP daemon — Echo v5, structured logging, TLS, OpenAPI.

## Install

From the GHCR OCI registry (recommended):

```sh
helm install my-app oci://ghcr.io/pandaswhocode/charts/pr-mirror-app --version <X.Y.Z>
```

From a release `.tgz` asset:

```sh
gh release download <vX.Y.Z> --repo PandasWhoCode/pr-mirror-app --pattern 'pr-mirror-app-*.tgz'
helm install my-app ./pr-mirror-app-<X.Y.Z>.tgz
```

## Verify supply-chain attestations

Every chart artifact carries a GitHub-signed Sigstore attestation. Verify before installing:

```sh
gh attestation verify oci://ghcr.io/pandaswhocode/charts/pr-mirror-app:<X.Y.Z> --owner PandasWhoCode
gh attestation verify ./pr-mirror-app-<X.Y.Z>.tgz --owner PandasWhoCode
```

The chart `.tgz`'s SHA256 is GPG-signed: `gpg --verify pr-mirror-app-<X.Y.Z>.tgz.sha256.asc pr-mirror-app-<X.Y.Z>.tgz.sha256 && shasum -a 256 -c pr-mirror-app-<X.Y.Z>.tgz.sha256`.

## Configuration

The daemon is fully env-driven. The chart exposes three paths for supplying configuration:

| Path                | When to use                                                                |
| ------------------- | -------------------------------------------------------------------------- |
| `.Values.config`    | Plain key/value pairs, rendered into a chart-managed ConfigMap             |
| `.Values.envFrom`   | Reference user-supplied Secrets / ConfigMaps that the chart does not own   |
| `.Values.env`       | Inline `name`/`value` or `valueFrom` entries for one-off overrides         |
| `.Values.secret`    | Chart-managed Secret (escape hatch — prefer `envFrom` for real deployments) |

### TLS

Set `tls.enabled=true` and `tls.existingSecret` to the name of an existing Kubernetes TLS Secret (with `tls.crt` and `tls.key`). The chart mounts the Secret at `tls.mountPath` and points the daemon at it via `PRMIRROR_SERVER_HTTPS_CERTIFICATE` and `PRMIRROR_SERVER_HTTPS_KEY`.

### Seccomp

`podSecurityContext.seccompProfile` and `securityContext.seccompProfile` accept the three Kubernetes profile types:

```yaml
securityContext:
  seccompProfile:
    type: RuntimeDefault              # default
    # type: Localhost
    # localhostProfile: profiles/audit.json
    # type: Unconfined
```

`Localhost` requires the profile JSON to already exist at `/var/lib/kubelet/seccomp/<localhostProfile>` on every node — the chart cannot ship it. Misconfigurations (invalid `type`, missing `localhostProfile`) fail at `helm install` time via a template `fail`.

### Optional observability resources

All default-off; flip to `enabled: true` only when the cluster has the required CRDs.

| Value                            | CRD                                          |
| -------------------------------- | -------------------------------------------- |
| `metrics.serviceMonitor.enabled` | `monitoring.coreos.com/v1` (Prometheus Op)   |
| `metrics.podMonitor.enabled`     | `monitoring.coreos.com/v1` (Prometheus Op)   |
| `logging.podLogs.enabled`        | `monitoring.grafana.com/v1alpha2` (Grafana Alloy) |

The starter binary does not currently expose `/metrics`. Enable the monitor resources only after wiring metrics into your downstream fork.

## Health endpoints

| Path              | Probe                          |
| ----------------- | ------------------------------ |
| `/api/v1/livez`   | Liveness; always 200 once up   |
| `/api/v1/readyz`  | Readiness; aggregates checks   |
| `/api/v1/healthz` | Alias of `/api/v1/readyz`      |

## Values

See [values.yaml](./values.yaml) for the full schema; key defaults:

- `image.repository`: `ghcr.io/pandaswhocode/pr-mirror-app`
- `service.port`: `8080` (HTTP), `service.httpsPort`: `8443`
- `replicaCount`: `1`
- `terminationGracePeriodSeconds`: `30`
- securityContext: non-root (UID 10001), read-only root FS, all capabilities dropped, `RuntimeDefault` seccomp
