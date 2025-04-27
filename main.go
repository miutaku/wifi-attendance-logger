package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	_ "modernc.org/sqlite"
)

var Version = "dev"

type Config struct {
	Entries []struct {
		SSID  string `yaml:"ssid"`
		Place string `yaml:"place"`
	} `yaml:"entries"`
	OnAttendanceCommands []string `yaml:"on_attendance_commands"`
}

type Attendance struct {
	ID    uint   `gorm:"primaryKey"`
	Date  string `gorm:"index:idx_date_place,unique"`
	Place string `gorm:"index:idx_date_place,unique"`
}

const (
	configPath = "config.yaml"
	dbPath     = "attendance.db"
)

func loadConfig() (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	return &cfg, err
}

func getCurrentSSID() (string, error) {
	switch os := runtime.GOOS; os {
	case "linux":
		out, err := exec.Command("iwgetid", "-r").Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(out)), nil
	case "darwin":
		out, err := exec.Command("sh", "-c", "ipconfig getsummary en0 | awk -F ' SSID : ' '/ SSID : / {print $2}'").Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(out)), nil
	case "windows":
		out, err := exec.Command("netsh", "wlan", "show", "interfaces").Output()
		if err != nil {
			return "", err
		}
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.Contains(line, "SSID") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					return strings.TrimSpace(parts[1]), nil
				}
			}
		}
		return "", fmt.Errorf("SSID not found")
	default:
		return "", fmt.Errorf("unsupported OS: %s", os)
	}
}

func initDB() (*gorm.DB, error) {
	sqlDB, err := sql.Open("sqlite", dbPath) // <- modernc driver名は"sqlite"
	if err != nil {
		return nil, err
	}

	db, err := gorm.Open(
		gorm.Dialector(&gorm.Config{
			ConnPool: sqlDB,
			Logger:   logger.Default.LogMode(logger.Silent),
		}),
	)
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&Attendance{})
	return db, err
}

func insertAttendance(db *gorm.DB, place string) (bool, error) {
	today := time.Now().Format("2006-01-02")

	var existing Attendance
	err := db.Where("date = ? AND place = ?", today, place).First(&existing).Error
	if err == nil {
		return false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}

	attendance := Attendance{Date: today, Place: place}
	createResult := db.Create(&attendance)
	if createResult.Error != nil {
		return false, createResult.Error
	}
	return true, nil
}

func runAttendanceCommands(commands []string) {
	for _, command := range commands {
		if command == "" {
			continue
		}
		parts := strings.Fields(command)
		if len(parts) == 0 {
			continue
		}
		cmd := exec.Command(parts[0], parts[1:]...)
		err := cmd.Start()
		if err != nil {
			log.Println("Failed to start attendance command:", err)
		} else {
			log.Println("Attendance command executed:", command)
		}
	}
}

func checkThisMonth(db *gorm.DB) error {
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	nextMonth := monthStart.AddDate(0, 1, 0)

	var records []Attendance
	tx := db.Where("date >= ? AND date < ?", monthStart.Format("2006-01-02"), nextMonth.Format("2006-01-02")).
		Order("date").Find(&records)

	if tx.Error != nil {
		return tx.Error
	}

	fmt.Println("[今月の出社ログ]")
	for _, r := range records {
		fmt.Printf("%s - %s\n", r.Date, r.Place)
	}
	fmt.Printf("\n出社日数合計: %d 日\n", len(records))
	return nil
}

func main() {
	checkFlag := flag.Bool("check", false, "今月の出社記録を確認する")
	versionFlag := flag.Bool("version", false, "バージョンを表示する")
	flag.Parse()

	if *versionFlag {
		fmt.Println("wifi-attendance-logger version:", Version)
		return
	}

	db, err := initDB()
	if err != nil {
		log.Fatal("Failed to init DB: ", err)
	}

	if *checkFlag {
		err := checkThisMonth(db)
		if err != nil {
			log.Fatal("Check error: ", err)
		}
		return
	}

	cfg, err := loadConfig()
	if err != nil {
		log.Fatal("Failed to load config: ", err)
	}

	ssid, err := getCurrentSSID()
	if err != nil {
		log.Println("Could not get SSID: ", err)
		return
	}

	for _, entry := range cfg.Entries {
		if entry.SSID == ssid {
			success, err := insertAttendance(db, entry.Place)
			if err != nil {
				log.Println("Insert error: ", err)
			} else if success {
				log.Println("Attendance recorded for", entry.Place)
				runAttendanceCommands(cfg.OnAttendanceCommands)
			} else {
				log.Println("Already recorded for today")
			}
			break
		}
	}
}
