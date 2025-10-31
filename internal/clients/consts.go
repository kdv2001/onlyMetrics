package clients

// Scheme Виды схем
type Scheme string

const (
	HTTP  Scheme = "http"
	HTTPS Scheme = "https"
)

// String возвращает строковое представление схемы
func (scheme Scheme) String() string {
	return string(scheme)
}
