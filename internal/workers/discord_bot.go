package workers

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/goldsheva/discord-story-bot/internal/configs"
	dto "github.com/goldsheva/discord-story-bot/internal/dto"
	i18n_pkg "github.com/goldsheva/discord-story-bot/internal/i18n"
	storyclient "github.com/goldsheva/discord-story-bot/internal/story"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithFields(logrus.Fields{"gopher": "discord_bot"})

func getTextWithCtx(i *discordgo.InteractionCreate, key string) string {
	// Try interaction locale first
	if i != nil && i.Interaction != nil {
		var raw string
		if i.Interaction.Locale != "" {
			raw = fmt.Sprint(i.Interaction.Locale)
		} else if i.Interaction.GuildLocale != nil {
			raw = fmt.Sprint(i.Interaction.GuildLocale)
		}
		if raw != "" {
			loc := i18n_pkg.DetectLocale(strings.ToLower(raw))
			if loc == "" {
				loc = "en"
			}
			return i18n_pkg.T(loc, key)
		}
	}
	// Fallback to configured locale or en
	cfg := configs.GetEnvConfig()
	loc := i18n_pkg.DetectLocale(strings.ToLower(cfg.LOCALE))
	if loc == "" {
		loc = "en"
	}
	return i18n_pkg.T(loc, key)
}

// convenience wrapper when context is not available
func getText(key string) string {
	return getTextWithCtx(nil, key)
}

// --- Discord Bot ---
func GoDiscordBot(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	config := configs.GetEnvConfig()
	dg, err := discordgo.New("Bot " + config.BOT_TOKEN)
	if err != nil {
		log.Fatal("Error creating Discord session: ", err)
	}

	client := storyclient.NewClient()

	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type == discordgo.InteractionApplicationCommand {
			data := i.ApplicationCommandData()
			name := data.Name
			if len(data.Options) == 0 {
				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{Content: getTextWithCtx(i, "missing_param")},
				})
				return
			}
			param := data.Options[0].StringValue()

			// Dispatch commands
			switch name {
			case "license":
				handleLicense(s, i, client, param)
			case "license_terms":
				handleLicenseTerms(s, i, client, param)
			case "license_infringement":
				handleLicenseInfringement(s, i, client, param)
			case "license_moderation":
				handleLicenseModeration(s, i, client, param)
			case "license_mint":
				handleLicenseMint(s, i, client, param)
			case "license_collection":
				handleLicenseCollection(s, i, client, param)
			case "collection":
				handleCollection(s, i, client, param)
			case "collection_disputes":
				handleCollectionDisputes(s, i, client, param)
			default:
				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{Content: getTextWithCtx(i, "unknown_command")},
				})
			}
			return
		}

		// component interaction (button clicks)
		if i.Type == discordgo.InteractionMessageComponent {
			handleComponentInteraction(s, i, client)
			return
		}
	})

	if err := dg.Open(); err != nil {
		log.Fatal("Error opening connection: ", err)
	}

	createCommands(dg)
	log.Info("Discord bot is running...")
	<-ctx.Done()
	dg.Close()
	log.Warn("Discord bot successfully stopped!")
}

func createCommands(dg *discordgo.Session) {
	cmds := []*discordgo.ApplicationCommand{
		{Name: "license", Description: "Check license by IP ID", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionString, Name: "ip_id", Description: "Intellectual Property ID", Required: true}}},
		{Name: "license_terms", Description: "Show license terms", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionString, Name: "ip_id", Description: "IP ID", Required: true}}},
		{Name: "license_infringement", Description: "Show infringement status", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionString, Name: "ip_id", Description: "IP ID", Required: true}}},
		{Name: "license_moderation", Description: "Show moderation safety", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionString, Name: "ip_id", Description: "IP ID", Required: true}}},
		{Name: "license_mint", Description: "Show mint info", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionString, Name: "ip_id", Description: "IP ID", Required: true}}},
		{Name: "license_collection", Description: "Show collection/contract info for license", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionString, Name: "ip_id", Description: "IP ID", Required: true}}},
		{Name: "collection", Description: "Get collection info by contract address", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionString, Name: "address", Description: "Contract address", Required: true}}},
		{Name: "collection_disputes", Description: "Get collection disputes counters", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionString, Name: "address", Description: "Contract address", Required: true}}},
	}

	// desired command names set
	desired := map[string]struct{}{}
	for _, c := range cmds {
		desired[c.Name] = struct{}{}
	}

	for _, cmd := range cmds {
		if _, err := dg.ApplicationCommandCreate(dg.State.User.ID, "", cmd); err != nil {
			log.Errorf("Cannot create command %s: %v", cmd.Name, err)
		}
	}

	// Cleanup: delete any existing global commands that are not in desired set
	existing, err := dg.ApplicationCommands(dg.State.User.ID, "")
	if err != nil {
		log.Warnf("Failed to list application commands for cleanup: %v", err)
	} else {
		for _, ec := range existing {
			if _, ok := desired[ec.Name]; !ok {
				if derr := dg.ApplicationCommandDelete(dg.State.User.ID, "", ec.ID); derr != nil {
					log.Warnf("Failed to delete legacy command %s: %v", ec.Name, derr)
				} else {
					log.Infof("Deleted legacy global command: %s", ec.Name)
				}
			}
		}
	}
}

