package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Word struct {
	ID         int    `gorm:"primaryKey" json:"id"`
	Word       string `json:"word"`
	Palindrome bool   `json:"palindrome"`
}

var DB *gorm.DB

func main() {
	// Load environment variables
	ConfigEnv := Environment()

	// Set up the database
	var err error
	DB, err = SetupDBSQL(ConfigEnv)
	if err != nil {
		log.Fatal(err.Error())
	}

	// Auto-migrate the Word model
	DB.AutoMigrate(&Word{})

	// Initialize the Gin router
	r := gin.Default()

	// API endpoints
	r.GET("/ispalindrome", func(c *gin.Context) {
		word := c.Query("word")
		c.JSON(http.StatusOK, gin.H{
			"message": IsPalindrome(word),
		})
	})

	r.POST("/savepalindrome", SavePalindromeHandler)
	r.GET("/words", GetWordsHandler)
	r.DELETE("/words/:id", DeleteWordHandler)

	r.Run() // listen and serve on 0.0.0.0:8080
}

func IsPalindrome(str string) bool {
	str = strings.ToLower(str)
	newStr := ""
	for _, c := range str {
		if unicode.IsLetter(c) {
			newStr += string(c)
		}
	}

	return newStr == ReverseStr(newStr)
}

func ReverseStr(str string) string {
	newString := ""

	for _, s := range str {
		newString = string(s) + newString
	}

	return newString
}

func SavePalindromeHandler(c *gin.Context) {
	word := c.Query("word")
	if word == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Word query parameter is required"})
		return
	}

	isPalindrome := IsPalindrome(word)

	// Save the word to the database
	newWord := Word{
		Word:       word,
		Palindrome: isPalindrome,
	}

	if err := DB.Create(&newWord).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save the word"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Word saved successfully",
		"word":       newWord.Word,
		"palindrome": newWord.Palindrome,
	})
}

func GetWordsHandler(c *gin.Context) {
	var words []Word
	if err := DB.Find(&words).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch words"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"words": words,
	})
}

func DeleteWordHandler(c *gin.Context) {
	id := c.Param("id")
	if err := DB.Delete(&Word{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete the word"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Word deleted successfully"})
}

func Environment() (config SchemaEnvironment) {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Error loading .env file")
		panic("Error loading .env file")
	}

	config.DB_HOST = os.Getenv("DB_HOST")
	config.DB_PORT = os.Getenv("DB_PORT")
	config.DB_USER = os.Getenv("DB_USER")
	config.DB_NAME = os.Getenv("DB_NAME")
	config.DB_PASS = os.Getenv("DB_PASS")
	config.DB_SSLMODE = os.Getenv("DB_SSLMODE")

	return config
}

type SchemaEnvironment struct {
	DB_USER    string
	DB_PASS    string
	DB_HOST    string
	DB_PORT    string
	DB_NAME    string
	DB_SSLMODE string
}

func SetupDBSQL(config SchemaEnvironment) (*gorm.DB, error) {
	CreateDB(config)

	dbHost := config.DB_HOST
	dbUsername := config.DB_USER
	dbPassword := config.DB_PASS
	dbName := config.DB_NAME
	dbPort := config.DB_PORT
	dbSSLMode := config.DB_SSLMODE
	timezone := "Asia/Jakarta"

	path := fmt.Sprintf("host=%v user=%v password=%v dbname=%v port=%v sslmode=%v TimeZone=%v",
		dbHost, dbUsername, dbPassword, dbName, dbPort, dbSSLMode, timezone)

	db, err := gorm.Open(postgres.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		defer logrus.Errorln("❌ Error Connect into Database Postgres", err.Error())
		return nil, err
	}

	postgreSQL, err := db.DB()
	postgreSQL.SetMaxOpenConns(10)
	postgreSQL.SetMaxIdleConns(5)
	postgreSQL.SetConnMaxLifetime(0)

	fmt.Println("💚 Connect into Database Postgres Success")
	return db, nil
}

func CreateDB(config SchemaEnvironment) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s port=%s sslmode=%s", config.DB_HOST, config.DB_USER, config.DB_PASS, config.DB_PORT, config.DB_SSLMODE)

	fmt.Println(dsn)
	// panic(1)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Println("failed to connect to the database: %v", err)
		return
	}

	createDBSQL := fmt.Sprintf("CREATE DATABASE %s;", config.DB_NAME)
	if err := db.Exec(createDBSQL).Error; err != nil {
		log.Println("failed to create database: %v", err)
		CloseDB(db)
	}
}

func CloseDB(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		log.Println("failed to get sql.DB from gorm.DB: %v", err)
	}

	if err := sqlDB.Close(); err != nil {
		log.Println("failed to close the database connection: %v", err)
	}
}
