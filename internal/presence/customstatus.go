package presence

// buildCustomStatus extracts the Discord custom status (activity type 4) into a
// clean CustomStatus object. It returns nil when no custom status activity is
// present. Mirrors the dedicated-field treatment of Spotify / YouTube Music.
func buildCustomStatus(activity map[string]any) *CustomStatus {
	if activity == nil {
		return nil
	}

	cs := &CustomStatus{}

	if state, ok := activity["state"].(string); ok && state != "" {
		cs.Text = &state
	}

	if em, ok := activity["emoji"].(map[string]any); ok {
		emoji := &CustomStatusEmoji{}
		if name, ok := em["name"].(string); ok && name != "" {
			emoji.Name = &name
		}
		if id, ok := em["id"].(string); ok && id != "" {
			emoji.ID = &id
		}
		if animated, ok := em["animated"].(bool); ok {
			emoji.Animated = animated
		}
		cs.Emoji = emoji
	}

	return cs
}
