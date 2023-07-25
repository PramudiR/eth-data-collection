package datacollector

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"math/rand"
	"net/http"
	"os"
	"strconv"

	"strings"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
)

func loadEnv() {
	if err := godotenv.Load(".env"); err != nil {
		fmt.Println("Error loading .env file:", err)
	}
}

const numTransactions = 15

func GasDataCollector(startTime string, endTime string, done chan bool) {
	// Lead environment variables from .env file
	loadEnv()

	// Read the INFURA_API_KEYS and ETHERSCAN_KEYS from environment variables
	infkeys := os.Getenv("INFURA_API_KEYS")
	ethkeys := os.Getenv("ETHERSCAN_KEYS")

	// Split the comma-separated keys into slices
	infuraApiKeys := strings.Split(infkeys, ",")
	etherscanKeys := strings.Split(ethkeys, ",")

	timeToStart, errWhenParsingStartTime := time.Parse(time.RFC3339, startTime)
	if errWhenParsingStartTime != nil {
		fmt.Println("An error occurred when parsing start time : ", errWhenParsingStartTime)
		return
	}

	end, errWhenParsingTimeEnd := time.Parse(time.RFC3339, endTime)
	if errWhenParsingTimeEnd != nil {
		fmt.Println("An error occurred when parsing time : ", errWhenParsingTimeEnd)
		return
	}

	apiKeyIndex := 0
	etherscanApiKeyIndex := 0
	var infuraClient *ethclient.Client
	var errWhenGetingInfuraClient error

	infuraClient, errWhenGetingInfuraClient = CreateInfuraClient(infuraApiKeys[apiKeyIndex])
	if errWhenGetingInfuraClient != nil {
		fmt.Println("Error when calling Infura client: ", errWhenGetingInfuraClient)
		return
	}

	timeObj := timeToStart.Unix()
	toTime := end.Unix()

	var blockUrl string

	//get the starting block
	blockUrl = CreateBlockUrl(etherscanKeys[etherscanApiKeyIndex], timeObj)

	var blockNoRes *http.Response
	var errWhenGettingBlockNumber error

	for {
		blockNoRes, errWhenGettingBlockNumber = http.Get(blockUrl)
		if errWhenGettingBlockNumber != nil {
			//rotate etherscan API key
			etherscanApiKeyIndex = (etherscanApiKeyIndex + 1) % len(etherscanKeys)
			blockUrl = CreateBlockUrl(etherscanKeys[etherscanApiKeyIndex], timeObj)
			blockNoRes, errWhenGettingBlockNumber = http.Get(blockUrl)
			if errWhenGettingBlockNumber != nil {
				fmt.Println("Error when calling the etherscan API to get initial block :" + errWhenGettingBlockNumber.Error())
				fmt.Println("Loop sleeping for 28 hours")
				time.Sleep(28 * time.Hour)
			}
		} else {
			break
		}
	}

	defer blockNoRes.Body.Close()
	body, errWhenReadingTheBlockNo := ioutil.ReadAll(blockNoRes.Body)
	if errWhenReadingTheBlockNo != nil {
		fmt.Println("Error when reading the block number from response : ", errWhenReadingTheBlockNo)
		return
	}

	var blockNoJsonResponse BlockNumberResponse
	errWhenUnMarshallingBlock := json.Unmarshal(body, &blockNoJsonResponse)
	if errWhenUnMarshallingBlock != nil {
		fmt.Println("Error when un marshalling : ", errWhenUnMarshallingBlock)
		return
	}

	var startBlock big.Int
	_, suc := startBlock.SetString(blockNoJsonResponse.Result, 10)
	if !suc {
		fmt.Println("Convert failed 0")
		return
	}

	//get the ending block;
	blockUrl = CreateBlockUrl(etherscanKeys[etherscanApiKeyIndex], toTime)
	for {
		blockNoRes, errWhenGettingBlockNumber = http.Get(blockUrl)
		if errWhenGettingBlockNumber != nil {
			//rotate etherscan API key
			etherscanApiKeyIndex = (etherscanApiKeyIndex + 1) % len(etherscanKeys)
			blockUrl = CreateBlockUrl(etherscanKeys[etherscanApiKeyIndex], timeObj)
			blockNoRes, errWhenGettingBlockNumber = http.Get(blockUrl)
			if errWhenGettingBlockNumber != nil {
				fmt.Println("Error when calling the etherscan API to get last block :" + errWhenGettingBlockNumber.Error())
				fmt.Println("Loop sleeping for 28 hours")
				time.Sleep(28 * time.Hour)
			}
		} else {
			break
		}
	}

	defer blockNoRes.Body.Close()
	body, errWhenReadingTheBlockNo = ioutil.ReadAll(blockNoRes.Body)
	if errWhenReadingTheBlockNo != nil {
		fmt.Println("Error when reading the block number from response : ", errWhenReadingTheBlockNo)
		return
	}

	errWhenUnMarshallingBlock = json.Unmarshal(body, &blockNoJsonResponse)
	if errWhenUnMarshallingBlock != nil {
		fmt.Println("Error when un marshalling : ", errWhenUnMarshallingBlock)
		return
	}

	var endBlock big.Int
	_, sucE := endBlock.SetString(blockNoJsonResponse.Result, 10)
	if !sucE {
		fmt.Println("Convert failed 1")
		return
	}

	//convert end block and start block into int
	startingBlock := startBlock.Int64()
	endingBlock := endBlock.Int64()

	for i := startingBlock; i < endingBlock; i++ {
		currentBlock := i

		var block *types.Block
		var errWhenLoadingBlock error

		var bigIntCurrentBlock big.Int
		bigIntCurrentBlock.SetInt64(currentBlock)
		//a do while approach where if an error occurs the loop will sleep for 25hours
		for {
			block, errWhenLoadingBlock = infuraClient.BlockByNumber(context.Background(), &bigIntCurrentBlock)
			if errWhenLoadingBlock != nil {
				//rotate API key
				apiKeyIndex = (apiKeyIndex + 1) % len(infuraApiKeys)
				infuraClient, errWhenGetingInfuraClient = CreateInfuraClient(infuraApiKeys[apiKeyIndex])
				if errWhenGetingInfuraClient != nil {
					fmt.Println("Error when calling Infura client: ", errWhenGetingInfuraClient)
					fmt.Println("Loop sleeping for 28 hours......")
					time.Sleep(28 * time.Hour)
				}
			} else {
				break
			}
		}

		//CSV file initialization
		fileName := strconv.FormatInt(currentBlock, 10)
		file, errWhenCreatingCSV := os.Create(fileName + ".csv")
		if errWhenCreatingCSV != nil {
			fmt.Println("Error when creating the CSV file : ", errWhenCreatingCSV)
			continue
		}
		defer file.Close()

		//write headers into CSV file
		headers := []string{"Timestamp", "Gas Price(Gwei)"}
		writer := csv.NewWriter(bufio.NewWriter(file))
		errWhenWritingHeadersToCsv := writer.Write(headers)
		if errWhenWritingHeadersToCsv != nil {
			fmt.Println("Error when writing to the headers CSV file : ", errWhenWritingHeadersToCsv)
			continue
		}
		headers = nil
		writer.Flush()

		//get the number of transactions in a block
		numTxns := len(block.Transactions())

		randomIndicies := GenerateRandomIndices(numTxns, numTransactions)

		selectedTxs := make([]*types.Transaction, 0)
		selectedCount := 0

		for _, idx := range randomIndicies {
			tx := block.Transactions()[idx]
			if tx.To() != nil {
				selectedTxs = append(selectedTxs, tx)
				selectedCount++
				if selectedCount == numTransactions {
					break
				}
			}
		}

		//query block transactions
		for _, tx := range selectedTxs {
			stringTxnHash := tx.Hash().String()
			var txnReceipt *types.Receipt
			var errWhenGettingTxnReceipt error
			//handle API key issue
			for {
				// get the transaction receipt
				txnReceipt, errWhenGettingTxnReceipt = infuraClient.TransactionReceipt(context.Background(), tx.Hash())
				if errWhenGettingTxnReceipt != nil {
					//rotate API key
					apiKeyIndex = (apiKeyIndex + 1) % len(infuraApiKeys)
					infuraClient, errWhenGetingInfuraClient = CreateInfuraClient(infuraApiKeys[apiKeyIndex])
					if errWhenGetingInfuraClient != nil {
						fmt.Println("Error when calling Infura client: ", errWhenGetingInfuraClient)
						fmt.Println("Loop sleeping for 28 hours......")
						time.Sleep(28 * time.Hour)
					}
				} else {
					break
				}
			}

			var block *types.Block
			var err error

			for {
				// get the timestamp from the block header
				block, err = infuraClient.BlockByHash(context.Background(), txnReceipt.BlockHash)
				if err != nil {
					//rotate API key
					apiKeyIndex = (apiKeyIndex + 1) % len(infuraApiKeys)
					infuraClient, errWhenGetingInfuraClient = CreateInfuraClient(infuraApiKeys[apiKeyIndex])
					if errWhenGetingInfuraClient != nil {
						fmt.Println("Error when calling Infura client: ", errWhenGetingInfuraClient)
						fmt.Println("Loop sleeping for 28 hours......")
						time.Sleep(28 * time.Hour)
					}
				} else {
					break
				}
			}

			blockTimestamp := block.Time()

			// convert the block timestamp to time.Time
			transactionTime := time.Unix(int64(blockTimestamp), 0).UTC()

			// format the timestamp as desired
			timestampFormat := "Jan-02-2006 03:04:05 PM UTC"
			transactionTimestamp := transactionTime.Format(timestampFormat)

			//check the status of the transaction
			if txnReceipt.Status == 1 {

				var transactionUrl string
				//call etherscan API to get the transaction details
				transactionUrl = CreateTransactionUrl(etherscanKeys[etherscanApiKeyIndex], stringTxnHash)

				var transactionRes *http.Response
				var errWhenGettingTransactionDetails error

				//handle API key issue by waiting 25 hours
				for {
					transactionRes, errWhenGettingTransactionDetails = http.Get(transactionUrl)
					if errWhenGettingTransactionDetails != nil {
						//rotate etherscan API key
						etherscanApiKeyIndex = (etherscanApiKeyIndex + 1) % len(etherscanKeys)
						transactionUrl = CreateTransactionUrl(etherscanKeys[etherscanApiKeyIndex], stringTxnHash)
						transactionRes, errWhenGettingTransactionDetails = http.Get(transactionUrl)
						if errWhenGettingTransactionDetails != nil {
							fmt.Println("Error when getting transaction from etherscan : " + errWhenGettingTransactionDetails.Error())
							fmt.Println("Loop sleeping for 28 hours...")
							time.Sleep(28 * time.Hour)
						}
					} else {
						break
					}
				}

				defer transactionRes.Body.Close()
				txnBody, errWhenReadingTxnBody := ioutil.ReadAll(transactionRes.Body)
				if errWhenReadingTxnBody != nil {
					fmt.Println("Error when reading transaction body : ", errWhenReadingTxnBody)
					continue
				}

				var transactionResponse TransactionResponse
				errWhenUnmarshallingTxnResponse := json.Unmarshal(txnBody, &transactionResponse)
				if errWhenUnmarshallingTxnResponse != nil {
					fmt.Println("Error when unmarshaling transaction body : ", errWhenUnmarshallingTxnResponse)
					continue
				}

				fmt.Println("Hash : ", stringTxnHash)
				fmt.Println("Timestamp : ", transactionTimestamp)
				fmt.Println("Gas Price (Gwei) : ", weiToGwei(hexToString(transactionResponse.Result.GasPrice)))

				data := []string{transactionTimestamp, weiToGwei(hexToString(transactionResponse.Result.GasPrice))}

				//stringData := `0x` + hex.EncodeToString(tx.Data())

				//write data to CSV
				errWhenWritingData := writer.Write(data)
				if errWhenWritingData != nil {
					fmt.Println("Error when writing data : ", errWhenWritingData)
					continue
				}
				writer.Flush()
				data = nil //manually release data
			} else {
				//skip
				continue
			}
		}
		writer.Flush()
		defer file.Close() //close file to release memeory

	}

	// signal that data collection has finished
	done <- true

}

func CreateInfuraClient(apiKey string) (*ethclient.Client, error) {
	client, err := ethclient.Dial("https://mainnet.infura.io/v3/" + apiKey)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func CreateBlockUrl(apiKey string, timeObj int64) string {
	return `https://api.etherscan.io/api?module=block&action=getblocknobytime&timestamp=` + strconv.FormatInt(timeObj, 10) + `&closest=before&apikey=` + apiKey
}

func CreateTransactionUrl(apiKey string, txn string) string {
	return `https://api.etherscan.io/api?module=proxy&action=eth_getTransactionByHash&txhash=` + txn + `&apikey=` + apiKey
}

// Function to generate random indices
func GenerateRandomIndices(max, count int) []int {
	indices := make([]int, count)
	for i := 0; i < count; i++ {
		indices[i] = rand.Intn(max)
	}
	return indices
}
