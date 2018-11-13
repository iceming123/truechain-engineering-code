// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package snailchain

import (
	"math/big"

	"github.com/truechain/truechain-engineering-code/common"
	"github.com/truechain/truechain-engineering-code/consensus"
	"github.com/truechain/truechain-engineering-code/core/types"
	"github.com/truechain/truechain-engineering-code/core/vm"
	"github.com/truechain/truechain-engineering-code/ethdb"
	"github.com/truechain/truechain-engineering-code/params"
	"time"
	"fmt"
)

// BlockGen creates blocks for testing.
// See GenerateChain for a detailed explanation.
type BlockGen struct {
	i           int
	parent      *types.SnailBlock
	chain       []*types.SnailBlock
	chainReader consensus.SnailChainReader
	header      *types.SnailHeader

	//gasPool *GasPool
	uncles  []*types.SnailHeader

	fruits []*types.SnailBlock
	signs  []*types.PbftSign

	config *params.ChainConfig
	engine consensus.Engine
}

// SetCoinbase sets the coinbase of the generated block.
// It can be called at most once.
func (b *BlockGen) SetCoinbase(addr common.Address) {
	if len(b.fruits) > 0 {
		panic("coinbase must be set before adding fruits")
	}
	b.header.Coinbase = addr
	//TODO not gaslimit 20180804
	//b.gasPool = new(GasPool).AddGas(b.header.GasLimit)
}

func (b *BlockGen) AddFruit(block *types.SnailBlock) {
	b.fruits = append(b.fruits, block)
}

// SetExtra sets the extra data field of the generated block.
func (b *BlockGen) SetExtra(data []byte) {
	b.header.Extra = data
}

// Number returns the block number of the block being generated.
func (b *BlockGen) Number() *big.Int {
	return new(big.Int).Set(b.header.Number)
}

// AddUncle adds an uncle header to the generated block.
func (b *BlockGen) AddUncle(h *types.SnailHeader) {
	b.uncles = append(b.uncles, h)
}

// PrevBlock returns a previously generated block by number. It panics if
// num is greater or equal to the number of the block being generated.
// For index -1, PrevBlock returns the parent block given to GenerateChain.
func (b *BlockGen) PrevBlock(index int) *types.SnailBlock {
	if index >= b.i {
		panic("block index out of range")
	}
	if index == -1 {
		return b.parent
	}
	return b.chain[index]
}

// OffsetTime modifies the time instance of a block, implicitly changing its
// associated difficulty. It's useful to test scenarios where forking is not
// tied to chain length directly.
func (b *BlockGen) OffsetTime(seconds int64) {
	b.header.Time.Add(b.header.Time, new(big.Int).SetInt64(seconds))
	if b.header.Time.Cmp(b.parent.Header().Time) <= 0 {
		panic("block time out of range")
	}

	b.header.Difficulty = b.engine.CalcSnailDifficulty(b.chainReader, b.header.Time.Uint64(), []*types.SnailHeader{b.parent.Header()})
}

// GenerateChain creates a chain of n blocks. The first block's
// parent will be the provided parent. db is used to store
// intermediate states and should contain the parent's state trie.
//
// The generator function is called with a new block generator for
// every block. Any transactions and uncles added to the generator
// become part of the block. If gen is nil, the blocks will be empty
// and their coinbase will be the zero address.
//
// Blocks created by GenerateChain do not contain valid proof of work
// values. Inserting them into BlockChain requires use of FakePow or
// a similar non-validating proof of work implementation.
func GenerateChain(config *params.ChainConfig, parent *types.SnailBlock, engine consensus.Engine, db ethdb.Database, n int, gen func(int, *BlockGen)) []*types.SnailBlock {
	if config == nil {
		config = params.TestChainConfig
	}
	blocks := make(types.SnailBlocks, n)
	genblock := func(i int, parent *types.SnailBlock) *types.SnailBlock {
		// TODO(karalabe): This is needed for clique, which depends on multiple blocks.
		// It's nonetheless ugly to spin up a blockchain here. Get rid of this somehow.
		blockchain, _ := NewSnailBlockChain(db, nil, config, engine, vm.Config{})
		defer blockchain.Stop()

		b := &BlockGen{i: i, parent: parent, chain: blocks, chainReader: blockchain, config: config, engine: engine}
		b.header = makeHeader(b.chainReader, parent, b.engine)

		// Execute any user modifications to the block and finalize it
		if gen != nil {
			gen(i, b)
		}

		if b.engine != nil {
			// TODO: add fruits support
			block, _ := b.engine.FinalizeSnail(b.chainReader, b.header, b.uncles, b.fruits, b.signs)

			return block
		}
		return nil
	}
	for i := 0; i < n; i++ {
		block := genblock(i, parent)
		blocks[i] = block
		parent = block
	}
	return blocks
}

