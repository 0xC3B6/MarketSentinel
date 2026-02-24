package fund

import (
	"encoding/json"
	"os"
	"time"

	"MarketSentinel/internal/model"
)

// LoadState reads the fund state from a JSON file. Returns a zero state if the file doesn't exist.
func LoadState(filePath string) (*model.FundState, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &model.FundState{}, nil
		}
		return nil, err
	}
	var state model.FundState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// SaveState writes the fund state to a JSON file.
func SaveState(filePath string, state *model.FundState) error {
	state.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}
