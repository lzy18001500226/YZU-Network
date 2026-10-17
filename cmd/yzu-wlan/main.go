package main

import (
	"context"
	"errors"
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
		log.Printf("installed Windows tasks: %s and %s-Repeat (one hourly check)", win.TaskName, win.TaskName)
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

	log.Printf("YZU-WLAN auto login started, service=%q once=%v", cfg.Service, *once)
	consecutiveFailures := 0
	lastRecovery := time.Time{}
	for {
		campusInterface, err := runCycle(context.Background(), cfg)
		if err != nil {
			consecutiveFailures++
			log.Printf("cycle failed: %v", err)
			if campusInterface != "" && shouldRecoverNetwork(cfg, consecutiveFailures, lastRecovery) {
				lastRecovery = time.Now()
				recoverNetwork(cfg, consecutiveFailures, campusInterface)
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

func recoverNetwork(cfg config.Config, failures int, campusInterface string) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.RecoveryTimeout)
	defer cancel()

	if status, err := win.NetworkStatus(ctx); err != nil {
		log.Printf("network recovery after %d consecutive failures: collect status failed: %v", failures, err)
	} else {
		log.Printf("network recovery after %d consecutive failures: %s", failures, status)
	}

	if err := win.RenewDHCP(ctx, campusInterface); err != nil {
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

type campusPathClient interface {
	TestConnection(checkURL, checkHost string) bool
	GetPortalURL() (string, error)
	OldLoginService(portalURL, userID, password, service string) error
}

type campusPathDetector func(context.Context, []string, []string) ([]win.CampusInterface, error)
type campusPathClientFactory func(interfaceName string) (campusPathClient, error)

func runCycle(ctx context.Context, cfg config.Config) (string, error) {
	return runCycleWith(ctx, cfg, win.DetectCampusInterfaces, func(interfaceName string) (campusPathClient, error) {
		return campusnet.NewOnInterface(cfg.CheckTimeout, cfg.UserAgent, interfaceName)
	})
}

func runCycleWith(ctx context.Context, cfg config.Config, detect campusPathDetector, newClient campusPathClientFactory) (string, error) {
	campusInterfaces, err := detect(ctx, cfg.CampusCIDRs, cfg.CampusSSIDs)
	if err != nil {
		return "", err
	}
	if len(campusInterfaces) == 0 {
		return "", fmt.Errorf("no connected interface matches campus CIDRs %v or SSIDs %v; no campus login attempted", cfg.CampusCIDRs, cfg.CampusSSIDs)
	}

	type campusPath struct {
		path       win.CampusInterface
		client     campusPathClient
		needsLogin bool
	}
	clients := make([]campusPath, 0, len(campusInterfaces))
	var setupFailures []error
	for _, path := range campusInterfaces {
		client, err := newClient(path.Name)
		if err != nil {
			log.Printf("skip campus interface %q: %v", path.Name, err)
			setupFailures = append(setupFailures, fmt.Errorf("interface %q: %w", path.Name, err))
			continue
		}
		log.Printf("campus path detected: interface=%q IPv4=%s via=%s", path.Name, path.IPv4, path.Reason)
		clients = append(clients, campusPath{path: path, client: client})
	}
	if len(clients) == 0 {
		if len(setupFailures) > 0 {
			return "", errors.Join(setupFailures...)
		}
		return "", fmt.Errorf("no usable campus interface has an IPv4 address")
	}

	loginNeeded := false
	for i := range clients {
		if !cfg.Debug && clients[i].client.TestConnection(cfg.CheckURL, cfg.CheckHost) {
			log.Printf("internet is available through %q", clients[i].path.Name)
			continue
		}
		clients[i].needsLogin = true
		loginNeeded = true
	}
	if !loginNeeded {
		if len(setupFailures) > 0 {
			return "", errors.Join(setupFailures...)
		}
		return "", nil
	}

	log.Printf("internet unavailable on one or more campus paths, login starts after %s", cfg.StartDelay)
	pendingInterface := ""
	for _, candidate := range clients {
		if candidate.needsLogin {
			pendingInterface = candidate.path.Name
			break
		}
	}
	select {
	case <-time.After(cfg.StartDelay):
	case <-ctx.Done():
		return pendingInterface, ctx.Err()
	}

	var loginFailures []error
	recoveryInterface := ""
	for _, candidate := range clients {
		if !candidate.needsLogin {
			continue
		}
		if !cfg.Debug && candidate.client.TestConnection(cfg.CheckURL, cfg.CheckHost) {
			log.Printf("internet became available through %q before login", candidate.path.Name)
			continue
		}

		portalURL, err := candidate.client.GetPortalURL()
		if err != nil {
			log.Printf("campus portal unavailable through %q: %v", candidate.path.Name, err)
			loginFailures = append(loginFailures, fmt.Errorf("interface %q: get portal URL: %w", candidate.path.Name, err))
			if recoveryInterface == "" {
				recoveryInterface = candidate.path.Name
			}
			continue
		}
		if err := candidate.client.OldLoginService(portalURL, cfg.Username, cfg.Password, cfg.Service); err != nil {
			log.Printf("campusnet login failed through %q: %v", candidate.path.Name, err)
			loginFailures = append(loginFailures, fmt.Errorf("interface %q: campusnet login: %w", candidate.path.Name, err))
			if recoveryInterface == "" {
				recoveryInterface = candidate.path.Name
			}
			continue
		}
		if candidate.client.TestConnection(cfg.CheckURL, cfg.CheckHost) {
			log.Printf("login verified through %q", candidate.path.Name)
			continue
		}
		log.Printf("login through %q did not restore internet access", candidate.path.Name)
		loginFailures = append(loginFailures, fmt.Errorf("interface %q: login request succeeded but internet check still failed", candidate.path.Name))
		if recoveryInterface == "" {
			recoveryInterface = candidate.path.Name
		}
	}

	cycleFailures := append(setupFailures, loginFailures...)
	if len(cycleFailures) > 0 {
		return recoveryInterface, errors.Join(cycleFailures...)
	}
	return "", nil
}

func logoutDevices(cfg config.Config) error {
	campusInterfaces, err := win.DetectCampusInterfaces(context.Background(), cfg.CampusCIDRs, cfg.CampusSSIDs)
	if err != nil {
		return err
	}
	if len(campusInterfaces) == 0 {
		return fmt.Errorf("no connected campus interface detected")
	}
	self, err := campusnet.NewSelfServiceOnInterface(cfg.CheckTimeout, cfg.UserAgent, campusInterfaces[0].Name)
	if err != nil {
		return fmt.Errorf("select campus interface %q: %w", campusInterfaces[0].Name, err)
	}
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