// --- helpers ---
func respondDeferred(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredChannelMessageWithSource})
	if err != nil {
		// ignore "already been acknowledged" errors (40060) which can happen
		// when a component interaction was already deferred by the caller.
		if strings.Contains(err.Error(), "already been acknowledged") || strings.Contains(err.Error(), "40060") {
			return nil
		}
		return err
	}
	return nil
}

func followupEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed) {
	// apply unified footer
	if embed != nil {
		embed.Footer = &discordgo.MessageEmbedFooter{Text: "🧩 Powered by Endorphine Stake"}
	}
	if len(embed.Fields) > 25 {
		embed.Fields = embed.Fields[:25]
	}
	if _, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{Embeds: []*discordgo.MessageEmbed{embed}}); err != nil {
		logrus.Error("Failed to send followup embed: ", err)
	}
}

func followupEmbedWithComponents(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed, components []discordgo.MessageComponent) (*discordgo.Message, error) {
	// apply unified footer
	if embed != nil {
		embed.Footer = &discordgo.MessageEmbedFooter{Text: "🧩 Powered by Endorphine Stake"}
	}
	if len(embed.Fields) > 25 {
		embed.Fields = embed.Fields[:25]
	}
	params := &discordgo.WebhookParams{Embeds: []*discordgo.MessageEmbed{embed}}
	if len(components) > 0 {
		params.Components = components
	}
	msg, err := s.FollowupMessageCreate(i.Interaction, true, params)
	if err != nil {
		logrus.Error("Failed to send followup embed: ", err)
		return nil, err
	}
	return msg, nil
}

func disableComponentsCopy(components []discordgo.MessageComponent) []discordgo.MessageComponent {
	var out []discordgo.MessageComponent
	for _, c := range components {
		if ar, ok := c.(discordgo.ActionsRow); ok {
			var rowComponents []discordgo.MessageComponent
			for _, rc := range ar.Components {
				if b, ok := rc.(discordgo.Button); ok {
					nb := b
					nb.Disabled = true
					rowComponents = append(rowComponents, nb)
				} else {
					rowComponents = append(rowComponents, rc)
				}
			}
			out = append(out, discordgo.ActionsRow{Components: rowComponents})
		} else {
			out = append(out, c)
		}
	}
	return out
}

