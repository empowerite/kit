# az

`az` is Microsoft's Azure CLI, and its `azure-devops` extension, `az extension add --name azure-devops`, adds
`az repos`. It authenticates with `az login`, or a personal access token in `AZURE_DEVOPS_EXT_PAT`, and takes the
organization and project from `az devops configure --defaults` or from `--org` and `--project`. `--output json` is
the default output, and `--query` filters it with JMESPath. A list takes `--top` and `--skip`. `az devops invoke
--area "${area}" --resource "${resource}" --route-parameters …` calls any REST resource by its area, and `az rest`
any URL. kit's `ado` tracker already uses `az boards`; see [Azure Boards](#azure-boards).

This page says what `az` reads and writes where it meets kit's scope, for each service it reaches, and the
machine-readable descriptions each publishes. The index of these pages is [1st.md](1st.md).

| Service | Category | kit status |
| --- | --- | --- |
| [Azure DevOps Repos](#azure-devops-repos) | [code forge](forges.md) | `provider` |
| [Azure Boards](#azure-boards) | [tracker](trackers.md) | `provider` (`ado`) |
| [Azure Pipelines runs](#azure-pipelines-runs) | [CI](ci.md) | `planned`, with an `ado` code forge |
| [Azure Pipelines environments and releases](#azure-pipelines-environments-and-releases) | [CD](cd.md) | none; an `ado` code forge is planned |

## Azure DevOps Repos

kit's provider, `forge/ado.sh`, reads the organization, the project, and the repository from the primary remote's
URL. It implements contract 3 and declares that Azure DevOps lacks rate limits it shows, cheap marks, releases, and
branches linked to a work item: `kit work` pushes the branch it names, and the pull request it opens links the work
item. Azure Boards is the forge's own tracker, as `#7` and `AB#7` alike, and `forge_settings.ado` in `kit.yml` names
the work item type and the closing states of the project's process.

Azure DevOps Server, the self-hosted product, answers at `https://${host}/${collection}` with the same REST API at an
older version. `--org` names the collection URL; which Server releases the extension supports was not checked.

### Read from Azure DevOps Repos

| Capability | Command |
| --- | --- |
| Signed in | `az account show --query user.name --output tsv`, as the `ado` tracker does |
| Owner and name | `az repos show --repository "${repo}" --query '[project.name, name]'` |
| Browser URL | `az repos show --repository "${repo}" --query webUrl`; a pull request's is `${webUrl}/pullrequest/${n}` |
| Caller's login | `az account show --query user.name` |
| Rate limits | no call; Azure DevOps sends `X-RateLimit-Resource`, `X-RateLimit-Delay`, `X-RateLimit-Limit`, `X-RateLimit-Remaining`, and `X-RateLimit-Reset` only on a request it delays, which `az` does not show |
| Marks | none cheap found; `az repos pr list --status all --top 1` gives one pull request, in an order not verified, and misses an update to an older one; no ETag was checked |
| Branch for an issue | `az boards work-item show --id "${n}" --expand relations` and its `ArtifactLink` relations named `Branch` or `Pull Request` |
| Title, body | `az repos pr show --id "${n}" --query '[title, description]'` |
| Author | `az repos pr show --id "${n}" --query createdBy.uniqueName` |
| State and merge commit | `az repos pr show --id "${n}" --query '[status, lastMergeCommit.commitId]'`: `active`, `abandoned`, or `completed` |
| Change on a branch | `az repos pr list --source-branch "${branch}" --status all` |
| Closing issues | `az repos pr work-item list --id "${n}"`: the linked work items, which completion can transition |
| Review threads | `az devops invoke --area git --resource pullRequestThreads --route-parameters project="${project}" repositoryId="${repo}" pullRequestId="${n}"` (raw API); each thread's `status` is `active`, `fixed`, `closed`, or another |
| Checks rollup | `az repos pr policy list --id "${n}"`: the policy evaluations, build validation among them; other statuses through `--resource pullRequestStatuses` (raw API) |
| Open changes | `az repos pr list --status active --top "${limit}"` |
| Reviewers | each pull request's `reviewers` in `az repos pr list`, each `vote` 10 or 5 approved, -5 or -10 changes, 0 asked; `az repos pr reviewer list --id "${n}"` lists them alone; a vote holds no time; `kit changes reviews` draws them |
| Merged changes | `az repos pr list --status completed --top 51`: the 50 Azure DevOps lists first, newest first in its reference's examples, though no parameter sets the order, each with its `closedDate` and `closedBy`, and a 51st that says there are more; `kit changes merged` draws them |

### Write to Azure DevOps Repos

| Capability | Command |
| --- | --- |
| Branch linked to an issue | `az repos ref create --name "refs/heads/${branch}" --object-id "${sha}"`, then `az boards work-item relation add --id "${n}" --relation-type Branch --target-url "vstfs:///Git/Ref/${project_id}%2F${repo_id}%2FGB${branch}"` (the link not verified) |
| Open a draft | `az repos pr create --draft true --source-branch "${branch}" --target-branch "${default}" --title "${title}" --work-items "${n}" --output json` |
| Mark ready | `az repos pr update --id "${n}" --draft false` |
| Set the title | `az repos pr update --id "${n}" --title "${title}"` |
| Set the body | `az repos pr update --id "${n}" --description "${body}"`; each value is a line |

Gaps: review threads and their state have no `az repos` command. Azure DevOps closes no work item from a pull
request's body by itself; work items link with `--work-items` and move on completion with `--transition-work-items`.
The `az devops invoke` area and resource names here were not tried against a live organization.

### Azure DevOps Repos schemas

- Microsoft publishes no OpenAPI document for the Azure DevOps REST API. The Swagger documents per area were in
  MicrosoftDocs/vsts-rest-api-specs, which answers 404 on 2026-11-10, and no other copy was found.
- The REST reference is [Azure DevOps Git REST API](https://learn.microsoft.com/en-us/rest/api/azure/devops/git/),
  a page, not a document; the rate-limit headers are on
  [Azure DevOps rate limits](https://learn.microsoft.com/en-us/azure/devops/integrate/concepts/rate-limits).
- No JSON Schema for service hook payloads or for `az` output is published. The pipeline file's schema is under
  [Azure Pipelines runs schemas](#azure-pipelines-runs-schemas).

## Azure Boards

`az` is Microsoft's own CLI; its `azure-devops` extension adds `az boards`, `az repos`, and `az pipelines`. `az login`
signs in with Microsoft Entra ID; `az devops login` or `AZURE_DEVOPS_EXT_PAT` takes a personal access token instead.
`--output json` prints JSON and `--query` filters it with JMESPath. `az devops invoke --area --resource
--route-parameters` reaches any REST route by its area, and `az rest` reaches any URL. `az boards query` runs a flat
Work Item Query Language (WIQL) query and answers every item it matches. kit's `tracker/ado.sh` uses these commands.

### Read from Azure Boards

| Capability | Command |
| --- | --- |
| Title | `az boards work-item show --id "${n}" --org "${org}" --output json`, `.fields["System.Title"]` |
| State | the same, `.fields["System.State"]`; kit reads Closed, Done, and Removed as closed |
| Assignees | the same, `.fields["System.AssignedTo"].uniqueName` |
| Caller | `az account show --query user.name --output tsv` |
| Issue page | none; kit builds `${org}/${project}/_workitems/edit/${n}` |
| New-issue page | none; kit builds `${org}/${project}/_workitems/create/${type}?[System.Title]=${title}` |
| Open issues | `az boards query --org "${org}" --project "${project}" --wiql "${wiql}"`, selecting `System.Title`, `System.AssignedTo`, `System.Tags`, and `System.CreatedDate` where `System.State` is not closed |
| Parent, children, and linked pull requests | `az boards work-item show --id "${n}" --expand relations --org "${org}"`, or `az boards work-item relation show --id "${n}"` |
| Iterations | `az boards iteration team list --team "${team}" --project "${project}" --org "${org}" --output json`, as `kit issues iterations` reads them, the team `forge_settings.ado.team` names, else the project's default team, with no `--team` |
| An iteration's work items | `az boards iteration team list-work-items --id "${iteration_id}" --team "${team}" --project "${project}"` |
| Board status | `az boards work-item show --id "${n}"`, `.fields["System.BoardColumn"]` |

### Write to Azure Boards

| Capability | Command |
| --- | --- |
| Assign | `az boards work-item update --id "${n}" --assigned-to "${login}" --org "${org}"` |
| Unassign | `az boards work-item update --id "${n}" --assigned-to "" --org "${org}"` |
| Close as completed | `az boards work-item update --id "${n}" --state Closed --org "${org}"` (`closed_state`) |
| Close as not planned | `az boards work-item update --id "${n}" --state Removed --org "${org}"` (`removed_state`); `--reason` sets the process's reason |
| Close as a duplicate | the comment `Duplicate of AB#${m}`, then the not-planned close; `az boards work-item relation add --relation-type "${type}"` can add a duplicate link, its name from `relation list-type` |
| Comment | `az boards work-item update --id "${n}" --discussion "${text}" --org "${org}"` |
| Create | `az boards work-item create --title "${title}" --type "${type}" --description "${body}" --project "${project}" --org "${org}" --output json` |
| Put under a parent | `az boards work-item relation add --id "${m}" --relation-type parent --target-id "${n}" --org "${org}"`; kit writes a task line in the parent's body instead |
| Link a change | `AB#${n}` in a GitHub pull request's title, description, or commits; `az repos pr work-item add --id "${pr}" --work-items "${n}"` for an Azure Repos pull request |
| Set the iteration | `az boards work-item update --id "${n}" --iteration "${iteration_path}"` |

Gaps: a work item has one assignee, and no milestone object; an iteration path or a tag stands in for one. A process
template defines the states, so the closed states are settings (`closed_state`, `removed_state`). The
[Azure Boards app for GitHub](https://learn.microsoft.com/en-us/azure/devops/boards/github/link-to-from-github) links
`AB#` references, and `Fixes AB#${n}` moves the work item to its done state when the pull request merges.

### Azure Boards schemas

- Microsoft publishes no OpenAPI document for the Azure DevOps REST API. The Swagger documents per area were in
  MicrosoftDocs/vsts-rest-api-specs, which answers 404 on 2026-11-10, and no other copy was found.
  [The work item tracking REST reference](https://learn.microsoft.com/en-us/rest/api/azure/devops/wit/) and the
  `work` area's, for iterations and boards, describe the routes in prose.
- No JSON Schema for work item types or service hook payloads. The SchemaStore entries for Azure DevOps cover
  `azure-pipelines.yml` and extension manifests; see [Azure Pipelines runs schemas](#azure-pipelines-runs-schemas).

## Azure Pipelines runs

The `azure-devops` extension to `az` covers Azure Pipelines; it installs itself on first use. It signs in with `az
login` or `AZURE_DEVOPS_EXT_PAT`, as the `ado` tracker does, and takes `--organization` and `--project` or their
defaults from `az devops configure`. Output is JSON by default; `--query` filters it with JMESPath, and `--top` limits
a list. `az devops invoke` calls any REST route with the extension's sign-in.

### Read from Azure Pipelines runs

| Capability | Command |
| --- | --- |
| Recent runs | `az pipelines runs list --branch "${branch}" --top 20 --query-order QueueTimeDesc --reason individualCI`; `--reason` takes one value per call, `schedule` or `manual` the others |
| A run's jobs and failed step | REST `GET https://dev.azure.com/${organization}/${project}/_apis/build/builds/${build_id}/timeline?api-version=7.1`: records of type `Stage`, `Job`, and `Task`, each with `result` |
| A change's checks | `az pipelines runs list --reason pullRequest --branch "refs/pull/${n}/merge"` on GitHub; an Azure Repos pull request's policies belong to the code forge |
| A run's log | REST `GET https://dev.azure.com/${organization}/${project}/_apis/build/builds/${build_id}/logs/${log_id}?api-version=7.1`; the timeline names each record's `log.id` |
| Artifacts | `az pipelines runs artifact list --run-id "${run_id}"`; `az pipelines runs artifact download --run-id "${run_id}" --artifact-name "${name}" --path "${dir}"` |

### Write to Azure Pipelines runs

| Capability | Command |
| --- | --- |
| Rerun | REST `PATCH https://dev.azure.com/${organization}/${project}/_apis/build/builds/${build_id}?retry=true&api-version=7.1` |
| Cancel | `az pipelines build cancel --build-id "${build_id}"` |
| Start a run | `az pipelines run --name "${pipeline}" --branch "${branch}"`, with `--parameters` or `--variables` |
| Report a status | on Azure Repos, REST `POST https://dev.azure.com/${organization}/${project}/_apis/git/repositories/${repository}/commits/${sha}/statuses?api-version=7.1` |

The CLI has no command for a run's jobs, its log, or a retry. The REST routes above go through `az devops invoke
--area build --resource timeline --route-parameters project="${project}" buildId="${build_id}"`; the area and
resource names were not checked against a live organization.

### Azure Pipelines runs schemas

- [Azure Pipelines YAML JSON Schema](https://raw.githubusercontent.com/microsoft/azure-pipelines-vscode/main/service-schema.json),
  SchemaStore's "Azure Pipelines" entry
- Microsoft publishes no OpenAPI document for the Azure DevOps REST API. The Swagger documents per area were in
  MicrosoftDocs/vsts-rest-api-specs, which answers 404 on 2026-11-10, and no other copy was found.
  The REST reference on Microsoft Learn describes the routes in prose.

## Azure Pipelines environments and releases

`az` with the `azure-devops` extension is Microsoft's own CLI, the one kit's `ado` tracker uses. It authenticates
with `az login` or `AZURE_DEVOPS_EXT_PAT`. `--output json` prints JSON and `--query` filters it with JMESPath.
`az pipelines release` covers classic releases. YAML environments, their deployment records, and pipeline approvals
have no command, so they go through `az rest` with `--resource 499b84ac-1321-427f-aa17-267ca6975798`, the Azure
DevOps resource, or `az devops invoke`. A list that has more pages returns a continuation token.

Azure Pipelines has two models. A YAML pipeline's deployment job targets an environment, which records each
deployment and holds approvals and checks. A classic release pipeline makes a release, a numbered object unrelated
to a git tag, and deploys it through stages. Microsoft recommends YAML, and an organization setting can disable the
creation of classic release pipelines; no retirement date for classic releases was found.

### Read from Azure Pipelines environments and releases

| Capability | Command |
| --- | --- |
| List classic releases | `az pipelines release list --definition-id "${definition_id}" --top "${n}" --output json` |
| A classic release and its stages | `az pipelines release show --id "${release_id}" --output json` |
| List environments | `az rest --resource 499b84ac-1321-427f-aa17-267ca6975798 --url "https://dev.azure.com/${org}/${project}/_apis/distributedtask/environments?api-version=7.1"` (raw API) |
| Deployments in an environment | `az rest ... --url "https://dev.azure.com/${org}/${project}/_apis/distributedtask/environments/${environment_id}/environmentdeploymentrecords?api-version=7.1"` (raw API) |
| Classic deployments, what is live where | `az rest ... --url "https://vsrm.dev.azure.com/${org}/${project}/_apis/release/deployments?definitionEnvironmentId=${stage_id}&latestAttemptsOnly=true&api-version=7.1"` (raw API) |
| Pending approvals, YAML | `az rest ... --url "https://dev.azure.com/${org}/${project}/_apis/pipelines/approvals?state=pending&api-version=7.1"` (raw API) |
| Pending approvals, classic | `az rest ... --url "https://vsrm.dev.azure.com/${org}/${project}/_apis/release/approvals?api-version=7.1"` (raw API) |

### Write to Azure Pipelines environments and releases

| Capability | Command |
| --- | --- |
| Create a classic release | `az pipelines release create --definition-id "${definition_id}" --artifact-metadata-list "${alias}=${version_id}"` |
| Deploy or promote, YAML | `az pipelines run --name "${pipeline}" --branch "${ref}" --parameters "environment=${environment}"`, when the pipeline takes that parameter |
| Deploy or promote, classic | `az rest --method patch ... --url "https://vsrm.dev.azure.com/${org}/${project}/_apis/Release/releases/${release_id}/environments/${stage_id}?api-version=7.1" --body '{"status":"inProgress"}'` (raw API) |
| Roll back | the same deploy on an earlier release or commit |
| Approve, YAML | `az rest --method patch ... --url "https://dev.azure.com/${org}/${project}/_apis/pipelines/approvals?api-version=7.1" --body "[{\"approvalId\":\"${approval_id}\",\"status\":\"approved\"}]"` (raw API) |
| Approve, classic | `az rest --method patch ... --url "https://vsrm.dev.azure.com/${org}/${project}/_apis/release/approvals/${approval_id}?api-version=7.1" --body '{"status":"approved"}'` (raw API) |

Here `...` stands for `--resource 499b84ac-1321-427f-aa17-267ca6975798`. Gaps: no command reads environments,
deployment records, or approvals. Azure Pipelines has no built-in way to write a GitHub deployment; its GitHub app
reports checks.

### Azure Pipelines environments and releases schemas

- [Azure Pipelines YAML JSON Schema](https://raw.githubusercontent.com/microsoft/azure-pipelines-vscode/master/service-schema.json),
  SchemaStore's "Azure Pipelines" entry.
- Microsoft publishes no OpenAPI document for the Azure DevOps REST API. The Swagger documents per area were in
  MicrosoftDocs/vsts-rest-api-specs, which answers 404 on 2026-11-10, and no other copy was found. The
  [Azure DevOps REST API reference](https://learn.microsoft.com/en-us/rest/api/azure/devops/) is the readable source
  of the routes above.

## Schemas of its output

`az` publishes no schema of what its commands print, so each command kit runs whose JSON kit reads has one
here, in [az/](az/): a JSON Schema of the fields kit reads, typed from the provider's API description,
with an example the gate validates against it.

| Schema | The command as kit runs it |
| --- | --- |
| [boards-iteration-team-list.schema.json](az/boards-iteration-team-list.schema.json) | `az boards iteration team list [--team ${team}] --project ${project} --output json` |
| [boards-query.schema.json](az/boards-query.schema.json) | `az boards query --project ${project} --wiql "SELECT [System.Id], [System.Title], [System.AssignedTo], [System.Tags], [System.CreatedDate] FROM WorkItems WHERE [System.TeamProject] = @project AND [System.State] NOT IN ('Closed', 'Done', 'Removed') ORDER BY [System.CreatedDate] ASC" --output json` |
| [boards-work-item-create.schema.json](az/boards-work-item-create.schema.json) | `az boards work-item create --title ${title} --type ${type} --description ${body} --project ${project} --output json` |
| [boards-work-item-show.schema.json](az/boards-work-item-show.schema.json) | `az boards work-item show --id ${id} --output json` |
| [devops-invoke-pull-request-threads.schema.json](az/devops-invoke-pull-request-threads.schema.json) | `az devops invoke --area git --resource pullRequestThreads --route-parameters project=${project} repositoryId=${repository} pullRequestId=${id} --api-version 7.1 --output json` |
| [pipelines-runs-list.schema.json](az/pipelines-runs-list.schema.json) | `az pipelines runs list --project ${project} --top 50 --query-order QueueTimeDesc --output json` |
| [repos-pr-create.schema.json](az/repos-pr-create.schema.json) | `az repos pr create --draft ${draft} --source-branch ${branch} --target-branch ${base} --title ${title} --project ${project} --repository ${repository} [--work-items ${item}] --output json` |
| [repos-pr-list.schema.json](az/repos-pr-list.schema.json) | `az repos pr list --project ${project} --repository ${repository} --status ${status} --top ${top} [--skip ${skip}] [--source-branch ${branch}] --output json` |
| [repos-pr-policy-list.schema.json](az/repos-pr-policy-list.schema.json) | `az repos pr policy list --id ${id} --top ${forge_list_limit} --output json` |
| [repos-pr-show.schema.json](az/repos-pr-show.schema.json) | `az repos pr show --id ${id} --output json` |
| [repos-pr-work-item-list.schema.json](az/repos-pr-work-item-list.schema.json) | `az repos pr work-item list --id ${id} --output json` |
