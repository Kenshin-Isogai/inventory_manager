package httpapi

type APIEnvelope[T any] struct {
	Data T `json:"data"`
}
