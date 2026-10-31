package llm

import "expense-tracker/agent/internal/models"

// SelectModel resolves default/router roles without switching on provider errors.
func SelectModel(policy models.ModelPolicy, role string) (models.ModelSelection, error) {
	if err := policy.Validate(); err != nil {
		return models.ModelSelection{}, err
	}
	profile := policy.Default
	selection := models.ModelSelection{Requested: role}
	switch role {
	case "default":
	case "router":
		profile = policy.Router
	default:
		selection.Fallback = true
		selection.Reason = "missing_or_unknown_model_role"
	}
	selection.Selected = profile.Name
	selection.MaxOutputTokens = profile.MaxOutputTokens
	return selection, nil
}
