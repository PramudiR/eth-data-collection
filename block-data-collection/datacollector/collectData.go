package datacollector

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
)

type BlockNumberResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  string `json:"result"`
}

type TransactionResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  struct {
		BlockHash            string      `json:"blockHash"`
		BlockNumber          string      `json:"blockNumber"`
		From                 string      `json:"from"`
		Gas                  string      `json:"gas"`
		GasPrice             string      `json:"gasPrice"`
		MaxFeePerGas         string      `json:"maxFeePerGas"`
		MaxPriorityFeePerGas string      `json:"maxPriorityFeePerGas"`
		Hash                 string      `json:"hash"`
		Input                string      `json:"input"`
		Nonce                string      `json:"nonce"`
		To                   string      `json:"to"`
		TransactionIndex     string      `json:"transactionIndex"`
		Value                string      `json:"value"`
		Type                 string      `json:"type"`
		AccessList           interface{} `json:"accessList"`
		ChainId              string      `json:"chainId"`
		V                    string      `json:"v"`
		R                    string      `json:"r"`
		S                    string      `json:"s"`
	} `json:"result"`
}

func CollectData(startTime string, endTime string) {
	start, errWhenParsingTime := time.Parse(time.RFC3339, startTime)
	if errWhenParsingTime != nil {
		fmt.Println("An error occurred when parsing time : ", errWhenParsingTime)
		return
	}

	end, errWhenParsingTimeEnd := time.Parse(time.RFC3339, endTime)
	if errWhenParsingTimeEnd != nil {
		fmt.Println("An error occurred when parsing time : ", errWhenParsingTimeEnd)
		return
	}

	client, errDiallingClient := ethclient.Dial("https://mainnet.infura.io/v3/f7ad4e6f2bd54303b26fb0e0679752f8")
	if errDiallingClient != nil {
		fmt.Println("Error when calling ethereum client : ", errDiallingClient)
		return
	}

	//loop through the timestamps
	for i := start.Unix(); i < end.Unix(); i += 60 {
		timeObj := time.Unix(i, 0)

		//call the block number endpoint
		blockUrl := `https://api.etherscan.io/api?module=block&action=getblocknobytime&timestamp=` + strconv.FormatInt(timeObj.Unix(), 10) + `&closest=before&apikey=AER6M2C3436231IGT7SV7JZ2URFYFX7MZ1`

		var blockNoRes *http.Response
		var errWhenGettingBlockNumber error

		//Handle the etherscan API request limit issue by waiting for 25 hours
		for {
			blockNoRes, errWhenGettingBlockNumber = http.Get(blockUrl)
			if errWhenGettingBlockNumber != nil {
				fmt.Println("Error when calling block number endpoint : ", errWhenGettingBlockNumber)
				fmt.Println("Loop sleeping until API requests allowed...")
				time.Sleep(28 * time.Hour)
			} else {
				break
			}
		}

		defer blockNoRes.Body.Close()
		body, errWhenReadingTheBlockNo := ioutil.ReadAll(blockNoRes.Body)
		if errWhenReadingTheBlockNo != nil {
			fmt.Println("Error when reading the block number from response : ", errWhenReadingTheBlockNo)
			continue
		}

		var blockNoJsonResponse BlockNumberResponse
		errWhenUnMarshallingBlock := json.Unmarshal(body, &blockNoJsonResponse)
		if errWhenUnMarshallingBlock != nil {
			fmt.Println("Error when un marshalling : ", errWhenUnMarshallingBlock)
			continue
		}

		var i big.Int
		_, suc := i.SetString(blockNoJsonResponse.Result, 10)
		if !suc {
			fmt.Println("Convert failed")
			continue
		}

		var block *types.Block
		var errWhenLoadingBlock error

		//a do while approach where if an error occurs the loop will sleep for 25hours
		for {
			block, errWhenLoadingBlock = client.BlockByNumber(context.Background(), &i)
			if errWhenLoadingBlock != nil {
				//retry after 24 hours
				fmt.Println("Error when loading the block : ", errWhenLoadingBlock)
				fmt.Println("Loop sleeping until API requests allowed...")
				time.Sleep(28 * time.Hour)
			} else {
				break
			}
		}

		//CSV file initialization
		fileName := strconv.FormatInt(timeObj.Unix(), 10)
		file, errWhenCreatingCSV := os.Create(`output/` + fileName + `.csv`)
		if errWhenCreatingCSV != nil {
			fmt.Println("Error when creating the CSV file : ", errWhenCreatingCSV)
			continue
		}
		defer file.Close()

		//write headers into CSV file
		headers := []string{"Transaction Hash", "Timestamp", "From", "To", "Value(Eth)", "Type", "Transaction Fee(Eth)", "Gas Price(Gwei)", "Gas Limit", "Block", "Data array length"}
		writer := csv.NewWriter(bufio.NewWriter(file))
		errWhenWritingHeadersToCsv := writer.Write(headers)
		if errWhenWritingHeadersToCsv != nil {
			fmt.Println("Error when writing to the headers CSV file : ", errWhenWritingHeadersToCsv)
			continue
		}

		headers = nil

		//query block transactions
		for _, tx := range block.Transactions() {
			//pick only the normal transaction by checking if the "To" is nil
			if tx.To() != nil {
				//get the transaction hash
				transactionHash := tx.Hash().String()

				txHash := common.HexToHash(tx.Hash().String())

				var txnReceipt *types.Receipt
				var errWhenGettingTxnReceipt error

				//handle API key issue
				for {
					//check the transaction status
					txnReceipt, errWhenGettingTxnReceipt = client.TransactionReceipt(context.Background(), txHash)
					if errWhenGettingTxnReceipt != nil {
						fmt.Println("Error when getting transaction receipt : ", errWhenGettingTxnReceipt)
						fmt.Println("Loop sleeping until API requests allowed...")
						time.Sleep(28 * time.Hour)
					} else {
						break
					}
				}

				//check the status of the transaction
				if txnReceipt.Status == 1 {
					//get data
					gasUsed := txnReceipt.GasUsed

					//call etherscan API to get the transaction details
					transactionUrl := `https://api.etherscan.io/api?module=proxy&action=eth_getTransactionByHash&txhash=` + transactionHash + `&apikey=AER6M2C3436231IGT7SV7JZ2URFYFX7MZ1`

					var transactionRes *http.Response
					var errWhenGettingTransactionDetails error

					//handle API key issue by waiting 25 hours
					for {
						transactionRes, errWhenGettingTransactionDetails = http.Get(transactionUrl)
						if errWhenGettingTransactionDetails != nil {
							fmt.Println("Error when calling transaction getting endpoint : ", errWhenGettingBlockNumber)
							fmt.Println("Loop sleeping until API requests allowed...")
							time.Sleep(28 * time.Hour)
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

					data := []string{tx.Hash().String(), timeObj.Format("2006-01-02 15:04:05 MST"), transactionResponse.Result.From, transactionResponse.Result.To, weiToEth(hexToString(transactionResponse.Result.Value)),
						getTheTransactionType(hexToString(transactionResponse.Result.Type)), calculateTransactionFee(strconv.FormatUint(gasUsed, 10), hexToString(transactionResponse.Result.GasPrice)),
						weiToGwei(hexToString(transactionResponse.Result.GasPrice)), strconv.FormatUint(gasUsed, 10), hexToString(transactionResponse.Result.BlockNumber), strconv.Itoa(len(tx.Data()))}

					//stringData := `0x` + hex.EncodeToString(tx.Data())

					//write data to CSV
					errWhenWritingData := writer.Write(data)
					if errWhenWritingData != nil {
						fmt.Println("Error when writing data : ", errWhenWritingData)
						continue
					}

					data = nil //manually release data
				} else {
					//skip
					continue
				}
			}
		}
		writer.Flush()
		defer file.Close() //close file to release memeory

	}
}

func hexToString(hex string) string {
	if hex == "" {
		return ""
	} else {
		decimalValue, err := strconv.ParseInt(hex, 0, 64)
		if err != nil {
			return ""
		}
		stringValue := fmt.Sprintf("%d", decimalValue)
		return stringValue
	}
}

func getTheTransactionType(number string) string {
	switch number {
	case "0":
		return "Value Transfer"
	case "1":
		return "Contract Creation"
	case "2":
		return "EIP-1559"
	case "3":
		return "Contract Call"
	case "4":
		return "Delegate Call"
	case "5":
		return "Create2"
	case "6":
		return "Self Destruct"
	default:
		return "Unknown"
	}
}

func calculateTransactionFee(gasLimit string, gasPrice string) string {
	gl, ok := new(big.Int).SetString(gasLimit, 10)
	if !ok {
		return ""
	}
	gp, ok := new(big.Int).SetString(gasPrice, 10)
	if !ok {
		return ""
	}

	fee := new(big.Int).Mul(gl, gp) //this is in wei

	ether := new(big.Float).Quo(new(big.Float).SetInt(fee), big.NewFloat(params.Ether))

	return ether.String()
}

func weiToEth(weiStr string) string {
	wei, success := new(big.Int).SetString(weiStr, 10)
	if !success {
		return ""
	}

	eth := new(big.Float).Quo(new(big.Float).SetInt(wei), new(big.Float).SetFloat64(1e18))
	return eth.Text('f', 18)
}

func weiToGwei(weiStr string) string {
	wei, success := new(big.Int).SetString(weiStr, 10)
	if !success {
		return ""
	}

	gwei := new(big.Float).Quo(new(big.Float).SetInt(wei), new(big.Float).SetFloat64(1e9))
	return gwei.Text('f', 9)
}
