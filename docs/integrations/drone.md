# drone

`drone` reads `DRONE_SERVER` and `DRONE_TOKEN`. It has no JSON flag; `--format '{{ jsonify . }}'` prints JSON through
a Go template. `drone build ls` filters its page by branch, event, and status on the client. Drone 2 releases come
from the `drone` branch of `harness/harness`, whose default branch is Harness Open Source.

This page says what `drone` reads and writes where it meets kit's scope, for each service it reaches, and the
machine-readable descriptions each publishes. The index of these pages is [1st.md](1st.md).

| Service | Category | kit status |
| --- | --- | --- |
| [Drone](#drone) | [CI](ci.md) | `none` |

## Drone

### Read from Drone

| Capability | Command |
| --- | --- |
| Recent runs | `drone build ls "${repo}" --event push --branch "${branch}" --limit 20 --format '{{ jsonify . }}'` |
| A run's stages and steps | `drone build info "${repo}" "${number}" --format '{{ jsonify . }}'`: `stages[].steps[]` with `exit_code` |
| A step's log | `drone log view "${repo}" "${number}" "${stage}" "${step}"` |
| Artifacts | none |

### Write to Drone

| Capability | Command |
| --- | --- |
| Rerun | `drone build restart "${repo}" "${number}"` |
| Cancel | `drone build stop "${repo}" "${number}"` |
| Start a run | `drone build create "${repo}" --branch "${branch}"` |
| Report a status | Drone posts it; see [who reports to the code forge](ci.md#who-reports-to-the-code-forge) |

### Drone schemas

- [Drone pipeline JSON Schema](https://www.schemastore.org/drone.json), SchemaStore's own "drone.json"; Drone
  publishes none.
