package story

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/goldsheva/discord-story-bot/internal/configs"
	"github.com/goldsheva/discord-story-bot/internal/dto"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
}

// APIError represents a structured error returned by Story API.
type APIError struct {
	Status int    `json:"status"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Raw    string `json:"-"`
}

func (e *APIError) Error() string {
	if e == nil {
		return "story api error"
	}
	if e.Title != "" || e.Detail != "" {
		return fmt.Sprintf("story api error: status=%d title=%s detail=%s", e.Status, e.Title, e.Detail)
	}
	return fmt.Sprintf("story api error: status=%d", e.Status)
}

func NewClient() *Client {
	config := configs.GetEnvConfig()
	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		baseURL:    config.STORY_API_BASE_URL,
		apiKey:     config.STORY_API_KEY,
	}
}

func (c *Client) doPost(path string, body interface{}, out interface{}) error {
	url := fmt.Sprintf("%s%s", c.baseURL, path)
	buf := &bytes.Buffer{}
	if body != nil {
		if err := json.NewEncoder(buf).Encode(body); err != nil {
			return err
		}
	}

	req, err := http.NewRequest("POST", url, buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Try to parse structured error
		var se APIError
		if err := json.Unmarshal(data, &se); err == nil && se.Status != 0 {
			se.Raw = string(data)
			return &se
		}
		return fmt.Errorf("story api error: status=%d body=%s", resp.StatusCode, string(data))
	}

	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			return err
		}
	}
	return nil
}

// GetAssetByID fetches IP asset(s) by ip id(s). It returns the first matching asset or nil if none.
func (c *Client) GetAssetByID(ipID string) (*dto.IPAsset, error) {
	reqBody := map[string]interface{}{
		"orderBy":         "blockNumber",
		"orderDirection":  "desc",
		"pagination":      map[string]interface{}{"offset": 0},
		"includeLicenses": true,
		"where":           map[string]interface{}{"ipIds": []string{ipID}},
	}
	var resp dto.AssetsResponse
	if err := c.doPost("/assets", reqBody, &resp); err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, nil
	}
	return &resp.Data[0], nil
}

// GetAssets is a generic fetch with custom where.
func (c *Client) GetAssets(where map[string]interface{}) ([]dto.IPAsset, error) {
	reqBody := map[string]interface{}{
		"orderBy":         "blockNumber",
		"orderDirection":  "desc",
		"pagination":      map[string]interface{}{"offset": 0},
		"includeLicenses": true,
		"where":           where,
	}
	var resp dto.AssetsResponse
	if err := c.doPost("/assets", reqBody, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// GetCollectionByAddress fetches collection metadata by contract address.
func (c *Client) GetCollectionByAddress(address string) (*dto.CollectionMetadata, error) {
	reqBody := map[string]interface{}{
		"orderBy":        "updatedAt",
		"orderDirection": "desc",
		"pagination":     map[string]interface{}{"offset": 0},
		"where":          map[string]interface{}{"collectionAddresses": []string{address}},
	}
	var resp dto.CollectionsResponse
	if err := c.doPost("/collections", reqBody, &resp); err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, nil
	}
	return &resp.Data[0].CollectionMetadata, nil
}

// GetCollectionDisputes fetches disputes counters for a collection contract address.
func (c *Client) GetCollectionDisputes(address string) (*dto.CollectionItem, error) {
	reqBody := map[string]interface{}{
		"orderBy":        "updatedAt",
		"orderDirection": "desc",
		"pagination":     map[string]interface{}{"offset": 0},
		"where":          map[string]interface{}{"collectionAddresses": []string{address}},
	}
	var resp dto.CollectionsResponse
	if err := c.doPost("/collections", reqBody, &resp); err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		// nothing found
		return nil, nil
	}
	return &resp.Data[0], nil
}

// GetCollectionMedia fetches media fields for a collection contract address.
// Note: collection media endpoint is not exposed; use GetCollectionByAddress as needed.

// Convenience methods that extract specific blocks from an asset
func (c *Client) GetAssetTerms(ipID string) (*dto.LicenseTermsWrapper, error) {
	asset, err := c.GetAssetByID(ipID)
	if err != nil {
		return nil, err
	}
	if asset == nil {
		return nil, nil
	}
	// Prefer primary LicenseTemplate, else try the first item in Licenses array
	var lt *dto.LicenseTermsWrapper
	if asset.LicenseTemplate != nil {
		lt = asset.LicenseTemplate
	} else if len(asset.Licenses) > 0 {
		lt = &asset.Licenses[0]
	}
	if lt == nil {
		return nil, nil
	}
	// normalize AttributionRequired from available attribution flags
	if lt.Terms != nil {
		lt.Terms.AttributionRequired = lt.Terms.DerivativesAttribution || lt.Terms.CommercialAttribution
	}
	return lt, nil
}

func (c *Client) GetAssetInfringement(ipID string) ([]dto.InfringementStatus, error) {
	asset, err := c.GetAssetByID(ipID)
	if err != nil {
		return nil, err
	}
	if asset == nil {
		return nil, nil
	}
	return asset.Infringement, nil
}

func (c *Client) GetAssetModeration(ipID string) (*dto.ModerationStatus, error) {
	asset, err := c.GetAssetByID(ipID)
	if err != nil {
		return nil, err
	}
	if asset == nil {
		return nil, nil
	}
	return asset.Moderation, nil
}

func (c *Client) GetAssetMint(ipID string) (*dto.NFTMint, error) {
	asset, err := c.GetAssetByID(ipID)
	if err != nil {
		return nil, err
	}
	if asset == nil {
		return nil, nil
	}
	// Prefer mint present under NFTMetadata (per openapi) and fallback to top-level
	if asset.NFTMetadata != nil && asset.NFTMetadata.Mint != nil {
		return asset.NFTMetadata.Mint, nil
	}
	return asset.Mint, nil
}

func (c *Client) GetAssetCollection(ipID string) (*dto.CollectionInfo, error) {
	asset, err := c.GetAssetByID(ipID)
	if err != nil {
		return nil, err
	}
	if asset == nil {
		return nil, nil
	}
	return asset.Collection, nil
}
