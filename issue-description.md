## Favourites Feature

Add the ability to mark AWS resources as favourites using the `f` keyboard shortcut. Favourited items are displayed with a `★` prefix and sorted to the top of command result tables. Favourites are persisted to disk in YAML format following k9s-style config storage.

### Behaviour

- Press `f` on a selected row to toggle its favourite status
- Favourited items get a `★` prefix in column 0 and are stable-sorted to the top
- A toast notification confirms the action (e.g. `★ my-bucket added` or `my-bucket removed`)
- The view refreshes from cache when available, avoiding unnecessary AWS CLI calls
- Drilling into a favourited item works correctly (the `★` prefix is stripped before variable substitution)

### Persistence

Favourites are stored at `~/.config/aws-commander/favourites.yaml` with the following structure:

```yaml
favourites:
  s3api:list-buckets:
    - my-important-bucket
    - another-bucket
  dynamodb:list-tables:
    - users
    - orders
```

### YAML Configuration

The `f` shortcut is opt-in per command via a new `favouritable` field in the YAML service definitions:

```yaml
commands:
  - name: "list-tables"
    resourceName: tableName
    favouritable: true
    # ...
```

The shortcut only appears when the current command has `favouritable: true`. It is automatically hidden on:
- Profile selection view
- Resource list view
- Command list view
- Dependent command selection view
- Any command without `favouritable: true`

### Enabled commands

`favouritable: true` is set on all top-level listing commands across all 13 service configs:

| Service | Command |
|---------|---------|
| dynamodb | `list-tables` |
| s3api | `list-buckets` |
| iam | `list-users`, `list-roles`, `list-groups`, `list-policies` |
| logs | `describe-log-groups` |
| eks | `list-clusters` |
| ecs | `list-clusters` |
| ec2 | `describe-instances`, `describe-security-groups`, `describe-vpcs`, `describe-volumes` |
| cloudtrail | `list-trails` |
| cloudfront | `list-distributions` |
| lambda | `list-functions` |
| events | `list-event-buses` |
| stepfunctions | `list-state-machines` |
| sqs | `list-queues` |

### Changes

| File | Change |
|------|--------|
| `cmd/favourites.go` | New — `FavouritesStore` with Load/Save/Toggle/IsFavourite/ApplyFavourites/StripFavouritePrefix |
| `cmd/favourites_test.go` | New — 6 unit tests covering toggle, lookup, sorting, mutation safety |
| `cmd/loader.go` | Added `Favouritable` field to `Command` struct |
| `handlers.go` | Added `handleToggleFavourite` handler, `f` shortcut gated by `Favouritable`, strip prefix in `itemHandler` |
| `parser/command.go` | Call `ApplyFavourites` in `parseToTableView` before creating the table |
| `main.go` | Call `Favourites.Load()` at startup |
| `views.go` | Reset `Command` state in `createResources`, `createCommandView`, `createDependentCommandView` |
| `configurations/*.yaml` | Added `favouritable: true` to 19 top-level listing commands across 13 configs |