// --- command handlers ---
func handleLicense(s *discordgo.Session, i *discordgo.InteractionCreate, client *storyclient.Client, ipId string) {
	if err := respondDeferred(s, i); err != nil {
		logrus.Error("defer failed: ", err)
		return
	}
	asset, err := client.GetAssetByID(ipId)
	if err != nil {
		// Friendly warn for invalid input formats
		if ae, ok := err.(*storyclient.APIError); ok {
			low := strings.ToLower(ae.Detail + " " + ae.Title)
			if strings.Contains(low, "invalid ip asset id") || strings.Contains(low, "invalid ip id") {
				followupEmbed(s, i, &discordgo.MessageEmbed{Title: "", Description: getTextWithCtx(i, "invalid_ip_id"), Color: 0xFFFF00})
				return
			}
		}
		followupEmbed(s, i, &discordgo.MessageEmbed{Title: "Error", Description: err.Error(), Color: 0xFF0000})
		return
	}
	if asset == nil {
		// Yellow warn if IP not found
		followupEmbed(s, i, &discordgo.MessageEmbed{Title: "License", Description: getText("not_found"), Color: 0xFFFF00})
		return
	}
	title := ipId
	desc := ""
	if asset.Title != "" {
		title = asset.Title
	}
	desc = asset.Description

	embed := &discordgo.MessageEmbed{Title: title, Description: desc, Color: 0x00FF00}
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: getTextWithCtx(i, "embed_id"), Value: ipId, Inline: false})
	var components []discordgo.MessageComponent
	if asset.OwnerAddress != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: getTextWithCtx(i, "embed_owner"), Value: asset.OwnerAddress, Inline: true})
	}
	// metadata
	var playLink string
	var contentType string
	if asset.NFTMetadata != nil {
		// Prefer explicit mediaType if provided, else infer from actual URLs
		mt := strings.ToLower(asset.NFTMetadata.MediaType)
		hasAnim := asset.NFTMetadata.Animation != nil && asset.NFTMetadata.Animation.OriginalUrl != ""
		hasImg := asset.NFTMetadata.Image != nil && asset.NFTMetadata.Image.OriginalUrl != ""
		switch mt {
		case "animation":
			if hasAnim {
				contentType = "Animation"
			} else if hasImg {
				contentType = "Image"
			}
		case "image":
			if hasImg {
				contentType = "Image"
			} else if hasAnim {
				contentType = "Animation"
			}
		default:
			if hasAnim {
				contentType = "Animation"
			} else if hasImg {
				contentType = "Image"
			}
		}

		// Only show the type in Metadata field (no URLs)
		if contentType != "" {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: getTextWithCtx(i, "embed_metadata"), Value: contentType, Inline: false})
		}

		// Play button for all media types; resolve URL by type without exposing it in the embed
		if contentType == "Image" {
			if asset.NFTMetadata.Image != nil && asset.NFTMetadata.Image.OriginalUrl != "" {
				playLink = asset.NFTMetadata.Image.OriginalUrl
			} else if asset.NFTMetadata.OriginalURL != "" {
				playLink = asset.NFTMetadata.OriginalURL
			} else if asset.NFTMetadata.ExternalURL != "" {
				playLink = asset.NFTMetadata.ExternalURL
			}
		} else if contentType == "Animation" {
			if asset.NFTMetadata.Animation != nil && asset.NFTMetadata.Animation.OriginalUrl != "" {
				playLink = asset.NFTMetadata.Animation.OriginalUrl
			} else if asset.NFTMetadata.OriginalURL != "" {
				playLink = asset.NFTMetadata.OriginalURL
			} else if asset.NFTMetadata.ExternalURL != "" {
				playLink = asset.NFTMetadata.ExternalURL
			}
		}
	}
	// add action buttons (scoped to user) - Terms | Infringement | Moderation | Mint | Collection
	// include user id in custom_id to restrict button usage
	uid := ""
	if i.Member != nil && i.Member.User != nil {
		uid = i.Member.User.ID
	} else if i.User != nil {
		uid = i.User.ID
	}
	if uid != "" {
		locale := i18n_pkg.DetectLocale(configs.GetEnvConfig().LOCALE)
		// choose collection button target: if we have contract address, call collection by address, else fallback to license collection
		collCID := fmt.Sprintf("lic:coll:%s:%s", ipId, uid)
		if asset.Contract != nil && asset.Contract.Address != "" {
			collCID = fmt.Sprintf("col:show:%s:%s", asset.Contract.Address, uid)
		}
		// primary row: up to 5 action buttons
		primaryRow := discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: i18n_pkg.T(locale, "btn_terms"), Style: discordgo.PrimaryButton, CustomID: fmt.Sprintf("lic:terms:%s:%s", ipId, uid)},
			discordgo.Button{Label: i18n_pkg.T(locale, "btn_infringement"), Style: discordgo.SecondaryButton, CustomID: fmt.Sprintf("lic:infr:%s:%s", ipId, uid)},
			discordgo.Button{Label: i18n_pkg.T(locale, "btn_moderation"), Style: discordgo.SecondaryButton, CustomID: fmt.Sprintf("lic:mod:%s:%s", ipId, uid)},
			discordgo.Button{Label: i18n_pkg.T(locale, "btn_mint"), Style: discordgo.SecondaryButton, CustomID: fmt.Sprintf("lic:mint:%s:%s", ipId, uid)},
			discordgo.Button{Label: i18n_pkg.T(locale, "btn_collection"), Style: discordgo.SecondaryButton, CustomID: collCID},
		}}
		components = append(components, primaryRow)
	}

	// if we have a play link (Image or Animation), add a separate link button row
	if playLink != "" {
		locale := i18n_pkg.DetectLocale(configs.GetEnvConfig().LOCALE)
		playRow := discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: i18n_pkg.T(locale, "btn_play"), Style: discordgo.LinkButton, URL: playLink},
		}}
		// prepend playRow so it appears above other action rows
		newComps := append([]discordgo.MessageComponent{playRow}, components...)
		components = newComps
	}
	if asset.CreatedAt != nil {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: getTextWithCtx(i, "embed_created"), Value: asset.CreatedAt.UTC().Format("2006-01-02"), Inline: true})
	}
	if asset.UpdatedAt != nil {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: getTextWithCtx(i, "embed_updated"), Value: asset.UpdatedAt.UTC().Format("2006-01-02"), Inline: true})
	}
	if len(components) > 0 {
		msg, err := followupEmbedWithComponents(s, i, embed, components)
		if err == nil && msg != nil {
			// schedule deletion after configured timeout; fallback to disabling if delete fails
			go func(channelID, messageID string, comps []discordgo.MessageComponent) {
				t := time.Duration(configs.GetEnvConfig().STORY_BUTTON_TIMEOUT_SEC) * time.Second
				time.Sleep(t)
				// try delete
				if derr := s.ChannelMessageDelete(channelID, messageID); derr != nil {
					logrus.Debugf("failed to delete message: %v, falling back to disabling components", derr)
					disabled := disableComponentsCopy(comps)
					_, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
						ID:         messageID,
						Channel:    channelID,
						Components: &disabled,
					})
					if err != nil {
						logrus.Debugf("failed to disable components: %v", err)
					}
				}
			}(msg.ChannelID, msg.ID, components)
		}
	} else {
		followupEmbed(s, i, embed)
	}
}

