package chaincode

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// SmartContract provides functions for managing an Asset
type SmartContract struct {
	contractapi.Contract
}

// Asset describes basic details of what makes up a simple asset
// Insert struct field in alphabetic order => to achieve determinism across languages
// golang keeps the order when marshal to json but doesn't order automatically
type Asset struct {
	ID     string `json:"ID"`
	Owner  string `json:"Owner"`
	Amount int    `json:"Amount"`
}

// InitLedger adds a base set of assets to the ledger
func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	assets := []Asset{
		{ID: "exp", Amount: 5, Owner: "Distrib"},
		{ID: "gem", Amount: 3000, Owner: "SEO"},
		{ID: "gem", Amount: 1, Owner: "Team1"},
		{ID: "exp", Amount: 6, Owner: "Team1"},
		{ID: "gem", Amount: 15, Owner: "Team2"},
		{ID: "exp", Amount: 15, Owner: "Team2"},
	}

	for _, asset := range assets {
		assetJSON, err := json.Marshal(asset)
		if err != nil {
			return err
		}

		compositeKey, err := ctx.GetStub().CreateCompositeKey("Asset", []string{asset.ID, asset.Owner})
		if err != nil {
			return err
		}

		err = ctx.GetStub().PutState(compositeKey, assetJSON)
		if err != nil {
			return err
		}
	}

	return nil
}

// CreateAsset issues a new asset to the world state with given details.
func (s *SmartContract) CreateAsset(ctx contractapi.TransactionContextInterface, id string, amount int, owner string) error {
	compositeKey, err := ctx.GetStub().CreateCompositeKey("Asset", []string{id, owner})
	if err != nil {
		return fmt.Errorf("failed to create composite key for asset: %v", err)
	}

	exists, err := s.AssetExists(ctx, compositeKey)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("the asset %s already exists", id)
	}

	asset := Asset{
		ID:     id,
		Amount: amount,
		Owner:  owner,
	}
	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(compositeKey, assetJSON)
}

// ReadAsset returns the asset stored in the world state with given id and owner.
func (s *SmartContract) ReadAsset(ctx contractapi.TransactionContextInterface, id string, owner string) (*Asset, error) {
	compositeKey, err := ctx.GetStub().CreateCompositeKey("Asset", []string{id, owner})
	if err != nil {
		return nil, fmt.Errorf("failed to create composite key: %v", err)
	}
	assetJSON, err := ctx.GetStub().GetState(compositeKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if assetJSON == nil {
		return nil, fmt.Errorf("the asset %s does not exist", id)
	}

	var asset Asset
	err = json.Unmarshal(assetJSON, &asset)
	if err != nil {
		return nil, err
	}

	return &asset, nil
}

// UpdateAsset updates an existing asset in the world state with provided parameters.
func (s *SmartContract) UpdateAsset(ctx contractapi.TransactionContextInterface, id string, owner string, amount int) error {
	exists, err := s.AssetExists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("the asset %s does not exist", id)
	}

	// overwriting original asset with new asset
	asset := Asset{
		ID:     id,
		Amount: amount,
		Owner:  owner,
	}
	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, assetJSON)
}

// AssetExists returns true when asset with given ID exists in world state
func (s *SmartContract) AssetExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	assetJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}

	return assetJSON != nil, nil
}

// TransferAsset updates the owner field of asset with given id in world state, and returns the old owner.
func (s *SmartContract) TransferAsset(ctx contractapi.TransactionContextInterface, id string, newOwner string, Amount int) (string, error) {
	asset, err := s.ReadAsset(ctx, id)
	if err != nil {
		return "", err
	}

	oldOwner := asset.Owner
	asset.Owner = newOwner

	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return "", err
	}

	err = ctx.GetStub().PutState(id, assetJSON)
	if err != nil {
		return "", err
	}

	return oldOwner, nil
}

