package tests

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/crypto/keys"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/okex/okexchain/app/crypto/hd"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

const (
	// keys that provided on node (from test.sh)
	mnemo1          = "tragic ugly suggest nasty retire luxury era depth present cross various advice"
	mnemo2          = "miracle desert mosquito bind main cage fiscal because flip turkey brother repair"
	defaultPassWd   = "12345678"
	defaultCoinType = 60
)

var (
	keyInfo1, keyInfo2 keys.Info
	Kb                 = keys.NewInMemory(hd.EthSecp256k1Options()...)
	hexAddr1, hexAddr2 string
)

func init() {
	config := sdk.GetConfig()
	config.SetCoinType(defaultCoinType)

	keyInfo1, _ = createAccountWithMnemo(mnemo1, "alice", defaultPassWd)
	keyInfo2, _ = createAccountWithMnemo(mnemo2, "bob", defaultPassWd)
	hexAddr1 = common.BytesToAddress(keyInfo1.GetAddress().Bytes()).Hex()
	hexAddr2 = common.BytesToAddress(keyInfo2.GetAddress().Bytes()).Hex()
}

func TestGetAddress(t *testing.T) {
	addr, err := GetAddress()
	require.NoError(t, err)
	require.True(t, strings.EqualFold(hexutil.Encode(addr), hexAddr1))
}

func createAccountWithMnemo(mnemonic, name, passWd string) (info keys.Info, err error) {
	hdPath := keys.CreateHDPath(0, 0).String()
	info, err = Kb.CreateAccount(name, mnemonic, "", passWd, hdPath, hd.EthSecp256k1)
	if err != nil {
		return info, fmt.Errorf("failed. Kb.CreateAccount err : %s", err.Error())
	}

	return info, err
}
