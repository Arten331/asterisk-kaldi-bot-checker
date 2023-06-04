package botchecker

import (
	"errors"
	"net/url"
	"strings"

	"github.com/Arten331/bot-checker/internal/botchecker/metrics"
	"github.com/zaf/agi"
)

func (b *BotChecker) AgiHandler(session *agi.Session) error {
	req := session.Env["request"]

	u, err := url.Parse(req)
	if err != nil {
		return err
	}

	uriPath := strings.Split(u.Path, "/")
	path := uriPath[1]
	switch path { //nolint:wsl // for future endpoints
	case "wfn-store-metric":
		phase, ok := session.Env["arg_1"]
		if !ok {
			return errors.New("phase missing in argument 1")
		}

		checkRslt, ok := session.Env["arg_2"]
		if !ok {
			return errors.New("check result missing in argument 2")
		}

		campaign, ok := session.Env["arg_3"]
		if !ok {
			return errors.New("check result missing in argument 2")
		}

		dn := session.Env["accountcode"]
		callerID := session.Env["callerid"]

		b.Metrics.StoreNoiseHangup(&metrics.WaitForNoise{
			Phase:    phase,
			Caller:   callerID,
			Dnid:     dn,
			Result:   checkRslt,
			Campaign: campaign,
		})
	default:
		return errors.New("wrong AGI path")
	}

	return nil
}