func handleLicenseTerms(s *discordgo.Session, i *discordgo.InteractionCreate, client *storyclient.Client, ipId string) {
	if err := respondDeferred(s, i); err != nil {
		logrus.Error(err)
		return
	}
	t, err := client.GetAssetTerms(ipId)
	if err != nil {
		if ae, ok := err.(*storyclient.APIError); ok {
			low := strings.ToLower(ae.Detail + " " + ae.Title)
			if strings.Contains(low, "invalid ip asset id") || strings.Contains(low, "invalid ip id") {
				followupEmbed(s, i, &discordgo.MessageEmbed{Title: "", Description: getText("invalid_ip_id"), Color: 0xFFFF00})
				return
			}
		}
		followupEmbed(s, i, &discordgo.MessageEmbed{Title: "Error", Description: err.Error(), Color: 0xFF0000})
		return
	}
	embed := &discordgo.MessageEmbed{Title: getTextWithCtx(i, "title_terms"), Color: 0x00AAFF}
	if t == nil {
		// Yellow warn if terms not found (likely random/invalid id)
		followupEmbed(s, i, &discordgo.MessageEmbed{Title: getTextWithCtx(i, "title_terms"), Description: getText("no_terms"), Color: 0xFFFF00})
		return
	}
	// Template fields split per spec
	if t.LicenseTemplateId != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: getTextWithCtx(i, "embed_template"), Value: t.LicenseTemplateId, Inline: false})
	}
	if t.TemplateName != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: getTextWithCtx(i, "embed_template_name"), Value: strings.ToUpper(t.TemplateName), Inline: false})
	}
	if t.TemplateMetadataUri != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: getTextWithCtx(i, "embed_template_url"), Value: t.TemplateMetadataUri, Inline: false})
	}
	if t.Terms != nil {
		yes := map[bool]string{true: "✅", false: "❌"}
		// Render with License first line, then a clean list
		lines := []string{
			fmt.Sprintf("%s: %s", getTextWithCtx(i, "embed_id"), ipId),
			"",
			fmt.Sprintf("%s: %s", getTextWithCtx(i, "embed_transferable"), yes[t.Terms.Transferable]),
			fmt.Sprintf("%s: %s", getTextWithCtx(i, "embed_commercial_use"), yes[t.Terms.CommercialUse]),
			fmt.Sprintf("%s: %s", getTextWithCtx(i, "embed_derivatives_allowed"), yes[t.Terms.DerivativesAllowed]),
			fmt.Sprintf("%s: %s", getTextWithCtx(i, "embed_derivatives_approval"), yes[t.Terms.DerivativesApproval]),
			fmt.Sprintf("%s: %s", getTextWithCtx(i, "embed_commercial_rev_share"), formatRevSharePercent(t.Terms.CommercialRevShare)),
			fmt.Sprintf("%s: %s", getTextWithCtx(i, "embed_attribution"), yes[t.Terms.AttributionRequired]),
		}
		embed.Description = strings.Join(lines, "\n")
	}
	followupEmbed(s, i, embed)
}

