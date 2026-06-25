package presence

// buildPrettyActivities mirrors zRuvix.Presence.Activity.build_pretty_activities.
//
// Note: the Elixir decorate_app_id/decorate_emoji clauses pattern-match on atom
// keys (:application_id, :emoji), but the activities arrive as JSON-decoded maps
// with string keys, so those clauses never match and the activities pass through
// unchanged. This reproduces that exact behaviour.
func buildPrettyActivities(activities any) []any {
	list, ok := activities.([]any)
	if !ok {
		return []any{}
	}
	return list
}
