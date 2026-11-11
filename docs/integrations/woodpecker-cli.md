# woodpecker-cli

`woodpecker-cli` ships with Woodpecker. It reads `WOODPECKER_SERVER` and `WOODPECKER_TOKEN`. It prints no JSON:
`ls`, `show`, and `last` take `--output table` or `--output go-template=`, and `ps` takes `--format`. Lists page up to
`--limit`. A run is a pipeline of workflows and steps; the server filters runs by `event`, `branch`, and `status`.

This page says what `woodpecker-cli` reads and writes where it meets kit's scope, for each service it reaches, and the
machine-readable descriptions each publishes. The index of these pages is [1st.md](1st.md).

| Service | Category | kit status |
| --- | --- | --- |
| [Woodpecker CI](#woodpecker-ci) | [CI](ci.md) | `none` |

## Woodpecker CI

### Read from Woodpecker CI

| Capability | Command |
| --- | --- |
| Recent runs | `woodpecker-cli pipeline ls "${repo}" --event push --branch "${branch}" --limit 20` |
| A run's steps and failed step | `woodpecker-cli pipeline ps "${repo}" "${number}"` |
| A step's log | `woodpecker-cli pipeline log show "${repo}" "${number}" "${step}"` |
| Artifacts | none |

### Write to Woodpecker CI

| Capability | Command |
| --- | --- |
| Rerun | `woodpecker-cli pipeline start "${repo}" "${number}"` |
| Cancel | `woodpecker-cli pipeline stop "${repo}" "${number}"`; `pipeline kill` forces it |
| Start a run | `woodpecker-cli pipeline create "${repo}" --branch "${branch}"` |
| Report a status | Woodpecker posts it; see [who reports to the code forge](ci.md#who-reports-to-the-code-forge) |

### Woodpecker CI schemas

- [Woodpecker pipeline JSON Schema](https://raw.githubusercontent.com/woodpecker-ci/woodpecker/main/pipeline/frontend/yaml/linter/schema/schema.json),
  SchemaStore's "Woodpecker pipeline config" entry
- The server generates its Swagger 2 document and serves it at `/swagger/doc.json`, as
  [Woodpecker's own instance](https://ci.woodpecker-ci.org/swagger/doc.json) does; the repository keeps no static
  copy.
