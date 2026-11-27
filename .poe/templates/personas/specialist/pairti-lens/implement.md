# Implement a Lens change

Identify the entry point for the requested writer action and follow it to the widget, document buffer, or export path that owns the behavior. Use a representative project file for reproduction and a temporary copy for save checks.

Exercise the changed path through its normal entry point. Run the focused Go tests and `CGO_ENABLED=0 go test ./...`. For visual changes, apply the canvas or window checks specified in the viewer plan. Report the changed behavior and verification results.