// formatRevSharePercent converts raw commercialRevShare integer into a human-friendly percent.
// Heuristic: values are large; treat them as scaled by 1e6 (e.g., 20000000 -> 20%).
func formatRevSharePercent(n int) string {
	if n <= 0 {
		return "0%"
	}
	pct := float64(n) / 1_000_000.0
	if math.Mod(pct, 1) == 0 {
		return fmt.Sprintf("%.0f%%", pct)
	}
	return fmt.Sprintf("%.2f%%", pct)
}

func handleLicenseInfringement(s *discordgo.Session, i *discordgo.InteractionCreate, client *storyclient.Client, ipId string) {
	if err := respondDeferred(s, i); err != nil {
		logrus.Error(err)
		return
	}
	arr, err := client.GetAssetInfringement(ipId)
	if err != nil {
		if ae, ok := err.(*storyclient.APIError); ok {
			low := strings.ToLower(ae.Detail + " " + ae.Title)
			if strings.Contains(low, "invalid ip asset id") || strings.Contains(low, "invalid ip id") {
				followupEmbed(s, i, &discordgo.MessageEmbed{Title: "", Description: getTextWithCtx(i, "invalid_ip_id"), Color: 0xFFFF00})
				return
			}
		}
		followupEmbed(s, i, &discordgo.MessageEmbed{Title: "Error", Description: err.Error(), Color: 0xFF0000})
		return
	}
	if len(arr) == 0 {
		followupEmbed(s, i, &discordgo.MessageEmbed{Title: "Infringement Status", Description: getText("no_checks"), Color: 0xFFFF00})
		return
	}
	status := "✅ Not infringing"
	color := 0x00FF00
	for _, it := range arr {
		if it.IsInfringing {
			status = "❌ Infringing"
			color = 0xFF0000
			break
		}
		if strings.ToLower(it.Status) != "succeeded" {
			status = "⚠️ Potential issue"
			color = 0xFFAA00
		}
	}
	embed := &discordgo.MessageEmbed{Title: fmt.Sprintf("%s — %s", getTextWithCtx(i, "title_infringement"), ipId), Color: color, Description: status}
	latest := arr[0]
	if latest.ProviderName != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: getTextWithCtx(i, "embed_provider"), Value: latest.ProviderName, Inline: true})
	}
	if latest.ResponseTime != nil {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: getTextWithCtx(i, "embed_checked"), Value: latest.ResponseTime.UTC().Format("2006-01-02 15:04 UTC"), Inline: true})
	}
	followupEmbed(s, i, embed)
}

func handleLicenseModeration(s *discordgo.Session, i *discordgo.InteractionCreate, client *storyclient.Client, ipId string) {
	if err := respondDeferred(s, i); err != nil {
		logrus.Error(err)
		return
	}
	m, err := client.GetAssetModeration(ipId)
	if err != nil {
		if ae, ok := err.(*storyclient.APIError); ok {
			low := strings.ToLower(ae.Detail + " " + ae.Title)
			if strings.Contains(low, "invalid ip asset id") || strings.Contains(low, "invalid ip id") {
				followupEmbed(s, i, &discordgo.MessageEmbed{Title: "", Description: getTextWithCtx(i, "invalid_ip_id"), Color: 0xFFFF00})
				return
			}
		}
		followupEmbed(s, i, &discordgo.MessageEmbed{Title: "Error", Description: err.Error(), Color: 0xFF0000})
		return
	}
	if m == nil {
		followupEmbed(s, i, &discordgo.MessageEmbed{Title: "Moderation", Description: getText("no_moderation"), Color: 0xFFFF00})
		return
	}
	// evaluate
	eval := evaluateModeration(m)
	color := 0x00FF00
	if eval == "Unsafe" {
		color = 0xFF0000
	} else if eval == "Review" {
		color = 0xFFAA00
	}
	embed := &discordgo.MessageEmbed{Title: fmt.Sprintf("%s — %s", getTextWithCtx(i, "title_moderation"), ipId), Color: color, Description: eval}
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: getTextWithCtx(i, "adult"), Value: m.Adult, Inline: true})
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: getTextWithCtx(i, "spoof"), Value: m.Spoof, Inline: true})
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: getTextWithCtx(i, "medical"), Value: m.Medical, Inline: true})
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: getTextWithCtx(i, "violence"), Value: m.Violence, Inline: true})
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: getTextWithCtx(i, "racy"), Value: m.Racy, Inline: true})
	followupEmbed(s, i, embed)
}

