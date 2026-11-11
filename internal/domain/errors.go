package domain

import "errors"

var (
	ErrUnauthorized       = errors.New("identidade do usuário ou tenant ausente no contexto")
	ErrForbidden          = errors.New("permissão insuficiente")
	ErrInternalAuthz      = errors.New("erro durante validação de autorização")
	ErrInvalidPolicy      = errors.New("política inválida")
	ErrPolicyAlreadyExist = errors.New("política já existente")
)

type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

func (e *AppError) Error() string {
	return e.Message
}
