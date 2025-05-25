package monitor

import (
	"time"

	cfg "ibp-geodns/src/common/config"
)

type Result struct {
	Member    cfg.Member
	Status    bool
	Checktime time.Time
	ErrorText string
	Data      map[string]interface{}
}
