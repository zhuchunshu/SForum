package extensionmanifest

import "encoding/json"

type manifestJSONAlias Manifest

func (locale ManifestLocale) MarshalJSON() ([]byte, error) {
	type localeDocument struct {
		Name        string          `json:"name,omitempty"`
		Description string          `json:"description,omitempty"`
		URL         string          `json:"url,omitempty"`
		Author      *ManifestAuthor `json:"author,omitempty"`
	}
	var author *ManifestAuthor
	if locale.Author != (ManifestAuthor{}) {
		copy := locale.Author
		author = &copy
	}
	return json.Marshal(localeDocument{
		Name: locale.Name, Description: locale.Description, URL: locale.URL, Author: author,
	})
}

func (manifest *Manifest) UnmarshalJSON(body []byte) error {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(body, &object); err != nil {
		return err
	}
	settingsRaw := object["settings"]
	delete(object, "settings")
	base, err := json.Marshal(object)
	if err != nil {
		return err
	}
	var alias manifestJSONAlias
	if err := json.Unmarshal(base, &alias); err != nil {
		return err
	}
	document, err := decodeSettingsDocument(settingsRaw)
	if err != nil {
		return err
	}
	*manifest = Manifest(alias)
	manifest.Settings = document.Fields
	manifest.SettingsDocument = document
	return nil
}

func (manifest Manifest) MarshalJSON() ([]byte, error) {
	alias := manifestJSONAlias(manifest)
	body, err := json.Marshal(alias)
	if err != nil {
		return nil, err
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(body, &object); err != nil {
		return nil, err
	}
	settings, err := json.Marshal(manifest.SettingsDocument.canonicalValue(manifest.Settings))
	if err != nil {
		return nil, err
	}
	object["settings"] = settings
	return json.Marshal(object)
}
