# printcost Helm chart

Deploys the PrintCost single-binary app.

## ⚠️ SQLite is single-writer

`replicaCount` must stay at `1`. PrintCost uses a single SQLite file on a
`ReadWriteOnce` PVC; running more than one replica against the same volume
will cause database lock errors and possible corruption. The Deployment uses
`strategy: Recreate` so a rolling update never briefly runs two pods against
the same volume.

## Install

```
helm install my-printcost ./helm/printcost \
  --set image.repository=your-registry/printcost \
  --set image.tag=1.0.0
```

## Values

See `values.yaml` for `image`, `service`, `ingress`, `persistence`, `env`,
`resources`, and `replicaCount`.
