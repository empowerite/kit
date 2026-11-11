# harness

`harness` is Harness's own CLI ([harness/cli](https://github.com/harness/cli), version 3); an older one,
harness/harness-cli, still exists and takes YAML files. It authenticates with `harness auth login`, or
`HARNESS_API_KEY`, and `harness auth setscope` sets the organization and project. `--format json` prints JSON;
`--limit` and `--all` set how much a list reads. Its commands are a verb and a resource, such as
`harness list environment`.

This page says what `harness` reads and writes where it meets kit's scope, for each service it reaches, and the
machine-readable descriptions each publishes. The index of these pages is [1st.md](1st.md).

| Service | Category | kit status |
| --- | --- | --- |
| [Harness CD](#harness-cd) | [CD](cd.md) | none |

## Harness CD

A Harness pipeline deploys a service to an environment's infrastructure; an execution is a deployment, and an approval
step makes an approval instance. Harness also runs a managed Argo CD for GitOps.

### Read from Harness CD

| Capability | Command |
| --- | --- |
| List environments | `harness list environment --format json --all` |
| Deployments (executions) | `harness list execution "${pipeline_id}" --format json --limit "${n}"` |
| A deployment's state | `harness get execution "${execution_id}" --format json` |
| Pending approvals | `harness list approval_instance "${execution_id}"` |
| GitOps applications | `harness list gitops_application` |

### Write to Harness CD

| Capability | Command |
| --- | --- |
| Deploy or promote | `harness execute pipeline "${pipeline_id}" --input environment="${environment}"`; `--follow` |
| Roll back | `harness execute execution:abort "${execution_id}" --interrupt-type StageRollback` |
| Retry | `harness execute execution:retry "${pipeline_id}/${execution_id}" --only-failed-stages` |
| Approve or reject | `harness execute approval_instance:approve "${approval_instance_id}" --comment "${comment}"`, `approval_instance:reject` |
| Sync a GitOps application | `harness execute gitops_application:sync "${app}"` |

Gaps: no command says which version of a service runs in each environment; the Harness API has instance routes, not
checked here. These commands are from Harness's documentation; `harness` was not run.

### Harness CD schemas

- Harness publishes its API at [the Harness API reference](https://apidocs.harness.io/), which offers an OpenAPI
  download; the download answered 500 when tried, so it is (not verified).
- No JSON Schema for pipeline YAML was found on SchemaStore.