// TransferGemToDistrib transfers Gem from a user to Distrib and credits Exp to the user's wallet
func (s *SmartContract) TransferGemToDistrib(ctx contractapi.TransactionContextInterface, user string, gemAmount int) error {
	if gemAmount <= 0 {
		return fmt.Errorf("gemAmount must be positive")
	}

	// Get the user's Gem asset
	userGemKey, err := ctx.GetStub().CreateCompositeKey("Asset", []string{"gem", user})
	if err != nil {
		return fmt.Errorf("failed to create composite key for user's Gem asset: %v", err)
	}
	userGemJSON, err := ctx.GetStub().GetState(userGemKey)
	if err != nil {
		return fmt.Errorf("failed to read user's Gem asset: %v", err)
	}
	if userGemJSON == nil {
		return fmt.Errorf("user does not own any Gem")
	}

	var userGem Asset
	err = json.Unmarshal(userGemJSON, &userGem)
	if err != nil {
		return fmt.Errorf("failed to unmarshal user's Gem asset: %v", err)
	}

	// Check if the user has enough Gem to transfer
	if userGem.Amount < gemAmount {
		return fmt.Errorf("insufficient Gem balance")
	}

	// Get Distrib's Gem asset
	distribGemKey, err := ctx.GetStub().CreateCompositeKey("Asset", []string{"gem", "Distrib"})
	if err != nil {
		return fmt.Errorf("failed to create composite key for Distrib's Gem asset: %v", err)
	}
	distribGemJSON, err := ctx.GetStub().GetState(distribGemKey)
	if err != nil {
		return fmt.Errorf("failed to read Distrib's Gem asset: %v", err)
	}
	var distribGem Asset
	if distribGemJSON != nil {
		err = json.Unmarshal(distribGemJSON, &distribGem)
		if err != nil {
			return fmt.Errorf("failed to unmarshal Distrib's Gem asset: %v", err)
		}
	} else {
		distribGem = Asset{ID: "gem", Owner: "Distrib", Amount: 0}
	}

	// Transfer Gem from user to Distrib
	userGem.Amount -= gemAmount
	distribGem.Amount += gemAmount

	// Update the user's Gem asset
	userGemJSON, err = json.Marshal(userGem)
	if err != nil {
		return fmt.Errorf("failed to marshal user's Gem asset: %v", err)
	}
	err = ctx.GetStub().PutState(userGemKey, userGemJSON)
	if err != nil {
		return fmt.Errorf("failed to update user's Gem asset: %v", err)
	}

	// Update Distrib's Gem asset
	distribGemJSON, err = json.Marshal(distribGem)
	if err != nil {
		return fmt.Errorf("failed to marshal Distrib's Gem asset: %v", err)
	}
	err = ctx.GetStub().PutState(distribGemKey, distribGemJSON)
	if err != nil {
		return fmt.Errorf("failed to update Distrib's Gem asset: %v", err)
	}

	// Get the user's Exp asset
	userExpKey, err := ctx.GetStub().CreateCompositeKey("Asset", []string{"exp", user})
	if err != nil {
		return fmt.Errorf("failed to create composite key for user's Exp asset: %v", err)
	}
	userExpJSON, err := ctx.GetStub().GetState(userExpKey)
	if err != nil {
		return fmt.Errorf("failed to read user's Exp asset: %v", err)
	}
	var userExp Asset
	if userExpJSON != nil {
		err = json.Unmarshal(userExpJSON, &userExp)
		if err != nil {
			return fmt.Errorf("failed to unmarshal user's Exp asset: %v", err)
		}
	} else {
		userExp = Asset{ID: "exp", Owner: user, Amount: 0}
	}

	// Credit Exp to the user's wallet
	userExp.Amount += gemAmount

	// Update the user's Exp asset
	userExpJSON, err = json.Marshal(userExp)
	if err != nil {
		return fmt.Errorf("failed to marshal user's Exp asset: %v", err)
	}
	err = ctx.GetStub().PutState(userExpKey, userExpJSON)
	if err != nil {
		return fmt.Errorf("failed to update user's Exp asset: %v", err)
	}

	return nil
}

// GetAllAssets returns all assets found in the world state
func (s *SmartContract) GetAllAssets(ctx contractapi.TransactionContextInterface) ([]Asset, error) {
	// Get all assets from the ledger
	resultsIterator, err := ctx.GetStub().GetStateByPartialCompositeKey("Asset", []string{})
	if err != nil {
		return nil, fmt.Errorf("failed to get assets: %v", err)
	}
	defer resultsIterator.Close()

	var assets []Asset
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate over assets: %v", err)
		}

		var asset Asset
		err = json.Unmarshal(queryResponse.Value, &asset)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal asset: %v", err)
		}
		assets = append(assets, asset)
	}

	return assets, nil
}
