package postman

import "time"

type Environment struct {
	ID                   string     `json:"id"`
	Name                 string     `json:"name"`
	Values               []EnvValue `json:"values"`
	PostmanVariableScope string     `json:"_postman_variable_scope,omitempty"`
	PostmanExportedAt    string     `json:"_postman_exported_at,omitempty"`
	PostmanExportedUsing string     `json:"_postman_exported_using,omitempty"`
}

type EnvValue struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Type    string `json:"type"`
	Enabled bool   `json:"enabled"`
}

func BuildEnvironment(name, baseURL string) Environment {
	return Environment{
		ID:   uuidV4(),
		Name: name,
		Values: []EnvValue{
			{Key: "baseUrl", Value: baseURL, Type: "text", Enabled: true},
		},
		PostmanVariableScope: "environment",
		PostmanExportedAt:    time.Now().Format(time.RFC3339),
		PostmanExportedUsing: "postman-gen",
	}
}
