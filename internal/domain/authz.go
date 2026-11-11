package domain

type CheckRequest struct {
	Subject string `json:"sub" binding:"required"`
	Domain  string `json:"dom" binding:"required"`
	Object  string `json:"obj" binding:"required"`
	Action  string `json:"act" binding:"required"`
}

type CheckResponse struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

type BatchCheckRequest struct {
	Checks []CheckRequest `json:"checks" binding:"required,min=1,max=100"`
}

type BatchCheckResult struct {
	Check   CheckRequest `json:"check"`
	Allowed bool         `json:"allowed"`
}

type PolicyRuleRequest struct {
	PType  string   `json:"ptype,omitempty"` // "p", "g", "g2"
	Params []string `json:"params,omitempty"`
	Sub    string   `json:"sub,omitempty"`
	Dom    string   `json:"dom,omitempty"`
	Obj    string   `json:"obj,omitempty"`
	Act    string   `json:"act,omitempty"`
}

// Authorizer defines the interface for authorization checks and rule management.
type Authorizer interface {
	Enforce(rvals ...interface{}) (bool, error)
	AddPolicy(params ...interface{}) (bool, error)
	RemovePolicy(params ...interface{}) (bool, error)
	AddGroupingPolicy(params ...interface{}) (bool, error)
	RemoveGroupingPolicy(params ...interface{}) (bool, error)
	LoadPolicy() error
}