func evaluateModeration(m *dto.ModerationStatus) string {
	// Note: storyclient.ModerationStatus type is defined in dto; use string checks
	// If any field LIKELY or VERY_LIKELY -> Unsafe
	// Else if any POSSIBLE -> Review
	// Else Safe
	anyPossible := false
	for _, v := range []string{m.Adult, m.Spoof, m.Medical, m.Violence, m.Racy} {
		up := strings.ToUpper(v)
		if strings.Contains(up, "VERY_LIKELY") || strings.Contains(up, "LIKELY") {
			return "Unsafe"
		}
		if strings.Contains(up, "POSSIBLE") || strings.Contains(up, "POSSIBLY") {
			anyPossible = true
		}
	}
	if anyPossible {
		return "Review"
	}
	return "Safe"
}

func handleLicenseMint(s *discordgo.Session, i *discordgo.InteractionCreate, client *storyclient.Client, ipId string) {
	if err := respondDeferred(s, i); err != nil {
		logrus.Error(err)
		return
	}
	m, err := client.GetAssetMint(ipId)
	if err != nil {
		if ae, ok := err.(*storyclient.APIError); ok {
			low := strings.ToLower(ae.Detail + " " + ae.Title)
			if strings.Contains(low, "invalid ip asset id") || strings.Contains(low, "invalid ip id") {
				followupEmbed(s, i, &discordgo.MessageEmbed{Title: "", Description: getTextWithCtx(i, "invalid_ip_id"), Color: 0xFFFF00})
				return
			}
		}
		followupEmbed(s, i, &discordgo.MessageEmbed{Title: "Error", Description: err.Error(), Color: 0xFF0000})
		return
	}
	if m == nil {
		followupEmbed(s, i, &discordgo.MessageEmbed{Title: "Mint Info", Description: getText("no_mint"), Color: 0xFFFF00})
		return
	}
	embed := &discordgo.MessageEmbed{Title: fmt.Sprintf("%s — %s", getTextWithCtx(i, "title_mint"), ipId), Color: 0x0099FF}
	var comps []discordgo.MessageComponent
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: getTextWithCtx(i, "mint_address"), Value: m.MintAddress, Inline: false})
	if m.BlockNumber != nil {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: getTextWithCtx(i, "block_number"), Value: m.BlockNumber.String(), Inline: true})
	}
	if m.Timestamp != nil {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: getTextWithCtx(i, "mint_timestamp"), Value: m.Timestamp.UTC().Format("2006-01-02 15:04:05 UTC"), Inline: true})
	}
	if m.TransactionHash != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: getTextWithCtx(i, "transaction"), Value: m.TransactionHash, Inline: false})
		// storyscan button
		txUrl := fmt.Sprintf("https://www.storyscan.io/tx/%s", m.TransactionHash)
		locale := i18n_pkg.DetectLocale(configs.GetEnvConfig().LOCALE)
		comps = append(comps, discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.Button{Label: i18n_pkg.T(locale, "btn_storyscan"), Style: discordgo.LinkButton, URL: txUrl}}})
	}
	if m.OwnerAddress != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: getTextWithCtx(i, "embed_owner"), Value: m.OwnerAddress, Inline: true})
	}
	if m.LastUpdatedAt != nil {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: getTextWithCtx(i, "last_updated_metadata"), Value: m.LastUpdatedAt.UTC().Format(time.RFC3339), Inline: true})
	}
	if m.TimeLastUpdated != nil {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: getTextWithCtx(i, "last_updated_system"), Value: m.TimeLastUpdated.UTC().Format(time.RFC3339), Inline: true})
	}
	if len(comps) > 0 {
		followupEmbedWithComponents(s, i, embed, comps)
	} else {
		followupEmbed(s, i, embed)
	}
}

