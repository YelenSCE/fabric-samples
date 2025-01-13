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
		{ID: "exp", Amount: 5, Owner: "distrib"},
		{ID: "gem", Amount: 3000, Owner: "seo"},
		{ID: "gem", Amount: 3, Owner: "team1"},
		{ID: "exp", Amount: 6, Owner: "team1"},
		{ID: "gem", Amount: 15, Owner: "team2"},
		{ID: "exp", Amount: 30, Owner: "team2"},
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

	exists, err := s.AssetExists(ctx, id, owner)
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
	exists, err := s.AssetExists(ctx, id, owner)
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
func (s *SmartContract) AssetExists(ctx contractapi.TransactionContextInterface, id string, owner string) (bool, error) {
	compositeKey, err := ctx.GetStub().CreateCompositeKey("Asset", []string{id, owner})
	if err != nil {
		return false, fmt.Errorf("failed to create composite key for asset: %v", err)
	}

	assetJSON, err := ctx.GetStub().GetState(compositeKey)
	if err != nil {
		return false, fmt.Errorf("failed to read asset: %v", err)
	}

	return assetJSON != nil, nil
}

// TransferAsset updates the owner field of asset with given id and owner in world state, and returns the old owner.
func (s *SmartContract) TransferAsset(ctx contractapi.TransactionContextInterface, id string, owner string, newOwner string, amount int) (string, error) {
	asset, err := s.ReadAsset(ctx, id, owner)
	if err != nil {
		return "", err
	}

	if asset.Amount < amount {
		return "", fmt.Errorf("the asset %s does not have enough amount to transfer", id)
	}

	oldOwner := asset.Owner
	asset.Amount -= amount

	// Update the asset for the new owner
	newAsset := Asset{
		ID:     id,
		Amount: amount,
		Owner:  newOwner,
	}
	newAssetJSON, err := json.Marshal(newAsset)
	if err != nil {
		return "", err
	}

	newCompositeKey, err := ctx.GetStub().CreateCompositeKey("Asset", []string{id, newOwner})
	if err != nil {
		return "", fmt.Errorf("failed to create composite key for new asset: %v", err)
	}

	err = ctx.GetStub().PutState(newCompositeKey, newAssetJSON)
	if err != nil {
		return "", err
	}

	// Update the asset for the old owner
	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return "", err
	}

	oldCompositeKey, err := ctx.GetStub().CreateCompositeKey("Asset", []string{id, owner})
	if err != nil {
		return "", fmt.Errorf("failed to create composite key for old asset: %v", err)
	}

	err = ctx.GetStub().PutState(oldCompositeKey, assetJSON)
	if err != nil {
		return "", err
	}

	return oldOwner, nil
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
