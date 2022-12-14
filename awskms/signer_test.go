package awskms

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/LampardNguyen234/evm-kms/common/erc20"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/ethereum/go-ethereum/ethclient"
	"math"
	"math/big"
	"testing"
	"time"
)

var (
	receiverAddr = common.HexToAddress("0x243e9517a24813a2d73e9a74cd2c1c699d0ff7a5")
	rpcHost      = "https://rpc-mumbai.maticvigil.com/"
	erc20Address = common.HexToAddress("0x2d7882beDcbfDDce29Ba99965dd3cdF7fcB10A1e")
	numTests     = 10
)

var c *AmazonKMSClient

func init() {
	ctx := context.Background()

	cfg, err := LoadStaticCredentialsConfigConfigFromFile("./config-static-credentials-example.json")
	c, err = NewAmazonKMSClientWithStaticCredentials(ctx, *cfg)
	if err != nil {
		panic(err)
	}
	c.WithChainID(new(big.Int).SetUint64(80001))
}

func TestAmazonKMSClient_GetPublicKey(t *testing.T) {
	pubKey, err := c.GetPublicKey()
	if err != nil {
		panic(err)
	}

	fmt.Printf("pubKey: %x\n", secp256k1.CompressPubkey(pubKey.X, pubKey.Y))
}

func TestAmazonKMSClient_GetAddress(t *testing.T) {
	address := c.GetAddress()

	fmt.Printf("address: %v\n", address)
}

func TestAmazonKMSClient_Sign(t *testing.T) {
	msg := []byte("Hello World")
	sig, err := c.SignHash(crypto.Keccak256Hash(msg))
	if err != nil {
		panic(err)
	}
	fmt.Printf("sig: %x\n", sig)
}

func waitForReceipt(evmClient *ethclient.Client, txHash common.Hash) (*types.Receipt, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timed-out")
		default:
			receipt, err := evmClient.TransactionReceipt(ctx, txHash)
			if err != nil {
				time.Sleep(10 * time.Second)
				continue
			}
			return receipt, nil
		}
	}
}

func TestSendETH(t *testing.T) {
	ctx := context.Background()
	evmClient, err := ethclient.Dial(rpcHost)
	if err != nil {
		panic(err)
	}

	for i := 0; i < numTests; i++ {
		fmt.Printf("========== TEST %v ==========\n", i)
		balance, err := evmClient.BalanceAt(ctx, c.GetAddress(), nil)
		if err != nil {
			panic(err)
		}
		fmt.Printf("currentBalance: %v\n", balance.Uint64())

		nonce, err := evmClient.PendingNonceAt(context.Background(), c.GetAddress())
		if err != nil {
			panic(err)
		}

		gasPrice, err := evmClient.SuggestGasPrice(ctx)
		if err != nil {
			panic(err)
		}
		gas := uint64(50000)

		testTx := types.NewTx(&types.LegacyTx{
			To:       &receiverAddr,
			Nonce:    nonce,
			GasPrice: gasPrice,
			Gas:      gas,
			Value:    big.NewInt(100),
			Data:     []byte{},
		})

		signedTx, err := c.GetDefaultEVMTransactor().Signer(c.GetAddress(), testTx)
		if err != nil {
			panic(err)
		}
		jsb, _ := json.Marshal(signedTx)
		fmt.Println("signedTx", string(jsb))

		err = evmClient.SendTransaction(ctx, signedTx)
		if err != nil {
			panic(err)
		}
		fmt.Printf("txHash: %v\n", signedTx.Hash())

		receipt, err := waitForReceipt(evmClient, signedTx.Hash())
		if err != nil {
			panic(err)
		}

		if receipt.Status != 1 {
			panic(fmt.Sprintf("tx %v FAILED", signedTx.Hash()))
		}

		fmt.Printf("========== FINISHED TEST %v ==========\n\n", i)
	}
}

func TestSendERC20(t *testing.T) {
	evmClient, err := ethclient.Dial(rpcHost)
	if err != nil {
		panic(err)
	}

	erc20Instance, err := erc20.NewErc20(erc20Address, evmClient)
	if err != nil {
		panic(err)
	}
	name, err := erc20Instance.Name(&bind.CallOpts{})
	if err != nil {
		panic(err)
	}
	symbol, err := erc20Instance.Symbol(&bind.CallOpts{})
	if err != nil {
		panic(err)
	}
	decimals, err := erc20Instance.Decimals(&bind.CallOpts{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("Token: %v, Symbol: %v, Decimals: %v, Address: %v\n",
		name, symbol, decimals, erc20Address.String())

	ctx := context.Background()
	myAddress := c.GetAddress()
	for i := 0; i < numTests; i++ {
		fmt.Printf("========== TEST %v ==========\n", i)
		balance, err := erc20Instance.BalanceOf(&bind.CallOpts{}, myAddress)
		if err != nil {
			panic(err)
		}
		fmt.Printf("currentBalance: %v\n", balance.String())

		nonce, err := evmClient.PendingNonceAt(context.Background(), c.GetAddress())
		if err != nil {
			panic(err)
		}

		gasPrice, err := evmClient.SuggestGasPrice(ctx)
		if err != nil {
			panic(err)
		}
		gas := uint64(100000)

		transactor := &bind.TransactOpts{
			From:      myAddress,
			Nonce:     new(big.Int).SetUint64(nonce),
			Signer:    c.GetEVMSignerFn(),
			GasPrice:  gasPrice,
			GasFeeCap: nil,
			GasTipCap: nil,
			GasLimit:  gas,
			Context:   ctx,
		}

		value, _ := rand.Int(rand.Reader, new(big.Int).SetUint64(
			uint64(math.Pow(10, float64(decimals.Uint64()/3)))))
		value = value.Add(value, new(big.Int).SetUint64(1))
		fmt.Printf("transferredValue: %v\n", value.String())

		tx, err := erc20Instance.Transfer(transactor, receiverAddr, value)
		if err != nil {
			panic(err)
		}
		jsb, _ := json.Marshal(tx)
		fmt.Printf("tx: %v\n", string(jsb))
		fmt.Printf("txHash: %x\n", tx.Hash())

		receipt, err := waitForReceipt(evmClient, tx.Hash())
		if err != nil {
			panic(err)
		}
		if receipt.Status != 1 {
			panic(fmt.Sprintf("tx %v FAILED", tx.Hash()))
		}

		fmt.Printf("========== FINISHED TEST %v ==========\n\n", i)
	}
}
