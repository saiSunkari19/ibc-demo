package app

import (
	"encoding/json"
	"log"
	
	abci "github.com/tendermint/tendermint/abci/types"
	tmtypes "github.com/tendermint/tendermint/types"
	
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	"github.com/cosmos/cosmos-sdk/x/staking"
)

func (app *InterchangeApp) ExportAppStateAndValidators(forZeroHeight bool, jailWhiteList []string,
) (appState json.RawMessage, validators []tmtypes.GenesisValidator, err error) {
	ctx := app.NewContext(true, abci.Header{Height: app.LastBlockHeight()})
	
	if forZeroHeight {
		app.prepForZeroHeightGenesis(ctx, jailWhiteList)
	}
	
	genState := app.mm.ExportGenesis(ctx)
	appState, err = codec.MarshalJSONIndent(app.cdc, genState)
	if err != nil {
		return nil, nil, err
	}
	validators = staking.WriteValidators(ctx, app.stakingKeeper) // TODO : exported authorities details like staking
	return appState, validators, nil
}

func (app *InterchangeApp) prepForZeroHeightGenesis(ctx sdk.Context, jailWhiteList []string) {
	applyWhiteList := false
	
	if len(jailWhiteList) > 0 {
		applyWhiteList = true
	}
	
	whiteListMap := make(map[string]bool)
	
	for _, addr := range jailWhiteList {
		_, err := sdk.ValAddressFromBech32(addr)
		if err != nil {
			log.Fatal(err)
		}
		whiteListMap[addr] = true
	}
	
	app.crisisKeeper.AssertInvariants(ctx)
	
	height := ctx.BlockHeight()
	ctx = ctx.WithBlockHeight(0)
	
	ctx = ctx.WithBlockHeight(height)
	
	app.stakingKeeper.IterateRedelegations(ctx, func(_ int64, red staking.Redelegation) (stop bool) {
		for i := range red.Entries {
			red.Entries[i].CreationHeight = 0
		}
		app.stakingKeeper.SetRedelegation(ctx, red)
		return false
	})
	
	app.stakingKeeper.IterateUnbondingDelegations(ctx, func(_ int64, ubd staking.UnbondingDelegation) (stop bool) {
		for i := range ubd.Entries {
			ubd.Entries[i].CreationHeight = 0
		}
		app.stakingKeeper.SetUnbondingDelegation(ctx, ubd)
		return false
	})
	
	store := ctx.KVStore(app.keys[staking.StoreKey])
	iter := sdk.KVStoreReversePrefixIterator(store, staking.ValidatorsKey)
	counter := int16(0)
	
	for ; iter.Valid(); iter.Next() {
		addr := sdk.ValAddress(iter.Key()[1:])
		validator, found := app.stakingKeeper.GetValidator(ctx, addr)
		if !found {
			panic("expected validator, not found")
		}
		
		validator.UnbondingHeight = 0
		if applyWhiteList && !whiteListMap[addr.String()] {
			validator.Jailed = true
		}
		
		app.stakingKeeper.SetValidator(ctx, validator)
		counter++
	}
	
	iter.Close()
	
	_ = app.stakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	
	app.slashingKeeper.IterateValidatorSigningInfos(
		ctx,
		func(addr sdk.ConsAddress, info slashing.ValidatorSigningInfo) (stop bool) {
			info.StartHeight = 0
			app.slashingKeeper.SetValidatorSigningInfo(ctx, addr, info)
			return false
		},
	)
}
