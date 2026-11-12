# jenkins

This page says what `jenkins-cli.jar` reads and writes where it meets kit's scope, for each service it reaches, and the
machine-readable descriptions each publishes. The index of these pages is [1st.md](1st.md).

| Service | Category | kit status |
| --- | --- | --- |
| [Jenkins](#jenkins) | [CI](ci.md) | `none` |

## Jenkins

Jenkins ships its CLI as `jenkins-cli.jar`, run as `java -jar jenkins-cli.jar -s "${jenkins_url}" -auth
"${user}:${token}" "${command}"`, or with `JENKINS_USER_ID` and `JENKINS_API_TOKEN`. It prints text. The REST API is
where JSON comes from: append `/api/json` to any page, and trim it with `tree=`. There is no paging. Stages come from
the Pipeline Stage View plugin's `wfapi` routes. A multibranch job keeps each branch and each pull request as a job of
its own (`main`, `PR-7`), which is how a run no change owns is told apart.

### Read from Jenkins

| Capability | Command |
| --- | --- |
| Recent runs | `GET /job/${job}/api/json?tree=builds[number,result,timestamp,duration,url]` (REST; `list-jobs` names jobs only) |
| A run's stages and failed step | `GET /job/${job}/${build}/wfapi/describe`, then `GET /job/${job}/${build}/execution/node/${node}/wfapi/describe` (REST) |
| A run's log | `java -jar jenkins-cli.jar console "${job}" "${build}"`; REST `GET /job/${job}/${build}/consoleText` |
| Artifacts | `artifacts[]` in `GET /job/${job}/${build}/api/json`; download `GET /job/${job}/${build}/artifact/${path}` (REST) |

### Write to Jenkins

| Capability | Command |
| --- | --- |
| Rerun | `java -jar jenkins-cli.jar replay-pipeline "${job}" -n "${build}"` |
| Cancel | `java -jar jenkins-cli.jar stop-builds "${job}"` stops every running build of the job; one build: `POST /job/${job}/${build}/stop` (REST) |
| Start a run | `java -jar jenkins-cli.jar build "${job}" -p "${key}=${value}" -s`; REST `POST /job/${job}/buildWithParameters` |
| Report a status | plugins post it: GitHub Branch Source a commit status, GitHub Checks a check run |

Jenkins lists no runs in JSON from its CLI, and the `wfapi` routes belong to a plugin. Blue Ocean is deprecated, and
Pipeline Graph View replaces it.

### Jenkins schemas

Jenkins publishes none. A `Jenkinsfile` is Groovy, and the REST API describes itself page by page at `/api`.
