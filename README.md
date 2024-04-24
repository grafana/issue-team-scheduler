# GitHub action to auto-assign issues to team members based on their availability and busyness

This repository contains 2 actions which may be useful to someone assigning issues to dedicated engineering teams and distributing (automatically assigning) them based on the availability of individual engineers.

## Regex labeler

This github action allows to assign labels based on regular expressions which are ranked against the issue title and body it's run against. This allows to assign team specific labels by matching Issues against these regex's. 

TODO: Document configuration and usage

## IC-Assignment

This action allows to distribute new issues among individuals based on their load and availability.

### Basic process

* Teams are matched by given label
* Availability is checked by fetching the ical or google calendar availability per person. A person is not seen as available if they have busy events scheduled for periods longer than 4hr the given business day.
* Load is determined by checking if team members already have issues assigned to them which had activity in the past 5 business days (except `ignoreLables` marks an issue as ignored).


### Team configuration

The configuration file is fetched from the repository this action runs in based on the `cfg-path` input parameter. `repo-token` is used to fetch this file, hence read access to the repo is required. In the Go version of the escalation scheduler `repo-token` has been renamed to `gh-token`.

Example config:
```yaml
teams:
  team_a:
    requireLabel: "product_a"
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
    matchers:
    - "^IMPORTANT:"
    members:
    - name: manager
      output: "slack-handle"
ignoreLabels:
  - stale
```

### Find matching team

Teams are taken from the configuration file. To match an issue against a team the following criteria have to be true:
* `requireLabel` will verify that the respective label is set on the issue. This can be a string, comma-seperated-list or array. In case of multiple labels specified, it's only checked that one of the labels match.
* `matchers` can be a list of regex expression. If defined, the team with the most matches will get the assignment. If a team has no `matchers` it will always be superseded by teams with a matcher. 

If no team is matched, the action will exit here.

### Configuration settings per team member

Within the configuration file you can specify various options per team member:
* `name`: The Github handle of the individual team member
* `ical-url`: The URL of the ical used for availability checking. Needs to be publicly accessible for now.
* `output`: Allows to define an individual output value set as `assignee` ouput in case this team member is assigned. For example this may be used to define a slack handle which is then passed to the next action in order to notify someone on slack.

#### How to add yourself to a team

Assuming our configuration file looks like this:
```yaml
teams:
  team_a:
    requireLabel: "product_a"
    members:
    - name: user1
      ical-url: JOHN_DOE_ICAL_URL
```

#### Add your ical calendar

The `ical-url` can be used to configure arbitrary ical urls to be used for availability checking. This URL needs to be publicly accessible without any authorization. As this is a less secure and less performant way of checking availability, it is recommended to use the Google calendar integration if possible.

#### Add your gcal calendar

This action works by using a google cloud service account to access the calendar of team members. Therefore it's required first to setup the necessary service account with access to the google calendar api.

Next members can share their availability with this service account via:
1. Open [Google Calendar](https://calendar.google.com/calendar/u/0/r)
2. Open Settings by opening the hamburger menu next to your personal calendar and clicking `Settings and sharing`
3. In the `Share with specific people or groups` section you click `+ Add people and groups`
4. Enter the email address of the service account and select `See only free/busy (hide details)` under Permissions.
5. Click `Send`

You are done.

### Calculate availability of team members

`members` of each team defines their respective members as a tuple of github name and optionally a ical calender to determine availability. If an ical calender is defined, the team member will be **marked as unavailable if an event exists (which marks them busy) for the day of the run which lasts longer than 4hrs**. If no ical is defined, team members are always seen as available.

* members can be part of multiple teams

### Filter all team members which are already busy with other escalations

Additionally team members who already have a relevant issue which matches the team criteria (`requireLabel`) assigned to them are filtered out.

An issue is relevant if it satisfies the following conditions:

- If it is open
  - It was updated within the "lookBackPeriod"
  - It does not have a label listed in "ignoreLabels" (for example "stale")
- If it is closed
  - It was closed within the "lookBackPeriod"

The "lookBackPeriod" is 5 days, except if it is includes a weekend day, in which case it is 7 days.

### Choose available team member

One of the remaining team members is randomly chosen (to be improved to fairly distribute issues between all members).

### Assign

Final step is to assign the issue to the actual member. If `dryRun` is set this step is only logged.

### Inputs

| Parameter                   | Type    | Required | Default                     | Description                                                                                              |
| --------------------------- | ------- | -------- | --------------------------- | -------------------------------------------------------------------------------------------------------- |
| `repo-token`                | String  | true     | $GITHUB_TOKEN               | Token to be used for all github operations.                                                              |
| `cfg-path`                  | String  | true     | .auto-assign-issue-cfg.yaml | Path to configuration file which contains definition of teams and requirements to match them on an issue |
| `abort-if-already-assigned` | Boolean | false    | false                       | Abort if issue already has assignee.                                                                     |
| `dry-run`                   | Boolean | false    | false                       | If set to true, assignment will only be logged.                                                          |
| `gcal_service_acount_key`   | String  | false    | ""                          | If set, this service account key will be used to check availability for google calendars.                |

### Outputs

| Parameter  | Type   | Default | Description                                                                                          |
| ---------- | ------ | ------- | ---------------------------------------------------------------------------------------------------- |
| `assignee` | String | `name`  | Name of the member assigned to the issue. If custom `output` is defined it's used instead of `name`. |

### Current limitations

* Not timezone aware - The intent is to distribute issues fairly between everybody, therefore we made the decision to not respect timezones. If you are interested in fixing this please open an issue and explain your usecase there. 
* Not completely fair - Some team members might resolve issues quicker than others and will potentially end up with more issues in the long run. Adding state to a github action to track the amount of completed issues is hard and people who get issues which are very hard to resolve (and therefore take a long time) would suffer from this. We will revisit the current approach after gaining experience with it, but start lightweight now. 
