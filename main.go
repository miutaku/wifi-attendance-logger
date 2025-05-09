package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

var Version = "dev"

type Config struct {
	Entries []struct {
		SSID  string `yaml:"ssid"`
		Place string `yaml:"place"`
	} `yaml:"entries"`
	PostInsertCommands []string `yaml:"post_insert_command"`
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
		return nil, fmt.Errorf("failed to read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	for i, entry := range cfg.Entries {
		if strings.TrimSpace(entry.SSID) == "" {
			return nil, fmt.Errorf("invalid config: entry %d has empty SSID", i)
		}
		if strings.TrimSpace(entry.Place) == "" {
			return nil, fmt.Errorf("invalid config: entry %d has empty Place (フィールド名が正しいか確認してください)", i)
		}
	}

	for i, cmd := range cfg.PostInsertCommands {
		if strings.TrimSpace(cmd) == "" {
			return nil, fmt.Errorf("invalid config: post_insert_command entry %d is empty", i)
		}
	}

	return &cfg, nil
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
		out, err := exec.Command("sh", "-c",
			"ipconfig getsummary en0 | awk -F ' SSID : ' '/ SSID : / {print $2}'",
		).Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(out)), nil
	case "windows":
		out, err := exec.Command("netsh", "wlan", "show", "interfaces").Output()
		if err != nil {
			return "", err
		}
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, "SSID") {
				parts := strings.SplitN(line, ":", 2)
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
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&Attendance{}); err != nil {
		return nil, err
	}
	return db, nil
}

func insertAttendance(db *gorm.DB, place string) (bool, error) {
	today := time.Now().Format("2006-01-02")
	var a Attendance
	err := db.Where("date = ? AND place = ?", today, place).First(&a).Error
	if err == nil {
		return false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}
	if err := db.Create(&Attendance{Date: today, Place: place}).Error; err != nil {
		return false, err
	}
	return true, nil
}

func runAttendanceCommands(commands []string) {
	for _, c := range commands {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		cmd := exec.Command("sh", "-c", c)
		err := cmd.Start()
		if err != nil {
			log.Println("Failed to start attendance command:", err)
		} else {
			log.Println("Attendance command executed:", c)
		}
	}
}

func checkThisMonth(db *gorm.DB) error {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 1, 0)
	var recs []Attendance
	if err := db.Where("date >= ? AND date < ?",
		start.Format("2006-01-02"), end.Format("2006-01-02")).
		Order("date").Find(&recs).Error; err != nil {
		return err
	}
	fmt.Println("[今月の出社ログ]")
	for _, r := range recs {
		fmt.Printf("%s - %s\n", r.Date, r.Place)
	}
	fmt.Printf("\n出社日数合計: %d 日\n", len(recs))
	return nil
}

func main() {
	check := flag.Bool("check", false, "今月の出社記録を確認する")
	ver := flag.Bool("version", false, "バージョンを表示する")
	flag.Parse()
	if *ver {
		fmt.Println("wifi-attendance-logger version:", Version)
		return
	}
	db, err := initDB()
	if err != nil {
		log.Fatal("Failed to init DB:", err)
	}
	if *check {
		if err := checkThisMonth(db); err != nil {
			log.Fatal("Check error:", err)
		}
		return
	}
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}
	ssid, err := getCurrentSSID()
	if err != nil {
		log.Println("Could not get SSID:", err)
		return
	}
	for _, e := range cfg.Entries {
		if e.SSID == ssid {
			if ok, err := insertAttendance(db, e.Place); err != nil {
				log.Println("Insert error:", err)
			} else if ok {
				log.Println("Attendance recorded for", e.Place)
				runAttendanceCommands(cfg.PostInsertCommands)
			} else {
				log.Println("Already recorded for today")
			}
			break
		}
	}
}
