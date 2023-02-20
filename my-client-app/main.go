/*
Copyright 2020 IBM All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/gateway"
)

func main() {
	os.Setenv("DISCOVERY_AS_LOCALHOST", "true")
	wallet, err := gateway.NewFileSystemWallet("wallet")
	if err != nil {
		fmt.Printf("Failed to create wallet: %s\n", err)
		os.Exit(1)
	}

	if !wallet.Exists("appUser") {
		err = populateWallet(wallet)
		if err != nil {
			fmt.Printf("Failed to populate wallet contents: %s\n", err)
			os.Exit(1)
		}
	}

	ccpPath := filepath.Join(
		"..",
		"test-network",
		"organizations",
		"peerOrganizations",
		"org4.example.com",
		"connection-org4.yaml",
	)

	gw, err := gateway.Connect(
		gateway.WithConfig(config.FromFile(filepath.Clean(ccpPath))),
		gateway.WithIdentity(wallet, "appUser"),
	)
	if err != nil {
		fmt.Printf("Failed to connect to gateway: %s\n", err)
		os.Exit(1)
	}
	defer gw.Close()

	network, err := gw.GetNetwork("mychannel")
	if err != nil {
		fmt.Printf("Failed to get network: %s\n", err)
		os.Exit(1)
	}

	contract := network.GetContract("basic")

	var option int

loop:
	for {
		fmt.Println("--- Enter the corresponding number ---\n")
		fmt.Println("1. Initialize ledger")
		fmt.Println("2. Query car by ID")
		fmt.Println("3. Query person by ID")
		fmt.Println("4. Transfer car ownership to different owner")
		fmt.Println("5. Paint car")
		fmt.Println("6. Add defect to car")
		fmt.Println("7. Repair all defects on car")
		fmt.Println("8. Query car by color")
		fmt.Println("9. Query car by owner and color")
		fmt.Println("10. Quit")

		fmt.Printf("Number: ")
		fmt.Scanf("%d", &option)
		fmt.Printf("\n")

		switch option {
		case 1:
			fmt.Println("--- Initializing ledger ---")

			initLedger(contract)

		case 2:
			fmt.Printf("Car ID: ")
			var id string
			fmt.Scanf("%s", &id)

			queryCar(contract, id)

		case 3:
			fmt.Printf("Person ID: ")
			var id string
			fmt.Scanf("%s", &id)

			queryPerson(contract, id)

		case 4:
			fmt.Printf("Car ID: ")
			var carID string
			fmt.Scanf("%s", &carID)

			fmt.Printf("New owner ID: ")
			var newOwnerID string
			fmt.Scanf("%s", &newOwnerID)

			fmt.Printf("Accept car with defects (Y/N)?: ")
			var acceptDefectsStr string
			fmt.Scanf("%s", &acceptDefectsStr)
			var acceptDefectsBool bool
			if acceptDefectsStr == "Y" {
				acceptDefectsBool = true
			} else {
				acceptDefectsBool = false
			}

			transferOwnershipOfCar(contract, carID, newOwnerID, acceptDefectsBool)

		case 5:
			fmt.Printf("Car ID: ")
			var id string
			fmt.Scanf("%s", &id)

			fmt.Printf("New color: ")
			var newColor string
			fmt.Scanf("%s", &newColor)

			paintCar(contract, id, newColor)

		case 6:
			fmt.Printf("Car ID: ")
			var id string
			fmt.Scanf("%s", &id)

			fmt.Println("Defect description:")
			var description string
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				description = scanner.Text()
			}

			fmt.Printf("Defect repair price: ")
			var repairPrice float32
			fmt.Scanf("%f", &repairPrice)

			createCarDefect(contract, id, description, repairPrice)

		case 7:
			fmt.Printf("Car ID: ")
			var id string
			fmt.Scanf("%s", &id)

			repairCar(contract, id)

		case 8:
			fmt.Printf("Car color: ")
			var color string
			fmt.Scanf("%s", &color)

			queryCarsByColor(contract, color)

		case 9:
			fmt.Printf("Car owner ID: ")
			var ownerID string
			fmt.Scanf("%s", &ownerID)

			fmt.Printf("Car color: ")
			var color string
			fmt.Scanf("%s", &color)

			queryCarsByOwnerAndColor(contract, ownerID, color)

		case 10:
			fmt.Printf("--- Quitting ---")
			break loop

		default:
			fmt.Println("Option not valid.")
		}

		fmt.Printf("\n\n")
	}
}

func populateWallet(wallet *gateway.Wallet) error {
	credPath := filepath.Join(
		"..",
		"test-network",
		"organizations",
		"peerOrganizations",
		"org4.example.com",
		"users",
		"User1@org4.example.com",
		"msp",
	)

	certPath := filepath.Join(credPath, "signcerts", "cert.pem")
	// read the certificate pem
	cert, err := ioutil.ReadFile(filepath.Clean(certPath))
	if err != nil {
		return err
	}

	keyDir := filepath.Join(credPath, "keystore")
	// there's a single file in this dir containing the private key
	files, err := ioutil.ReadDir(keyDir)
	if err != nil {
		return err
	}
	if len(files) != 1 {
		return errors.New("keystore folder should have contain one file")
	}
	keyPath := filepath.Join(keyDir, files[0].Name())
	key, err := ioutil.ReadFile(filepath.Clean(keyPath))
	if err != nil {
		return err
	}

	identity := gateway.NewX509Identity("Org4MSP", string(cert), string(key))

	err = wallet.Put("appUser", identity)
	if err != nil {
		return err
	}
	return nil
}

// InitLedger adds a base set of assets to the ledger
func initLedger(contract *gateway.Contract) {
	fmt.Printf("--- Submitting transaction: InitLedger ---\n")

	_, err := contract.SubmitTransaction("InitLedger")
	if err != nil {
		fmt.Println(fmt.Errorf("failed to submit transaction: %w", err))
		return
	}

	fmt.Printf("--- Transaction committed! ---\n")
}

func queryPerson(contract *gateway.Contract, id string) {
	fmt.Printf("--- Evaluate transaction: QueryPerson ---\n")

	evaluateResult, err := contract.EvaluateTransaction("QueryPerson", id)
	if err != nil {
		fmt.Println(fmt.Errorf("failed to evaluate transaction: %w", err))
		return
	}
	result := prettifyJSON(evaluateResult)

	fmt.Printf("--- Evaluated: %s\n", result)
}

func queryCar(contract *gateway.Contract, id string) {
	fmt.Printf("--- Evaluate transaction: QueryCar ---\n")

	evaluateResult, err := contract.EvaluateTransaction("QueryCar", id)
	if err != nil {
		fmt.Println(fmt.Errorf("failed to evaluate transaction: %w", err))
		return
	}
	result := prettifyJSON(evaluateResult)

	fmt.Printf("--- Evaluated: %s\n", result)
}

func transferOwnershipOfCar(contract *gateway.Contract, id string, newOwner string, defectsAccepted bool) {
	fmt.Printf("--- Submitting transaction: TransferOwnershipOfCar ---\n")

	_, err := contract.SubmitTransaction("TransferOwnershipOfCar", id, newOwner, strconv.FormatBool(defectsAccepted))
	if err != nil {
		fmt.Println(fmt.Errorf("failed to submit transaction: %w", err))
		return
	}

	fmt.Printf("--- Transaction committed! ---\n")
}

func paintCar(contract *gateway.Contract, id string, newColor string) {
	fmt.Printf("--- Submitting transaction: PaintCar ---\n")

	_, err := contract.SubmitTransaction("PaintCar", id, newColor)
	if err != nil {
		fmt.Println(fmt.Errorf("failed to submit transaction: %w", err))
		return
	}

	fmt.Printf("--- Transaction committed! ---\n")
}

func createCarDefect(contract *gateway.Contract, id string, description string, repairPrice float32) {
	fmt.Printf("--- Submitting transaction: CreateCarDefect ---\n")

	_, err := contract.SubmitTransaction("CreateCarDefect", id, description, fmt.Sprintf("%f", repairPrice))
	if err != nil {
		fmt.Println(fmt.Errorf("failed to submit transaction: %w", err))
		return
	}

	fmt.Printf("--- Transaction committed! ---\n")
}

func repairCar(contract *gateway.Contract, id string) {
	fmt.Printf("--- Submitting transaction: RepairCar ---\n")

	_, err := contract.SubmitTransaction("RepairCar", id)
	if err != nil {
		fmt.Println(fmt.Errorf("failed to submit transaction: %w", err))
		return
	}

	fmt.Printf("--- Transaction committed! ---\n")
}

func queryCarsByColor(contract *gateway.Contract, color string) {
	fmt.Println("--- Evaluate transaction: QueryCarsByColor ---\n")

	evaluateResult, err := contract.EvaluateTransaction("QueryCarsByColor", color)
	if err != nil {
		fmt.Println(fmt.Errorf("failed to evaluate transaction: %w", err))
		return
	}
	result := prettifyJSON(evaluateResult)

	fmt.Printf("--- Evaluated: %s\n", result)
}

func queryCarsByOwnerAndColor(contract *gateway.Contract, ownerID string, color string) {
	fmt.Println("--- Evaluate transaction: QueryCarsByOwnerAndColor ---\n")

	evaluateResult, err := contract.EvaluateTransaction("QueryCarsByOwnerAndColor", ownerID, color)
	if err != nil {
		fmt.Println(fmt.Errorf("failed to evaluate transaction: %w", err))
		return
	}
	result := prettifyJSON(evaluateResult)

	fmt.Printf("--- Evaluated: %s\n", result)
}

func prettifyJSON(data []byte) string {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, data, " ", ""); err != nil {
		panic(fmt.Errorf("failed to parse JSON: %w", err))
	}
	return prettyJSON.String()
}
