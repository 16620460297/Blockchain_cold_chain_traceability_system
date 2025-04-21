package service

import (
	"back_Blockchain_cold_chain_traceability_system/configs"
	"crypto/sha256"
	"encoding/hex"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

// BlockchainService 实现区块链相关功能
type BlockchainService struct{}

// 生成哈希值
func generateHash(data string, previousHash string, timestamp time.Time) string {
	h := sha256.New()
	h.Write([]byte(data + previousHash + timestamp.String()))
	return hex.EncodeToString(h.Sum(nil))
}

// AddToBlockchain 添加记录到区块链
func (s *BlockchainService) AddToBlockchain(productSKU string, recordType int, data string) (string, error) {
	// 获取最新的区块哈希作为前一个哈希
	var lastBlock configs.BlockchainLog
	var previousHash string
	var blockHeight int64 = 1 // 默认是第一个区块

	result := configs.DB.Where("product_sku = ?", productSKU).
		Order("created_at DESC").
		First(&lastBlock)

	if result.Error == nil {
		previousHash = lastBlock.Hash
		blockHeight = lastBlock.BlockHeight + 1
	}

	// 生成当前区块哈希
	timestamp := time.Now()
	hash := generateHash(data, previousHash, timestamp)

	// 创建区块记录
	block := configs.BlockchainLog{
		ProductSKU:   productSKU,
		RecordType:   recordType,
		RecordData:   data,
		Hash:         hash,
		PreviousHash: previousHash,
		BlockHeight:  blockHeight,
	}

	result = configs.DB.Create(&block)
	if result.Error != nil {
		return "", result.Error
	}

	return hash, nil
}

// VerifyBlockchain 验证区块链完整性
func (s *BlockchainService) VerifyBlockchain(c *gin.Context) {
	productSKU := c.Query("sku")
	if productSKU == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请提供产品SKU",
		})
		return
	}

	// 查询所有区块
	var blocks []configs.BlockchainLog
	result := configs.DB.Where("product_sku = ?", productSKU).
		Order("created_at").
		Find(&blocks)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "查询区块链记录失败",
		})
		return
	}

	if len(blocks) == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "未找到区块链记录",
		})
		return
	}

	// 验证每个区块
	valid := true
	invalidBlock := 0
	var previousHash string

	for i, block := range blocks {
		if i == 0 {
			// 第一个区块，检查前置哈希是否为空
			if block.PreviousHash != "" {
				valid = false
				invalidBlock = i + 1
				break
			}
		} else {
			// 检查前置哈希是否匹配
			if block.PreviousHash != previousHash {
				valid = false
				invalidBlock = i + 1
				break
			}
		}

		// 检查哈希值是否正确
		calculatedHash := generateHash(block.RecordData, block.PreviousHash, block.CreatedAt)
		if calculatedHash != block.Hash {
			valid = false
			invalidBlock = i + 1
			break
		}

		previousHash = block.Hash
	}

	if valid {
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "区块链验证通过",
			"data": gin.H{
				"valid":        true,
				"total_blocks": len(blocks),
			},
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "区块链验证失败",
			"data": gin.H{
				"valid":         false,
				"invalid_block": invalidBlock,
				"total_blocks":  len(blocks),
				// 续上段代码
			},
		})
	}
}

// GetBlockchainData 获取区块链数据
func (s *BlockchainService) GetBlockchainData(c *gin.Context) {
	productSKU := c.Query("sku")
	if productSKU == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请提供产品SKU",
		})
		return
	}

	// 查询所有区块
	var blocks []configs.BlockchainLog
	result := configs.DB.Where("product_sku = ?", productSKU).
		Order("created_at ASC").
		Find(&blocks)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "查询区块链记录失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取区块链数据成功",
		"data":    blocks,
	})
}

// SetupBlockchainRoutes 设置区块链服务路由
func SetupBlockchainRoutes(router *gin.Engine) {
	blockchainService := &BlockchainService{}

	// 公开接口
	publicGroup := router.Group("/api/blockchain")
	{
		publicGroup.GET("/verify", blockchainService.VerifyBlockchain)
		publicGroup.GET("/data", blockchainService.GetBlockchainData)
	}
}
