package main

type EvolutionMedia struct {
	MediaType string                 `json:"mediaType"`
	FileName  string                 `json:"fileName"`
	Caption   string                 `json:"caption"`
	Size      map[string]interface{} `json:"size"`
	MimeType  string                 `json:"mimetype"`
	Base64    string                 `json:"base64"`
	Buffer    string                 `json:"buffer"`
}
