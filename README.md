# GitHub Action(s) to help automatically assign issues to the right persons

This repository contains 2 actions which are useful to someone assigning issues to dedicated engineering teams and distributing (automatically assigning) them based on the availability of individual engineers:

## Regex labeler

This action allows to assign labels based on regular expressions which are ranked against the issue title and body it's run against. This allows to assign team specific labels by matching issues against these regex's.

### Inputs

| Parameter  | Type   | Required | Default                     | Description                                |
| ---------- | ------ | -------- | --------------------------- | ------------------------------------------ |
| `gh-token` | String | false    | `${{ github.token }}`       | The github token to be used for API access |
| `cfg-path` | String | true     | `.github/regex-labeler.yml` | Path for regex labeler config file         |

### Configuration

An exemplary configuration looks like this:

```yaml
required_labels:
labels:
  potential-bug:
    matchers:
      - regex: "(bug|not working)"
  performance-degradation:
    matchers:
      - regex: "(slower|increased.*consumption)"
        weight: 10
```

In this example the action would apply to all new issues (as no `required_labels` are set). If the issue title and/or body contains the words "bug" or "not working" it would be labeled as `potential-bug` by default. But if the words `slower` or a combination of `increased` and `consumption` is matched, the label `performance-degradation` would be assigned instead (given the higher weight).

#### Configuration structs

Root configuration struct:

| Parameter         | Type                        | Required | Default | Description                                                                                                                                                      |
| ----------------- | --------------------------- | -------- | ------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `required_labels` | List of Strings             | false    | `[]`    | List of labels which are required to run this action. If it's triggered on an issue which doesn't have **all** required labels, it exits without doing something |
| `labels`          | Map op label configurations | true     | `nil`   | Definition of which labels are assigned by which matcher configuration. Only the label with the best matching matchers is assigned                               |

Matcher configuration:

| Parameter  | Type            | Required | Default | Description                                                                                                                                                                                             |
| ---------- | --------------- | -------- | ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `matchers` | List of Matcher | false    | `[]`    | List matchers which are required to assign this label. Each matching matcher increases the likeliness (by weight) the owning label is assigned. At least one matcher needs to match to assign a label. |

Matcher:

| Parameter | Type    | Required | Default | Description                                                                                                                                                                                         |
| --------- | ------- | -------- | ------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `regex`   | String  | true     | ``      | The regular expression a label issue and body are checked against. If there is any match, this matcher is seen as successful                                                                        |
| `weight`  | Integer | false    | `1`     | The weight of this matcher. Can be used to overwrite other label's matcher. E.g. A weight of 10, would overrule another label with 9 individual matchers matching (if they have the default weight) |



## IC-Assignment

This action assigns individual members of teams to an incoming issue. First the matching team is determined by a set of labels required by a given team. After a team has been matched, it tries to assign the issue to the member of a team who is available and least busy (in comparison to the rest of their team). If multiple members of a team are seen as available and have the same lowest level of busyness, the issue is assigned randomly to one of them. In case no one is found who is available, the action will still assign it to someone in the team (chosen randomly) to ensure no issue is lost.

### Way of working

#### Determine availability of individual members of a team

Availability of individual members of a team is determined by accessing their calendar and checking for single events which mark them as busy for more than **6hrs** (not configurable at the moment) and is still ongoing at the time the issue is created or is about to start in the next 12-48hrs. The lookahead time differs as the process tries to accomodate weekends when usually no one is working. On a Friday the upcoming Monday is therefore checked for potential events of this kind.

Calendars are accessed either via a public ical feed or via google calendar api (see configuration below for more details).

#### Determine busyness of individual members of a team

Busyness of team members is calculated by the amount of issues someone is assigned to and which got updated in the past 7 days (not configurable at the moment). Issues updated in this timeframe are taken into account if
* Open issues: If they don't contain any of the labels which are configured to be ignored. This allows for example to ignore issues which got marked as `stale`
* Closed issues: If they were closed within the lookback time (7 days).

The higher this count, the more busy an individual team member is seen compared to other members.

### Inputs

| Parameter                 | Type    | Required | Default                       | Description                                                                                              |
| ------------------------- | ------- | -------- | ----------------------------- | -------------------------------------------------------------------------------------------------------- |
| `gh-token`                | String  | false    | `${{ github.token }}`         | The github token to be used for API access                                                               |
| `cfg-path`                | String  | true     | `.auto-assign-issue-cfg.yaml` | Path to configuration file which contains definition of teams and requirements to match them on an issue |
| `labels`                  | String  | false    | ``                            | The labels to use if you do not want to use the one provided by the GitHub issue                         |
| `dry-run`                 | Boolean | false    | `true`                        | If set to true, assignment will only be logged.                                                          |
| `gcal-service-acount-key` | String  | false    | ``                            | If set, this service account key will be used to check availability for google calendars.                |

### Outputs

| Parameter  | Type   | Default | Description                                                                                          |
| ---------- | ------ | ------- | ---------------------------------------------------------------------------------------------------- |
| `assignee` | String | `name`  | Name of the member assigned to the issue. If custom `output` is defined it's used instead of `name`. |

### Configuration

