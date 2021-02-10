package config

import sdk "github.com/cosmos/cosmos-sdk/types"

//All of the OIP Configure in this package

const (
	DisableTransferToContractBlock = int64(10)
)

func IsDisableTransferToContract(ctx sdk.Context) bool {
	return ctx.BlockHeight() >= DisableTransferToContractBlock
}
