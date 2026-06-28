package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func runPlatforms(cmd *cobra.Command, args []string) error {
	fmt.Println("Supported Platforms:")
	fmt.Println()
	fmt.Println("  shopee - Shopee Indonesia")
	fmt.Println("    Types: seller, product_list, seller_rating, product_detail, product_rating")
	fmt.Println()
	fmt.Println("More platforms coming soon!")
	return nil
}