The configuration file is fetched from the repository this action runs in based on the `cfg-path` input parameter. `gh-token` is used to fetch this file, hence read access to the repo is required.

Example config:
```yaml
teams:
  team_a:
    requireLabel:
    - "product_a"
    members:
    - name: user1
      ical-url: https://.../cal.ics
  team_b:
    requireLabel: "product_b,product_B"
    members:
    - name: user1
      ical-url: https://.../cal.ics
  team_b_important:
    requireLabel:
    - "product_b"
    - "product_b-important"
    members:
    - name: manager
      output: "slack-handle"
ignoreLabels:
  - stale
unavailabilityLimit: 6h
```

#### Root configuration struct

| Parameter      | Type                       | Required | Default | Description                                                                                                                                                          |
| -------------- | -------------------------- | -------- | ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ignoreLabels` | List of Strings            | false    | `[]`    | List of labels which mark this issue to be ignored. If triggered on an issue which has at least **one** of the labels to be ignored, the action exits without doing something |
| `teams`        | Map of Team configurations | true     | `nil`   | Definition of the teams this issue is distributed between.                                                                                                           |
| `unavailabilityLimit` | Duration | false | `6h` | Duration for which a calendar event must block someone's availability for them to be considered unavailable. |

#### Team configuration struct

| Parameter      | Type            | Required | Default | Description                                                                                                                                                                                                     |
| -------------- | --------------- | -------- | ------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `requireLabel` | List of Strings | false    | `[]`    | List of labels which are required to match a given team. Only if all labels match, the issue may be assigned to someone of this team. If multiple teams match all members of all matching teams are considered. |
| `members`      | List of Members | true     | `nil`   | Definition of the individual members of a team.                                                                                                                                                                 |


#### Member configuration struct

| Parameter        | Type   | Required | Default | Description                                                                                                                                                                                                                                  |
| ---------------- | ------ | -------- | ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `name`           | String | true     | ``      | Github handle of a team member                                                                                                                                                                                                               |
| `output`         | String | false    | ``      | Value which is set as output of this action in case this member is assigned. E.g. can be used with slack handles to map users to their slack names and notify them in a later step in the workflow. If not specified `name` is used instead. |
| `ical-url`       | String | false    | ``      | Public ICal feed of this member used to determine availability of someone  at a given time.                                                                                                                                                  |
| `googleCalendar` | String | false    | ``      | Google Calendar name which is checked through the specified service account to determine availability. If set, `ical-url` is ignored.                                                                                                        |

### Considerations

#### Timezone awareness

During the initial implementation it was carefully discussed if the action should take timzeones into consideration (aka only assign issues to someone during their working hours). After in-depth analysis of issues in other projects we concluded that the creation time of issues don't align to working hours of the majority of the teams we were working in (e.g. majority of issues created during american working hours while majority of the team wasn't located there). **This led us to make the action not timezone aware as of today.** As this obviously cause issues when you need issues to handle in a timely manner we are open to proposals for improvement, if they take into account that our main priority is to distribute issues as fairly as possible across a team.

#### Fairness

Fairness in assigning issues to individuals of a team is subjective. As mean time to resolve depends on time of creation, issue complexity, productivity, knowledge, experience, availability and some degree of luck per team member it needs to be acknowledged that their is no objective fair distribution of issues among a group of people. Taking all these metrics into account is almost impossible and most likely would require additional state which we would need to store somewhere. We also discussed ideas like tracking the amount of completed issues per IC, but had to realise that issues which were hard to resolve (and therefore take a long time) would suffer from this. We will revisit the current approach after gaining experience with it, but start lightweight now.

### Google calendar configuration

If you want to use the functionality to determine availability based on someones `googleCalendar` a service account with google calendar api access is needed. To create this:

#### Service account creation

1. Open [Google cloud console](https://console.cloud.google.com/)
2. Choose a project this fits in / works well or create a new one as described [here](https://developers.google.com/workspace/guides/create-project)
3. Activate Google Calendar API for your project [here](https://console.cloud.google.com/flows/enableapi?apiid=calendar-json.googleapis.com)
4. Create a service account for your project as described [here](https://cloud.google.com/iam/docs/service-accounts-create)
5. Open the service account from the [service account overview](https://console.cloud.google.com/iam-admin/serviceaccounts)
6. **Remember the service account email**. This is the email all configured members have to share their calendar with (free/busy information is enough)
7. Go to **Keys** > `ADD KEY` and create a new json based key which is used as `gcal-service-acount-key` during workflow runs. It's recommended to store as a secret in your repo and use the secret during workflow runs as input.

#### Share calendars with the service account

This action works by using a google cloud service account to access the calendar of team members. Therefore it's required first to setup the necessary service account with access to the google calendar api.

Next members can share their availability with this service account via:
1. Open [Google Calendar](https://calendar.google.com/calendar/u/0/r)
2. On the left, find the "My calendars" section. To expand it, click the Down arrow.
3. Hover over your personal calendar and click on the three dots to open a dropdown. Click on `Settings and sharing` to open the settings.
4. In the `Share with specific people or groups` section you click `+ Add people and groups`
5. Enter the email address of the service account you created and select `See only free/busy (hide details)` under Permissions.
6. Click `Send` and you are done