func handleLicenseCollection(s *discordgo.Session, i *discordgo.InteractionCreate, client *storyclient.Client, ipId string) {
	if err := respondDeferred(s, i); err != nil {
		logrus.Error(err)
		return
	}
	asset, err := client.GetAssetByID(ipId)
	if err != nil {
		followupEmbed(s, i, &discordgo.MessageEmbed{Title: "Error", Description: err.Error(), Color: 0xFF0000})
		return
	}
	if asset == nil {
		followupEmbed(s, i, &discordgo.MessageEmbed{Title: "Collection", Description: getText("no_collection"), Color: 0xFFFF00})
		return
	}
	embed := &discordgo.MessageEmbed{Title: getTextWithCtx(i, "title_collection"), Color: 0x00CC99}
	// Show License ID first
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: getTextWithCtx(i, "embed_id"), Value: ipId, Inline: false})
	// prefer nested nftMetadata.contract/collection per openapi; fallback to top-level fields
	var contract *dto.ContractInfo
	var collection *dto.CollectionInfo
	if asset.NFTMetadata != nil {
		if asset.NFTMetadata.Contract != nil {
			contract = asset.NFTMetadata.Contract
		}
		if asset.NFTMetadata.Collection != nil {
			collection = asset.NFTMetadata.Collection
		}
	}
	if contract == nil {
		contract = asset.Contract
	}
	if collection == nil {
		collection = asset.Collection
	}
	if contract != nil {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: getTextWithCtx(i, "contract_name"), Value: contract.Name, Inline: true})
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: getTextWithCtx(i, "symbol"), Value: contract.Symbol, Inline: true})
		if contract.Address != "" {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: getTextWithCtx(i, "contract_address"), Value: contract.Address, Inline: false})
		}
		if contract.TotalSupply != nil {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: getTextWithCtx(i, "total_supply"), Value: contract.TotalSupply.String() + " NFTs", Inline: true})
		}
	}
	// Collection name can be absent (one-off NFT). Always show the field per spec.
	if collection != nil && collection.Name != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: getTextWithCtx(i, "collection_name"), Value: collection.Name, Inline: false})
	} else {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: getTextWithCtx(i, "collection_name"), Value: "❌", Inline: false})
	}
	// add button to open full collection view, if we know the contract address and have user id
	uid := ""
	if i.Member != nil && i.Member.User != nil {
		uid = i.Member.User.ID
	} else if i.User != nil {
		uid = i.User.ID
	}
	if uid != "" && contract != nil && contract.Address != "" {
		locale := i18n_pkg.DetectLocale(configs.GetEnvConfig().LOCALE)
		actionRow := discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: i18n_pkg.T(locale, "btn_view_collection"), Style: discordgo.SecondaryButton, CustomID: fmt.Sprintf("col:show:%s:%s", contract.Address, uid)},
		}}
		_, _ = followupEmbedWithComponents(s, i, embed, []discordgo.MessageComponent{actionRow})
		return
	}
	followupEmbed(s, i, embed)
}

func handleCollection(s *discordgo.Session, i *discordgo.InteractionCreate, client *storyclient.Client, addr string) {
	if err := respondDeferred(s, i); err != nil {
		logrus.Error(err)
		return
	}
	meta, err := client.GetCollectionByAddress(addr)
	if err != nil {
		followupEmbed(s, i, &discordgo.MessageEmbed{Title: "Error", Description: err.Error(), Color: 0xFF0000})
		return
	}
	if meta == nil {
		followupEmbed(s, i, &discordgo.MessageEmbed{Title: "Collection", Description: getText("not_found"), Color: 0xFFFF00})
		return
	}
	embed := &discordgo.MessageEmbed{Title: fmt.Sprintf("%s %s", getTextWithCtx(i, "title_collection"), addr), Color: 0x0055FF}
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: getTextWithCtx(i, "name"), Value: meta.Name, Inline: true})
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: getTextWithCtx(i, "symbol"), Value: meta.Symbol, Inline: true})
	if meta.TotalSupply != nil {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: getTextWithCtx(i, "total_supply"), Value: meta.TotalSupply.String(), Inline: true})
	}
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: getTextWithCtx(i, "token_type"), Value: meta.TokenType, Inline: true})
	if meta.CreatedAt != nil {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: getTextWithCtx(i, "embed_created"), Value: meta.CreatedAt.UTC().Format("2006-01-02"), Inline: true})
	}
	if meta.UpdatedAt != nil {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: getTextWithCtx(i, "embed_updated"), Value: meta.UpdatedAt.UTC().Format("2006-01-02"), Inline: true})
	}
	// add action buttons (Disputes) - include user id if present
	uid := ""
	if i.Member != nil && i.Member.User != nil {
		uid = i.Member.User.ID
	} else if i.User != nil {
		uid = i.User.ID
	}
	if uid != "" {
		locale := i18n_pkg.DetectLocale(configs.GetEnvConfig().LOCALE)
		actionRow := discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: i18n_pkg.T(locale, "btn_disputes"), Style: discordgo.SecondaryButton, CustomID: fmt.Sprintf("col:disputes:%s:%s", addr, uid)},
		}}
		_, _ = followupEmbedWithComponents(s, i, embed, []discordgo.MessageComponent{actionRow})
		return
	}
	followupEmbed(s, i, embed)
}

