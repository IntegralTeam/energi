package mn_back

import (
	"bufio"
	"bytes"
	"github.com/IntegralTeam/energi/rlp"
	"github.com/magiconair/properties/assert"
	"reflect"
	"testing"
)

func TestHeartbeat_Serialization(t *testing.T) {
	heartbeat := Heartbeat{}

	heartbeat.Timestamp = 0xFFFFFFFFFFFF // 6 bytes
	assert.Equal(t, heartbeat.Time().Unix(), int64(heartbeat.Timestamp))
	heartbeat.Auth.Sig = make([]byte, 65, 65)
	heartbeat.Auth.Sig[0] = 5

	var b bytes.Buffer
	writer := bufio.NewWriter(&b)
	err := rlp.Encode(writer, &heartbeat)
	assert.Equal(t, err, nil)
	writer.Flush()

	heartbeatDecoded := Heartbeat{}
	reader := bufio.NewReader(&b)
	s := rlp.NewStream(reader, uint64(len(b.Bytes())))
	err = s.Decode(&heartbeatDecoded)
	assert.Equal(t, err, nil)

	assert.Equal(t, heartbeatDecoded, heartbeat)
}

// Check that different messages have different Hash and GetDataToSign
func TestHeartbeat_GetDataToSign_Hash(t *testing.T) {
	heartbeat := Heartbeat{}

	heartbeat.Timestamp = 0xFFFFFFFFFFFF // 6 bytes
	heartbeat.Auth.Sig = make([]byte, 65, 65)
	heartbeat.Auth.Sig[0] = 5

	heartbeat2 := heartbeat
	heartbeat2.Timestamp = 1

	heartbeat3 := heartbeat
	heartbeat3.Auth.Sig = make([]byte, 65, 65)
	heartbeat3.Auth.Sig[0] = 6

	assert.Equal(t, heartbeat.Hash().String() != heartbeat2.Hash().String(), true)
	assert.Equal(t, heartbeat.Hash().String() != heartbeat3.Hash().String(), true)

	assert.Equal(t, reflect.DeepEqual(heartbeat.GetDataToSign(), heartbeat2.GetDataToSign()), false)
	assert.Equal(t, reflect.DeepEqual(heartbeat.GetDataToSign(), heartbeat3.GetDataToSign()), true)
}
