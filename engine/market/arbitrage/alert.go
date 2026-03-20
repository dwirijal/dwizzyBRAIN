package arbitrage

import "fmt"

func FormatDiscordAlert(opportunity Opportunity) string {
	return fmt.Sprintf(
		"arb %s buy %s @ %.4f sell %s @ %.4f spread %.4f%%",
		opportunity.CoinID,
		opportunity.BuyExchange,
		opportunity.BuyPrice,
		opportunity.SellExchange,
		opportunity.SellPrice,
		opportunity.GrossSpreadPct,
	)
}
