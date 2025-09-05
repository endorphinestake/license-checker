package dto

import (
	"encoding/json"
	"time"
)

// Minimal DTOs mapping only fields required by the Task.md

type AssetsResponse struct {
	Data []IPAsset `json:"data"`
}

type IPAsset struct {
	IPIDs           []string              `json:"ipIds"`
	IpId            string                `json:"ipId"`
	Title           string                `json:"title"`
	Description     string                `json:"description"`
	OwnerAddress    string                `json:"ownerAddress"`
	CreatedAt       *time.Time            `json:"createdAt"`
	UpdatedAt       *time.Time            `json:"updatedAt"`
	NFTMetadata     *NFTMetadata          `json:"nftMetadata"`
	LicenseTemplate *LicenseTermsWrapper  `json:"licenseTemplate"`
	Licenses        []LicenseTermsWrapper `json:"licenses"`
	Infringement    []InfringementStatus  `json:"infringementStatus"`
	Moderation      *ModerationStatus     `json:"moderationStatus"`
	Mint            *NFTMint              `json:"mint"`
	Contract        *ContractInfo         `json:"contract"`
	Collection      *CollectionInfo       `json:"collection"`
	TokenContract   string                `json:"tokenContract"`
	TokenId         string                `json:"tokenId"`
}

type NFTMetadata struct {
	MediaType       string                `json:"mediaType"`
	Image           *NFTImageMetadata     `json:"image"`
	Animation       *NFTAnimationMetadata `json:"animation"`
	Contract        *ContractInfo         `json:"contract"`
	Collection      *CollectionInfo       `json:"collection"`
	OriginalURL     string                `json:"originalUrl"`
	ExternalURL     string                `json:"externalUrl"`
	Mint            *NFTMint              `json:"mint"`
	TimeLastUpdated *time.Time            `json:"timeLastUpdated"`
}

type NFTImageMetadata struct {
	OriginalUrl string `json:"originalUrl"`
}

type NFTAnimationMetadata struct {
	OriginalUrl string `json:"originalUrl"`
}

type LicenseTermsWrapper struct {
	TemplateName        string        `json:"templateName"`
	TemplateMetadataUri string        `json:"templateMetadataUri"`
	LicenseTemplateId   string        `json:"licenseTemplateId"`
	Terms               *LicenseTerms `json:"terms"`
}

type LicenseTerms struct {
	Transferable           bool `json:"transferable"`
	CommercialUse          bool `json:"commercialUse"`
	DerivativesAllowed     bool `json:"derivativesAllowed"`
	DerivativesApproval    bool `json:"derivativesApproval"`
	CommercialRevShare     int  `json:"commercialRevShare"`
	AttributionRequired    bool `json:"attributionRequired"`
	DerivativesAttribution bool `json:"derivativesAttribution"`
	CommercialAttribution  bool `json:"commercialAttribution"`
}

type InfringementStatus struct {
	Status              string     `json:"status"`
	IsInfringing        bool       `json:"isInfringing"`
	ProviderName        string     `json:"providerName"`
	ProviderURL         string     `json:"providerURL"`
	InfringementDetails string     `json:"infringementDetails"`
	ResponseTime        *time.Time `json:"responseTime"`
	CreatedAt           *time.Time `json:"createdAt"`
	UpdatedAt           *time.Time `json:"updatedAt"`
}

type ModerationStatus struct {
	Adult    string `json:"adult"`
	Spoof    string `json:"spoof"`
	Medical  string `json:"medical"`
	Violence string `json:"violence"`
	Racy     string `json:"racy"`
}

type NFTMint struct {
	MintAddress     string       `json:"mintAddress"`
	BlockNumber     *json.Number `json:"blockNumber"`
	Timestamp       *time.Time   `json:"timestamp"`
	TransactionHash string       `json:"transactionHash"`
	OwnerAddress    string       `json:"ownerAddress"`
	LastUpdatedAt   *time.Time   `json:"lastUpdatedAt"`
	TimeLastUpdated *time.Time   `json:"timeLastUpdated"`
}

type ContractInfo struct {
	Name        string       `json:"name"`
	Symbol      string       `json:"symbol"`
	Address     string       `json:"address"`
	TotalSupply *json.Number `json:"totalSupply"`
}

type CollectionInfo struct {
	Name            string `json:"name"`
	BannerImageUrl  string `json:"bannerImageUrl"`
	Slug            string `json:"slug"`
	ExternalUrl     string `json:"externalUrl"`
	Description     string `json:"description"`
	TwitterUsername string `json:"twitterUsername"`
	DiscordUrl      string `json:"discordUrl"`
}

// Collections response
type CollectionsResponse struct {
	Data []CollectionItem `json:"data"`
}

type CollectionItem struct {
	CollectionMetadata    CollectionMetadata `json:"collectionMetadata"`
	CancelledDisputeCount int                `json:"cancelledDisputeCount"`
	ResolvedDisputeCount  int                `json:"resolvedDisputeCount"`
	RaisedDisputeCount    int                `json:"raisedDisputeCount"`
	JudgedDisputeCount    int                `json:"judgedDisputeCount"`
}

type CollectionMetadata struct {
	Address         string       `json:"address"`
	Name            string       `json:"name"`
	Symbol          string       `json:"symbol"`
	TotalSupply     *json.Number `json:"totalSupply"`
	TokenType       string       `json:"tokenType"`
	CreatedAt       *time.Time   `json:"createdAt"`
	UpdatedAt       *time.Time   `json:"updatedAt"`
	BannerImageUrl  string       `json:"bannerImageUrl"`
	Slug            string       `json:"slug"`
	ExternalUrl     string       `json:"externalUrl"`
	Description     string       `json:"description"`
	TwitterUsername string       `json:"twitterUsername"`
	DiscordUrl      string       `json:"discordUrl"`
}
