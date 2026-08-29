package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"yzu-wlan/internal/campusnet"
	"yzu-wlan/internal/config"
	win "yzu-wlan/internal/windows"
)

func main() {
	envPath := flag.String("e", ".env", "path to env file")
	once := flag.Bool("once", false, "run one check-login cycle and exit")
	logout := flag.Bool("logout", false, "logout online devices and exit")
	install := flag.Bool("install", false, "install Windows startup tasks")
	uninstall := flag.Bool("uninstall", false, "uninstall Windows startup tasks")
	flag.Parse()

	if *install {
		exe, err := os.Executable()
		if err != nil {
			log.Fatalf("get executable path: %v", err)
		}
		absEnv, _ := filepath.Abs(*envPath)
		if err := win.InstallStartupTask(exe, absEnv); err != nil {
			log.Fatalf("install startup task: %v", err)
		}
		if err := win.InstallRepeatTask(exe, absEnv); err != nil {
			log.Fatalf("install repeat task: %v", err)
		}
		log.Printf("installed Windows tasks: %s and %s-Repeat", win.TaskName, win.TaskName)
		return
	}

	if *uninstall {
		if err := win.UninstallTasks(); err != nil {
			log.Fatalf("uninstall tasks: %v", err)
		}
		log.Printf("uninstalled Windows tasks")
		return
	}

	cfg, err := config.Load(*envPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	setupLogging(cfg.LogFile)

	if *logout {
		if err := logoutDevices(cfg); err != nil {
			log.Fatalf("logout: %v", err)
		}
		return
	}

	log.Printf("YZU-WLAN auto login started, ssid=%q service=%q once=%v", cfg.TargetSSID, cfg.Service, *once)
	consecutiveFailures := 0
	lastRecovery := time.Time{}
	for {
		err := runCycle(context.Background(), cfg)
		if err != nil {
			consecutiveFailures++
			log.Printf("cycle failed: %v", err)
			if shouldRecoverNetwork(cfg, consecutiveFailures, lastRecovery) {
				lastRecovery = time.Now()
				recoverNetwork(cfg, consecutiveFailures)
			}
		} else {
			consecutiveFailures = 0
		}
		if *once {
			return
		}
		time.Sleep(cfg.CheckInterval)
	}
}

func shouldRecoverNetwork(cfg config.Config, failures int, lastRecovery time.Time) bool {
	if failures < cfg.RecoveryFailures {
		return false
	}
	return lastRecovery.IsZero() || time.Since(lastRecovery) >= cfg.RecoveryCooldown
}

func recoverNetwork(cfg config.Config, failures int) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.RecoveryTimeout)
	defer cancel()

	if status, err := win.NetworkStatus(ctx); err != nil {
		log.Printf("network recovery after %d consecutive failures: collect status failed: %v", failures, err)
	} else {
		log.Printf("network recovery after %d consecutive failures: %s", failures, status)
	}

	if err := win.RenewDHCP(ctx); err != nil {
		log.Printf("network recovery: DHCP renew failed: %v", err)
		return
	}
	log.Printf("network recovery: DHCP renew completed")
}

func setupLogging(path string) {
	if path == "" {
		return
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			log.Printf("create log directory failed: %v", err)
			return
		}
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		log.Printf("open log file failed: %v", err)
		return
	}
	log.SetOutput(io.MultiWriter(os.Stdout, file))
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
}

func runCycle(ctx context.Context, cfg config.Config) error {
	if !cfg.DisableWiFiStep {
		wifiCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.WiFiRetries+1)*cfg.WiFiRetryDelay+10*time.Second)
		ok, current, err := win.WaitForWiFi(wifiCtx, cfg.TargetSSID, cfg.WiFiRetries, cfg.WiFiRetryDelay)
		cancel()
		if err != nil {
			return fmt.Errorf("check WiFi: %w", err)
		}
		if !ok {
			return fmt.Errorf("target WiFi not connected: current=%q target=%q", current, cfg.TargetSSID)
		}
	}

	client := campusnet.New(cfg.CheckTimeout, cfg.UserAgent)
	if !cfg.Debug && client.TestConnection(cfg.CheckURL, cfg.CheckHost) {
		log.Printf("internet is available")
		return nil
	}

	log.Printf("internet unavailable, login starts after %s", cfg.StartDelay)
	select {
	case <-time.After(cfg.StartDelay):
	case <-ctx.Done():
		return ctx.Err()
	}

	if err := client.Login(cfg.Username, cfg.Password, cfg.Service); err != nil {
		time.Sleep(cfg.RetryInterval)
		return fmt.Errorf("campusnet login: %w", err)
	}
	if client.TestConnection(cfg.CheckURL, cfg.CheckHost) {
		log.Printf("login verified")
		return nil
	}
	return fmt.Errorf("login request succeeded but internet check still failed")
}

func logoutDevices(cfg config.Config) error {
	self := campusnet.NewSelfService(cfg.CheckTimeout, cfg.UserAgent)
	if err := self.Login(cfg.Username, cfg.Password); err != nil {
		return err
	}
	ips, err := self.GetOnlines()
	if err != nil {
		return err
	}
	for _, ip := range ips {
		if err := self.Logout(cfg.Username, ip); err != nil {
			return err
		}
		log.Printf("logged out %s", ip)
	}
	return nil
}
