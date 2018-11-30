// Code generated by github.com/fjl/gencodec. DO NOT EDIT.

package mn_back

import (
	"encoding/json"
	"errors"

	"github.com/IntegralTeam/energi/common/hexutil"
)

var _ = (*authMarshaling)(nil)

// MarshalJSON marshals as JSON.
func (a Auth) MarshalJSON() ([]byte, error) {
	type Auth struct {
		Sig hexutil.Bytes `json:"sig" gencodec:"required"`
	}
	var enc Auth
	enc.Sig = a.Sig
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (a *Auth) UnmarshalJSON(input []byte) error {
	type Auth struct {
		Sig *hexutil.Bytes `json:"sig" gencodec:"required"`
	}
	var dec Auth
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.Sig == nil {
		return errors.New("missing required field 'sig' for Auth")
	}
	a.Sig = *dec.Sig
	return nil
}
