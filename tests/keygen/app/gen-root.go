//go:build pivkey
// +build pivkey

package app

import (
	"fmt"
	"io/ioutil"
	"log"

	test "github.com/sigstore/root-signing/tests"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	outputCA     string
	outputSigner string
)

var genRootCmd = &cobra.Command{
	Use:   "gen-root",
	Short: "Gen root command",
	Long:  `Generates a test root CA`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("error initializing cmd line args: %s", err)
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		log.SetFlags(0)

		cert, signer, err := test.CreateRootCA()
		if err != nil {
			log.Fatal(err)
		}

		certPem, err := cryptoutils.MarshalCertificateToPEM(cert)
		if err != nil {
			log.Fatal(err)
		}
		if err := ioutil.WriteFile(outputCA, certPem, 0644); err != nil {
			log.Fatal(err)
		}

		signerPem, err := cryptoutils.MarshalPrivateKeyToPEM(signer)
		if err != nil {
			log.Fatal(err)
		}
		if err := ioutil.WriteFile(outputSigner, signerPem, 0644); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	genRootCmd.Flags().StringVar(&outputCA, "output-ca", "test-root-attestation.ca", "path to file to write output root CA")
	genRootCmd.Flags().StringVar(&outputSigner, "output-signer", "test-root.pem", "path to file to write output root CA signer")

	rootCmd.AddCommand(genRootCmd)
}
