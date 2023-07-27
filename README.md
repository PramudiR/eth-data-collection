# Ethereum Gas Price Extractor
The Ethereum Gas Price Extractor is a Go program designed to extract a sample of gas price values from successful Ethereum transactions for every block using the Etherscan and Infura APIs. Gas price values are an essential aspect of Ethereum transactions as they determine the fee paid to miners for processing transactions on the network. This tool provides valuable insights into the varying gas prices within the Ethereum blockchain.

## Prerequisites
To run this program, ensure you have the following installed:
- Go programming language (version 1.14 or higher)
- Three or more [Etherscan](https://etherscan.io/) API keys to access their API services and run the program smoothly.
- Three or more [Infura](https://www.infura.io/) API keys to access their Ethereum node API continuously.

## Installation
1. Clone this repository to your local machine:
```
  git clone https://github.com/dileepaj/eth-data-collection.git
  cd block-data-collection
```
2. Install the required Go packages:
```
  go mod download
```

## Usage
1. Make sure you have obtained the API keys from Etherscan and Infura.
2. Create .env file with **INFURA_API_KEYS=** your Infura API keys only separated by commas and **ETHERSCAN_KEYS=** your Ethersacn API keys only separated by commas.
3. Run the Ethereum Gas Price Extractor program:
```
  go run main.go
```
The program will connect to the Infura Ethereum node and use the Etherscan API to start extracting gas price samples from successful transactions. 

## Output
The extracted gas price values will be stored in separate CSV files, one file for each block, in the **'block-data-collection'** The files, will be named using the block number, for example: **'1345678.csv'**, **'1345679.csv'**, etc.
### Sample Output (block-data-collection/1345680.csv)
| Timestamp     | Gas Price (Gwei) |
|---------------|------------------|
| Jun-11-27 ... | 100              |
| Jun-11-27 ... | 95               |
| Jun-11-27 ... | 110              |
| Jun-11-27 ... | 105              |
| ...           | ...              |

## Configuration
You can modify **' gasDataCollector.go'** file to customize the behavior of the Ethereum Gas Price Extractor.

## Contributing
Contributions are welcome! If you find any bugs or have ideas for improvements, feel free to open an issue or submit a pull request.

## License
This project is licensed under the [GNU General Public License (GPL)](https://www.gnu.org/licenses/gpl-3.0.en.html). Feel free to modify and use the codebase according to your requirements.