func handleCollectionDisputes(s *discordgo.Session, i *discordgo.InteractionCreate, client *storyclient.Client, addr string) {
	if err := respondDeferred(s, i); err != nil {
		logrus.Error(err)
		return
	}
	item, err := client.GetCollectionDisputes(addr)
	if err != nil {
		followupEmbed(s, i, &discordgo.MessageEmbed{Title: "Error", Description: err.Error(), Color: 0xFF0000})
		return
	}
	if item == nil {
		// Yellow warn for not found collection address
		followupEmbed(s, i, &discordgo.MessageEmbed{Title: getTextWithCtx(i, "title_disputes"), Description: getText("not_found"), Color: 0xFFFF00})
		return
	}
	embed := &discordgo.MessageEmbed{Title: fmt.Sprintf("%s — %s", getTextWithCtx(i, "title_disputes"), addr), Color: 0x00AA00}
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Raised", Value: fmt.Sprintf("%d", item.RaisedDisputeCount), Inline: true})
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Resolved", Value: fmt.Sprintf("%d", item.ResolvedDisputeCount), Inline: true})
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Cancelled", Value: fmt.Sprintf("%d", item.CancelledDisputeCount), Inline: true})
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Judged", Value: fmt.Sprintf("%d", item.JudgedDisputeCount), Inline: true})
	followupEmbed(s, i, embed)
}

// handleComponentInteraction parses button custom_id and routes to command handlers.
// custom_id format: "lic:<action>:<ipId>" where action in terms|infringement|moderation|mint|collection
func handleComponentInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, client *storyclient.Client) {
	// Acknowledge interaction quickly to avoid 'Interaction Failed' (must respond within 3s)
	if err := respondDeferred(s, i); err != nil {
		// Log prominently and try a fallback immediate ephemeral response so the client
		// doesn't stay in "thinking" state.
		logrus.Warnf("failed to defer component interaction: %v; attempting fallback ACK", err)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "", Flags: 1 << 6},
		})
	}
	cid := i.MessageComponentData().CustomID
	parts := strings.Split(cid, ":")
	if len(parts) < 4 {
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{Content: getTextWithCtx(i, "invalid_component")})
		return
	}
	kind := parts[0]
	action := parts[1]
	// support both 4-part and 5-part formats
	var id, ownerId string
	if len(parts) == 4 {
		id = parts[2]
		ownerId = parts[3]
	} else if len(parts) >= 5 {
		// legacy formats may include an extra part; ignore mode
		id = parts[3]
		ownerId = parts[4]
	}

	// disable buttons immediately to avoid double clicks
	if i.Message != nil && len(i.Message.Components) > 0 {
		disabled := disableComponentsCopy(i.Message.Components)
		_, _ = s.ChannelMessageEditComplex(&discordgo.MessageEdit{ID: i.Message.ID, Channel: i.Message.ChannelID, Components: &disabled})
	}

	// restrict buttons to owner
	clicker := ""
	if i.Member != nil && i.Member.User != nil {
		clicker = i.Member.User.ID
	} else if i.User != nil {
		clicker = i.User.ID
	}
	if ownerId != "" && clicker != ownerId {
		// Send an ephemeral followup message to notify unauthorized user.
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: getTextWithCtx(i, "unauth_button"),
			Flags:   1 << 6,
		})
		return
	}
	// view toggle removed; any 'view:*' components are unknown now
	switch kind {
	case "lic":
		switch action {
		case "terms":
			handleLicenseTerms(s, i, client, id)
		case "infr":
			handleLicenseInfringement(s, i, client, id)
		case "mod":
			handleLicenseModeration(s, i, client, id)
		case "mint":
			handleLicenseMint(s, i, client, id)
		case "coll":
			handleLicenseCollection(s, i, client, id)
		default:
			_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{Content: getText("unknown_action")})
		}
	case "col":
		switch action {
		case "disputes":
			handleCollectionDisputes(s, i, client, id)
		case "show":
			// allow sub-actions: show:disputes:<addr>:<uid>
			// treat as open collection overview -> show collection
			handleCollection(s, i, client, id)
		default:
			_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{Content: getText("unknown_action")})
		}
	default:
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{Content: getText("unknown_component")})
	}
}