func makeHeader(chain consensus.SnailChainReader, parent *types.SnailBlock, engine consensus.Engine) *types.SnailHeader {

	var time *big.Int
	if parent.Time() == nil {
		time = big.NewInt(10)
	} else {
		time = new(big.Int).Add(parent.Time(), big.NewInt(10)) // block time is fixed at 10 seconds
	}

	return &types.SnailHeader{
		ParentHash: parent.Hash(),
		Coinbase:   parent.Coinbase(),
		Difficulty: engine.CalcSnailDifficulty(chain, time.Uint64(), []*types.SnailHeader{&types.SnailHeader{
			Number:     parent.Number(),
			Time:       new(big.Int).Sub(time, big.NewInt(10)),
			Difficulty: parent.BlockDifficulty(),
			UncleHash:  parent.UncleHash(),
		}}),
		Number: new(big.Int).Add(parent.Number(), common.Big1),
		Time:   time,
	}
}

// makeHeaderChain creates a deterministic chain of headers rooted at parent.
func makeHeaderChain(parent *types.SnailHeader, n int, engine consensus.Engine, db ethdb.Database, seed int) []*types.SnailHeader {
	blocks := makeBlockChain(types.NewSnailBlockWithHeader(parent), n, engine, db, seed)
	headers := make([]*types.SnailHeader, len(blocks))
	for i, block := range blocks {
		headers[i] = block.Header()
	}
	return headers
}

// makeBlockChain creates a deterministic chain of blocks rooted at parent.
func makeBlockChain(parent *types.SnailBlock, n int, engine consensus.Engine, db ethdb.Database, seed int) []*types.SnailBlock {
	blocks := GenerateChain(params.TestChainConfig, parent, engine, db, n, func(i int, b *BlockGen) {
		b.SetCoinbase(common.Address{0: byte(seed), 19: byte(i)})
	})

	return blocks
}

func makeSnailBlockFruit(chain *SnailBlockChain,makeStartFastNum int,makeFruitSize int,config *params.ChainConfig,
	pubkey []byte,coinbaseAddr common.Address,signs []*types.PbftSign,isBlock bool) (*types.SnailBlock,error){

	var  fruitsetCopy []*types.SnailBlock


	if chain == nil{
		return  nil,fmt.Errorf("chain is nil")
	}

	// create head
	parent := chain.CurrentBlock()
	snailFruitsLastFastNumber := parent.Fruits()[len(parent.Fruits())-1].FastNumber()
	//parentNum := parent.Number()

	if isBlock{
		if makeFruitSize < params.MinimumFruits || snailFruitsLastFastNumber.Int64()>= int64(makeStartFastNum){
			return  nil,fmt.Errorf("fruitSet is nill or size less then 60")
		}
	}
	if makeFastNum.Cmp(snailFruitsLastFastNumber) <=0 {
		return  nil,fmt.Errorf("fast number less then current block's fruitset fast number")
	}


	makeHead := func(chain *SnailBlockChain,pubkey []byte,coinbaseAddr common.Address,fastNumber *big.Int,isFruit bool)(*types.SnailHeader) {
		parent := chain.CurrentBlock()
		num := parent.Number()
		tstamp = time.Now().Unix()
		header := &types.SnailHeader{
			ParentHash: parent.Hash(),
			Publickey:  pubkey,
			Number:     num.Add(num, common.Big1),
			Time:       big.NewInt(tstamp),
			Coinbase: coinbaseAddr,
			Fruit: isFruit,
			FastNumber:fastNumber,
		}

		pointerNum := new(big.Int).Sub(parent.Number(), pointerHashFresh)
		if pointerNum.Cmp(common.Big0) < 0 {
			pointerNum = new(big.Int).Set(common.Big0)
		}
		pointer := self.chain.GetBlockByNumber(pointerNum.Uint64())
		header.PointerHash = pointer.Hash()
		header.PointerNumber = pointer.Number()


		return header
	}

	makeFruit := func(chain *SnailBlockChain,fastNumber *big.Int,pubkey []byte,coinbaseAddr common.Address) (*types.SnailBlock,error){
		head := makeHead(chain,pubkey,coinbaseAddr,fastNumber,true)
		fruit :=  types.NewSnailBlock(
			head,
			nil,
			signs,
			nil,
		)
		return fruit,nil
	}

	// creat fruits
	if isBlock {
		for i = makeStartFastNum; i< makeStartFastNum+makeFruitSize; i++{
			fruit ,err:= makeFruit(chain,new(big.Int).SetInt64(i),pubkey,coinbaseAddr)
			if err != nil{
				return nil, err
			}
			fruitsetCopy = append(fruitsetCopy,fruit )
		}
		if len(fruitsetCopy) != makeFruitSize{
			return nil,fmt.Errorf("fruits make fail the length less then makeFruitSize")
		}

		block := types.NewSnailBlock(
			makeHead(chain,pubkey,coinbaseAddr,new(big.Int).SetInt64(int64(makeStartFastNum)),false),
			fruitsetCopy,
			signs,
			nil,
		)
		return  block,nil

	}else{
		fruit ,err:= makeFruit(chain,new(big.Int).SetInt64(int64(makeStartFastNum)),pubkey,coinbaseAddr)
		if err != nil{
			return nil, err
		}
		return fruit,nil
	}

	return nil,nil

}

