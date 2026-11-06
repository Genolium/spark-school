package bot

import (
	"sort"
	"strings"
)

// payStartParam is the parsed form of a /start pay deep link.
type payStartParam struct {
	Tier       string
	Promo      string
	Ref        string
	Source     string
	AttrToken  string
	PriceLower bool
}

// parsePayStartParam parses links built by the landing:
//
//	pay_<tier>[_<PROMO>][_pe_lower][_a_<token>][_ref_<code>][_src_<code>]
//
// The ref/src suffixes may follow the attribution token, so they are extracted
// before the token; the tier is the first segment of what remains, and the rest
// is the promo code (or a partner username).
func parsePayStartParam(param string) payStartParam {
	result := payStartParam{Tier: "accelerator"}
	payload := strings.TrimPrefix(param, "pay_")
	if payload == param {
		payload = ""
	}

	// Whole-payload forms used by direct bot links.
	if strings.HasPrefix(payload, "ref_") {
		result.Ref = strings.TrimPrefix(payload, "ref_")
		payload = ""
	} else if strings.HasPrefix(payload, "src_") {
		result.Source = strings.TrimPrefix(payload, "src_")
		payload = ""
	}

	// Suffixes can come in any order (promo, token, ref, source), so collect
	// every marker, sort by position and slice the value between neighbours.
	type marker struct {
		key string
		pos int
	}
	markers := make([]marker, 0, 3)
	for _, key := range []string{"_a_", "_ref_", "_src_"} {
		if idx := strings.Index(payload, key); idx != -1 {
			markers = append(markers, marker{key: key, pos: idx})
		}
	}
	if len(markers) > 0 {
		sort.Slice(markers, func(i, j int) bool { return markers[i].pos < markers[j].pos })
		actionEnd := markers[0].pos
		for i, m := range markers {
			end := len(payload)
			if i+1 < len(markers) {
				end = markers[i+1].pos
			}
			value := payload[m.pos+len(m.key) : end]
			switch m.key {
			case "_a_":
				result.AttrToken = sanitizeAttributionToken(value)
			case "_ref_":
				result.Ref = strings.TrimSpace(value)
			case "_src_":
				result.Source = strings.TrimSpace(value)
			}
		}
		payload = payload[:actionEnd]
	}

	switch {
	case strings.HasPrefix(payload, "basic"):
		result.Tier = "basic"
		payload = strings.TrimPrefix(strings.TrimPrefix(payload, "basic"), "_")
	case strings.HasPrefix(payload, "vip"):
		result.Tier = "vip"
		payload = strings.TrimPrefix(strings.TrimPrefix(payload, "vip"), "_")
	case strings.HasPrefix(payload, "accelerator"):
		result.Tier = "accelerator"
		payload = strings.TrimPrefix(strings.TrimPrefix(payload, "accelerator"), "_")
	}

	if strings.HasPrefix(payload, "pe_lower") {
		result.PriceLower = true
		payload = strings.TrimPrefix(strings.TrimPrefix(payload, "pe_lower"), "_")
	}

	result.Promo = strings.TrimPrefix(strings.TrimSpace(payload), "@")
	return result
}
