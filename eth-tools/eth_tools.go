package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/CESSProject/cess-dcdn-components/config"
	"github.com/CESSProject/cess-dcdn-components/eth-tools/tools"
	"github.com/CESSProject/cess-dcdn-components/light-cacher/ctype"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   ctype.CACHE_NAME,
	Short: "ethereum tools for light cache node",
}

func Execute() {
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func InitCmd() {
	buyTokenCmd := &cobra.Command{
		Use:                   "buy-token",
		Short:                 "buy light cache node token",
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, args []string) {
			cpath, err := ParseFlagValue("config", "c", cmd)
			if err != nil {
				log.Fatal("error:", err)
			}
			screctKey, err := ParseFlagValue("screctKey", "s", cmd)
			if err != nil {
				log.Fatal("error:", err)
			}
			value, err := ParseFlagValue("value", "v", cmd)
			if err != nil {
				log.Fatal("error:", err)
			}
			value += "000000000000000000"
			conf := &tools.Config{}
			err = config.ParseCommonConfig(cpath, "yaml", conf)
			if err != nil {
				log.Fatal("error:", err)
			}
			tokenId, err := tools.BuyCacherToken(*conf, screctKey, value)
			if err != nil {
				log.Fatal("error:", err)
			}
			log.Println("success, token Id is ", tokenId)
		},
	}
	buyTokenCmd.Flags().StringP("config", "c", "", "custom profile")
	buyTokenCmd.Flags().StringP("screctKey", "s", "", "ethereum account private key")
	buyTokenCmd.Flags().StringP("value", "v", "0", "the amount required to purchase Token (in CESS)")

	signNodeCmd := &cobra.Command{
		Use:                   "sign-node",
		Short:                 "signature (node account, tokenId) message pair, used to cache node registration",
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, args []string) {
			nodeAcc, err := ParseFlagValue("nodeAcc", "n", cmd)
			if err != nil {
				log.Fatal("error:", err)
			}
			screctKey, err := ParseFlagValue("screctKey", "s", cmd)
			if err != nil {
				log.Fatal("error:", err)
			}
			tokenId, err := ParseFlagValue("tokenId", "t", cmd)
			if err != nil {
				log.Fatal("error:", err)
			}
			sign, err := tools.CacherAuthorizationSign(screctKey, nodeAcc, tokenId)
			if err != nil {
				log.Fatal("error:", err)
			}
			log.Println("success, the signature is ", hex.EncodeToString(sign))
		},
	}
	signNodeCmd.Flags().StringP("nodeAcc", "n", "", "cache node's ethereum account")
	signNodeCmd.Flags().StringP("screctKey", "s", "", "token owner's ethereum account private key")
	signNodeCmd.Flags().StringP("tokenId", "t", "", "the tokenId that needs to be bound")

	rootCmd.AddCommand(
		buyTokenCmd,
		signNodeCmd,
	)
}

func ParseFlagValue(name string, shorthand string, cmd *cobra.Command) (string, error) {
	value, err := cmd.Flags().GetString(name)
	if err != nil || value == "" {
		value, err = cmd.Flags().GetString(shorthand)
		if err != nil || value == "" {
			return "", fmt.Errorf("parse flag %s [%s] value error %v", name, shorthand, err)
		}
	}
	return value, nil
}

func main() {
	InitCmd()
	Execute()
}
