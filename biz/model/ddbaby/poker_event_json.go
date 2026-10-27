package ddbaby

import "encoding/json"

// PokerEventJSON carries an unchanged JSON token, rather than binary's default
// base64 encoding. Keep these methods outside generated code.
func (p *PokerEventJSON) UnmarshalJSON(data []byte) error {
	var raw json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*p = PokerEventJSON(raw)
	return nil
}

func (p PokerEventJSON) MarshalJSON() ([]byte, error) {
	return json.RawMessage(p).MarshalJSON()
}

// Acknowledgements/errors contain only BaseResp. A decision includes both card
// fields even when the board is nil (null) or empty ([]), as existing clients expect.
func (p TexasPokerEventResp) MarshalJSON() ([]byte, error) {
	if p.Result == nil {
		return json.Marshal(struct {
			BaseResp *BaseResp `json:"BaseResp"`
		}{BaseResp: p.BaseResp})
	}
	type response TexasPokerEventResp
	return json.Marshal(response(p))
}
